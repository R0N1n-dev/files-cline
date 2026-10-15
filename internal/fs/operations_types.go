package fs

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Operation kinds.
const (
	OpCopy    = "copy"
	OpMove    = "move"
	OpTrash   = "trash"
	OpExtract = "extract"
	OpDelete  = "delete"
)

// Operation states.
const (
	StateRunning    = "running"
	StatePaused     = "paused"
	StateCancelling = "cancelling"
	StateCancelled  = "cancelled"
	StateCompleted  = "completed"
	StateError      = "error"
)

// Conflict resolution policies.
const (
	ConflictAsk       = "ask"
	ConflictOverwrite = "overwrite"
	ConflictSkip      = "skip"
	ConflictRename    = "rename"
)

// ErrCancelled is returned when an operation is cancelled mid-flight.
var ErrCancelled = errors.New("operation cancelled by user")

// UndoEntry records a reversible filesystem action.
type UndoEntry struct {
	CreatedPath string `json:"createdPath,omitempty"`
	MovedFrom   string `json:"movedFrom,omitempty"`
	MovedTo     string `json:"movedTo,omitempty"`
	TrashEntry  string `json:"trashEntry,omitempty"`
}

// Operation is a single cancellable filesystem job (copy, move, trash, extract).
type Operation struct {
	ID           string      `json:"id"`
	Kind         string      `json:"kind"`
	State        string      `json:"state"`
	Sources      []string    `json:"sources"`
	DestDir      string      `json:"destDir"`
	ConflictMode string      `json:"conflictMode"`
	TotalBytes   int64       `json:"totalBytes"`
	DoneBytes    int64       `json:"doneBytes"`
	Progress     float64     `json:"progress"`
	SpeedBps     float64     `json:"speedBps"`
	ETASeconds   float64     `json:"etaSeconds"`
	CurrentFile  string      `json:"currentFile"`
	ItemsDone    int         `json:"itemsDone"`
	ItemsTotal   int         `json:"itemsTotal"`
	Skipped      int         `json:"skipped"`
	Overwritten  int         `json:"overwritten"`
	Renamed      int         `json:"renamed"`
	StartedAt    time.Time   `json:"startedAt"`
	FinishedAt   time.Time   `json:"finishedAt"`
	Error        string      `json:"error,omitempty"`
	CanUndo      bool        `json:"canUndo"`
	Message      string      `json:"message,omitempty"`
	Undo         []UndoEntry `json:"-"`

	ctx       context.Context
	cancel    context.CancelFunc
	pauseCh   chan struct{}
	paused    bool
	lastEmit  time.Time
	lastBytes int64
	lastTime  time.Time
	mu        *sync.Mutex
}

// newOperation builds an Operation with its mutex ready for use.
func newOperation(kind string, sources []string, destDir, conflictMode string, totalBytes int64, items int) *Operation {
	ctx, cancel := context.WithCancel(context.Background())
	return &Operation{
		Kind:         kind,
		State:        StateRunning,
		Sources:      append([]string{}, sources...),
		DestDir:      destDir,
		ConflictMode: conflictMode,
		TotalBytes:   totalBytes,
		ItemsTotal:   items,
		StartedAt:    time.Now(),
		ctx:          ctx,
		cancel:       cancel,
		pauseCh:      make(chan struct{}),
		lastTime:     time.Now(),
		mu:           &sync.Mutex{},
	}
}
