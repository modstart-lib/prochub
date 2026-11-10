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

func TestUpdateDefinitionKeepsRuntimeState(t *testing.T) {
	m := NewManager()
	m.Register(Definition{ID: "proc-1", Name: "test", Command: "old"})

	// Mark the entry as running to prove the update leaves state untouched.
	m.mu.Lock()
	m.entries["proc-1"].status = StatusRunning
	m.entries["proc-1"].pid = 4242
	m.mu.Unlock()

	next := Definition{ID: "other-id", Name: "test", Command: "new"}
	if err := m.UpdateDefinition("proc-1", next); err != nil {
		t.Fatalf("UpdateDefinition failed: %v", err)
	}

	snap, err := m.Get("proc-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if snap.Definition.Command != "new" {
		t.Errorf("definition not replaced: %+v", snap.Definition)
	}
	if snap.Definition.ID != "proc-1" {
		t.Errorf("update must keep the entry ID, got %q", snap.Definition.ID)
	}
	if snap.Status != StatusRunning || snap.PID != 4242 {
		t.Errorf("runtime state changed: status=%s pid=%d", snap.Status, snap.PID)
	}

	if err := m.UpdateDefinition("missing", next); err != ErrNotFound {
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
