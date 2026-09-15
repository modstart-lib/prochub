package process

import "testing"

func TestSetAutoStart(t *testing.T) {
	m := NewManager()
	m.Register(Definition{ID: "proc-1", Name: "test", Command: "echo", AutoStart: false})

	if err := m.SetAutoStart("proc-1", true); err != nil {
		t.Fatalf("SetAutoStart(true) failed: %v", err)
	}
	snap, err := m.Get("proc-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !snap.Definition.AutoStart {
		t.Error("expected autoStart to be true after enabling")
	}

	if err := m.SetAutoStart("proc-1", false); err != nil {
		t.Fatalf("SetAutoStart(false) failed: %v", err)
	}
	snap, err = m.Get("proc-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if snap.Definition.AutoStart {
		t.Error("expected autoStart to be false after disabling")
	}

	if err := m.SetAutoStart("unknown", true); err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown process, got %v", err)
	}
}

func TestRunningNames(t *testing.T) {
	m := NewManager()
	m.Register(Definition{ID: "proc-1", Name: "alpha", Command: "sleep"})
	m.Register(Definition{ID: "proc-2", Name: "beta", Command: "sleep"})

	if names := m.RunningNames(); len(names) != 0 {
		t.Fatalf("expected no running processes, got %v", names)
	}

	// Mark one entry as running directly to avoid spawning a real command.
	m.mu.Lock()
	m.entries["proc-1"].status = StatusRunning
	m.mu.Unlock()

	names := m.RunningNames()
	if len(names) != 1 || names[0] != "alpha" {
		t.Fatalf("expected [alpha], got %v", names)
	}
}
