package fs

import "time"

// waitIfPaused blocks while the operation is paused and aborts when cancelled.
func (o *Operation) waitIfPaused() error {
	o.mu.Lock()
	paused := o.paused
	ch := o.pauseCh
	o.mu.Unlock()
	if !paused {
		return nil
	}
	select {
	case <-ch:
		return nil
	case <-o.ctx.Done():
		return ErrCancelled
	}
}

// checkAbort returns ErrCancelled when the context is done, and honours pause state.
func (o *Operation) checkAbort() error {
	if err := o.waitIfPaused(); err != nil {
		return err
	}
	if o.ctx.Err() != nil {
		return ErrCancelled
	}
	return nil
}

// reportProgress updates counters and emits a throttled progress event (~12/s).
func (o *Operation) reportProgress(m *OperationsManager, current string, deltaBytes int64) {
	now := time.Now()
	o.mu.Lock()
	o.DoneBytes += deltaBytes
	if current != "" {
		o.CurrentFile = current
	}
	if o.TotalBytes > 0 {
		p := float64(o.DoneBytes) / float64(o.TotalBytes) * 100
		if p > 100 {
			p = 100
		}
		o.Progress = p
	}
	elapsed := now.Sub(o.lastTime).Seconds()
	if elapsed >= 0.25 {
		o.SpeedBps = float64(o.DoneBytes-o.lastBytes) / elapsed
		o.lastBytes = o.DoneBytes
		o.lastTime = now
		if o.SpeedBps > 1 && o.TotalBytes > 0 {
			remaining := o.TotalBytes - o.DoneBytes
			if remaining < 0 {
				remaining = 0
			}
			o.ETASeconds = float64(remaining) / o.SpeedBps
		}
	}
	shouldEmit := now.Sub(o.lastEmit) >= 80*time.Millisecond
	if shouldEmit {
		o.lastEmit = now
	}
	o.mu.Unlock()

	if shouldEmit {
		m.emitEvent("op:progress", snapshot(o))
	}
}
