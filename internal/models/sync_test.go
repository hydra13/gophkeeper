package models

import "testing"

func TestSyncConflictResolve_Local(t *testing.T) {
	c := &SyncConflict{
		UserID:         1,
		RecordID:       10,
		LocalRevision:  3,
		ServerRevision: 4,
		Resolved:       false,
	}
	if err := c.Resolve(ConflictResolutionLocal); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Resolved {
		t.Fatal("expected conflict to be resolved")
	}
	if c.Resolution != ConflictResolutionLocal {
		t.Fatalf("expected resolution 'local', got %s", c.Resolution)
	}
}

func TestSyncConflictResolve_Server(t *testing.T) {
	c := &SyncConflict{
		UserID:         1,
		RecordID:       10,
		LocalRevision:  3,
		ServerRevision: 4,
		Resolved:       false,
	}
	if err := c.Resolve(ConflictResolutionServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Resolved {
		t.Fatal("expected conflict to be resolved")
	}
	if c.Resolution != ConflictResolutionServer {
		t.Fatalf("expected resolution 'server', got %s", c.Resolution)
	}
}

func TestSyncConflictResolve_InvalidResolution(t *testing.T) {
	c := &SyncConflict{
		UserID:         1,
		RecordID:       10,
		LocalRevision:  3,
		ServerRevision: 4,
		Resolved:       false,
	}
	if err := c.Resolve("invalid"); err != ErrInvalidConflictResolution {
		t.Fatalf("expected ErrInvalidConflictResolution, got: %v", err)
	}
	if c.Resolved {
		t.Fatal("expected conflict to remain unresolved")
	}
}

func TestSyncConflictResolve_AlreadyResolved(t *testing.T) {
	c := &SyncConflict{
		UserID:         1,
		RecordID:       10,
		LocalRevision:  3,
		ServerRevision: 4,
		Resolved:       false,
	}
	if err := c.Resolve(ConflictResolutionLocal); err != nil {
		t.Fatalf("first resolve: unexpected error: %v", err)
	}
	if err := c.Resolve(ConflictResolutionServer); err != ErrConflictAlreadyResolved {
		t.Fatalf("expected ErrConflictAlreadyResolved on second resolve, got: %v", err)
	}
	if c.Resolution != ConflictResolutionLocal {
		t.Fatalf("expected resolution to remain 'local', got %s", c.Resolution)
	}
}
