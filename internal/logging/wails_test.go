package logging

import "testing"

func TestWailsLoggerBuffersUntilSink(t *testing.T) {
	logger := NewWailsLogger()
	logger.Info("early")
	logger.Error("boot")

	var got []string
	logger.SetSink(func(level, message string) {
		got = append(got, level+":"+message)
	})
	logger.Warning("late")

	want := []string{"info:early", "error:boot", "warning:late"}
	if len(got) != len(want) {
		t.Fatalf("expected %d messages, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("message %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestWailsLoggerBufferIsBounded(t *testing.T) {
	logger := NewWailsLogger()
	for i := 0; i < 500; i++ {
		logger.Debug("x")
	}
	if len(logger.pending) != 200 {
		t.Fatalf("pending buffer = %d, want 200", len(logger.pending))
	}
}
