package service

import "testing"

func TestWSLManagerReportsStartingWhileBusy(t *testing.T) {
	m := NewWSLManager()

	if m.Status().Starting {
		t.Fatal("new manager should not report starting")
	}

	m.busy = 1
	if !m.Status().Starting {
		t.Error("expected Starting while a start/restart is in progress")
	}

	m.busy = 0
	if m.Status().Starting {
		t.Error("expected Starting to clear once the operation finished")
	}
}
