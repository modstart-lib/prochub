package logging

import "sync"

// WailsLogger adapts the Wails framework logger to this project's log store, so
// framework-level messages (binding dispatch errors, recovered panics, frontend
// errors) share the same system log file as the rest of the application.
type WailsLogger struct {
	mu      sync.Mutex
	sink    func(level, message string)
	pending []wailsMessage
}

type wailsMessage struct {
	level   string
	message string
}

// NewWailsLogger creates a logger whose destination is installed later with
// SetSink, because the system log store only exists after the app has started.
func NewWailsLogger() *WailsLogger {
	return &WailsLogger{}
}

// SetSink installs the destination for log messages and flushes any message
// that was emitted before the destination became available (boot-time logs).
func (l *WailsLogger) SetSink(sink func(level, message string)) {
	l.mu.Lock()
	l.sink = sink
	pending := l.pending
	l.pending = nil
	l.mu.Unlock()

	for _, msg := range pending {
		sink(msg.level, msg.message)
	}
}

func (l *WailsLogger) write(level, message string) {
	l.mu.Lock()
	sink := l.sink
	if sink == nil {
		// Keep a bounded buffer of early messages; drop the oldest beyond it.
		if len(l.pending) < 200 {
			l.pending = append(l.pending, wailsMessage{level: level, message: message})
		}
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()
	sink(level, message)
}

// Print implements logger.Logger.
func (l *WailsLogger) Print(message string) { l.write("info", message) }

// Trace implements logger.Logger.
func (l *WailsLogger) Trace(message string) { l.write("trace", message) }

// Debug implements logger.Logger.
func (l *WailsLogger) Debug(message string) { l.write("debug", message) }

// Info implements logger.Logger.
func (l *WailsLogger) Info(message string) { l.write("info", message) }

// Warning implements logger.Logger.
func (l *WailsLogger) Warning(message string) { l.write("warning", message) }

// Error implements logger.Logger.
func (l *WailsLogger) Error(message string) { l.write("error", message) }

// Fatal implements logger.Logger.
func (l *WailsLogger) Fatal(message string) { l.write("fatal", message) }
