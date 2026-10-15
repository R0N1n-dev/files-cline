//go:build headless

// Vela Files — headless preview/dev server. It exposes the exact same App API
// over HTTP JSON plus a Server-Sent Events stream, so the Vue frontend can be
// developed, tested, and demoed in a browser without GTK/WebKit2GTK installed.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

type eventHub struct {
	mu   sync.Mutex
	subs map[chan []byte]bool
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[chan []byte]bool)}
}

func (h *eventHub) publish(name string, payload any) {
	data, err := json.Marshal(map[string]any{"event": name, "data": payload})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- data:
		default:
		}
	}
}

func (h *eventHub) subscribe() chan []byte {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	h.subs[ch] = true
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

func main() {
	hub := newEventHub()
	app := NewApp(func(name string, data any) { hub.publish(name, data) })
	defer app.Close()

	addr := os.Getenv("VELA_ADDR")
	if addr == "" {
		addr = "127.0.0.1:34115"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ch := hub.subscribe()
		defer hub.unsubscribe(ch)
		for {
			select {
			case msg, open := <-ch:
				if !open {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		var args []json.RawMessage
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&args); err != nil && err.Error() != "EOF" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
		} else {
			query := r.URL.Query().Get("args")
			if query != "" {
				if err := json.Unmarshal([]byte(query), &args); err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
					return
				}
			}
		}
		result, err := callMethod(app, method, args)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": result})
	})

	staticDir := os.Getenv("VELA_STATIC_DIR")
	if staticDir == "" {
		staticDir = filepath.Join("frontend", "dist")
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Vela Files headless API is running. Frontend bundle not found at %s.\nRun: cd frontend && npm install && npm run build\n", staticDir)
		})
	}

	log.Printf("vela-files headless API on http://%s (events: /api/events)", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// callMethod dispatches /api/<ExportedMethod> onto the App via reflection.
func callMethod(app *App, method string, rawArgs []json.RawMessage) (any, error) {
	mv := reflect.ValueOf(app).MethodByName(method)
	if !mv.IsValid() {
		return nil, fmt.Errorf("unknown method %q", method)
	}
	mt := mv.Type()
	if len(rawArgs) != mt.NumIn() {
		return nil, fmt.Errorf("%s expects %d argument(s), got %d", method, mt.NumIn(), len(rawArgs))
	}
	args := make([]reflect.Value, mt.NumIn())
	for i := 0; i < mt.NumIn(); i++ {
		ptr := reflect.New(mt.In(i))
		if len(rawArgs[i]) > 0 {
			if err := json.Unmarshal(rawArgs[i], ptr.Interface()); err != nil {
				return nil, fmt.Errorf("argument %d of %s: %w", i+1, method, err)
			}
		}
		args[i] = ptr.Elem()
	}

	out := mv.Call(args)
	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		return out[0].Interface(), nil
	default:
		if errVal := out[len(out)-1]; !errVal.IsNil() {
			return out[0].Interface(), errVal.Interface().(error)
		}
		return out[0].Interface(), nil
	}
}
