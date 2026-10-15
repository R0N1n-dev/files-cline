package fs

import (
	"fmt"
	"sync"
	"time"
)

// OperationsManager tracks and controls all active and finished operations.
type OperationsManager struct {
	mu        sync.Mutex
	ops       map[string]*Operation
	order     []string
	resolvers map[string]chan string // opID+path -> conflict decision
	emit      EventEmitter
	seq       int64
}

// NewOperationsManager constructs an operations manager.
func NewOperationsManager(emit EventEmitter) *OperationsManager {
	return &OperationsManager{
		ops:       make(map[string]*Operation),
		resolvers: make(map[string]chan string),
		emit:      emit,
	}
}

func (m *OperationsManager) nextID(kind string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	return fmt.Sprintf("%s-%d-%d", kind, time.Now().Unix(), m.seq)
}

func (m *OperationsManager) emitEvent(name string, data any) {
	if m.emit != nil {
		m.emit(name, data)
	}
}

func (m *OperationsManager) register(op *Operation) {
	m.mu.Lock()
	m.ops[op.ID] = op
	m.order = append(m.order, op.ID)
	if len(m.order) > 60 {
		drop := m.order[0]
		m.order = m.order[1:]
		delete(m.ops, drop)
	}
	m.mu.Unlock()
}

// List returns active and recent operations, newest first.
func (m *OperationsManager) List() []Operation {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Operation, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		if op, ok := m.ops[m.order[i]]; ok {
			out = append(out, snapshot(op))
		}
	}
	return out
}

// Get returns one operation snapshot.
func (m *OperationsManager) Get(id string) (Operation, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.ops[id]
	if !ok {
		return Operation{}, false
	}
	return snapshot(op), true
}

func snapshot(op *Operation) Operation {
	op.mu.Lock()
	defer op.mu.Unlock()
	return Operation{
		ID: op.ID, Kind: op.Kind, State: op.State,
		Sources: op.Sources, DestDir: op.DestDir, ConflictMode: op.ConflictMode,
		TotalBytes: op.TotalBytes, DoneBytes: op.DoneBytes, Progress: op.Progress,
		SpeedBps: op.SpeedBps, ETASeconds: op.ETASeconds, CurrentFile: op.CurrentFile,
		ItemsDone: op.ItemsDone, ItemsTotal: op.ItemsTotal,
		Skipped: op.Skipped, Overwritten: op.Overwritten, Renamed: op.Renamed,
		StartedAt: op.StartedAt, FinishedAt: op.FinishedAt, Error: op.Error,
		CanUndo: op.CanUndo, Message: op.Message,
	}
}

// Cancel stops an operation mid-copy and removes its partial output.
func (m *OperationsManager) Cancel(id string) error {
	m.mu.Lock()
	op, ok := m.ops[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("operation %q not found", id)
	}
	op.mu.Lock()
	if op.State == StateCompleted || op.State == StateCancelled || op.State == StateError {
		op.mu.Unlock()
		return fmt.Errorf("operation %q already finished", id)
	}
	op.State = StateCancelled
	cancel := op.cancel
	op.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.emitEvent("op:state", snapshot(op))
	return nil
}

// Pause suspends an operation between chunks (resumes with Resume).
func (m *OperationsManager) Pause(id string) error {
	m.mu.Lock()
	op, ok := m.ops[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("operation %q not found", id)
	}
	op.mu.Lock()
	defer op.mu.Unlock()
	if op.State != StateRunning {
		return fmt.Errorf("cannot pause operation in state %q", op.State)
	}
	op.paused = true
	op.State = StatePaused
	return nil
}

// Resume resumes a paused operation.
func (m *OperationsManager) Resume(id string) error {
	m.mu.Lock()
	op, ok := m.ops[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("operation %q not found", id)
	}
	op.mu.Lock()
	defer op.mu.Unlock()
	if op.State != StatePaused {
		return fmt.Errorf("cannot resume operation in state %q", op.State)
	}
	op.paused = false
	op.State = StateRunning
	close(op.pauseCh)
	op.pauseCh = make(chan struct{})
	return nil
}

// ResolveConflict feeds a user decision to a blocked "ask" conflict prompt.
func (m *OperationsManager) ResolveConflict(opID, srcPath, decision string) {
	key := opID + "|" + srcPath
	m.mu.Lock()
	ch, ok := m.resolvers[key]
	m.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- decision:
	default:
	}
}

// CancelAll cancels every running or paused operation.
func (m *OperationsManager) CancelAll() {
	for _, op := range m.List() {
		if op.State == StateRunning || op.State == StatePaused {
			_ = m.Cancel(op.ID)
		}
	}
}

// ClearFinished removes completed/failed/cancelled operations from the list.
func (m *OperationsManager) ClearFinished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	keep := make([]string, 0, len(m.order))
	for _, id := range m.order {
		op, ok := m.ops[id]
		if !ok {
			continue
		}
		op.mu.Lock()
		done := op.State == StateCompleted || op.State == StateCancelled || op.State == StateError
		op.mu.Unlock()
		if done {
			delete(m.ops, id)
			continue
		}
		keep = append(keep, id)
	}
	m.order = keep
}
