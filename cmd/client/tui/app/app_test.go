package app

import (
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

func TestVisibleRecordsSkipsDeleted(t *testing.T) {
	now := time.Now()
	records := []models.Record{
		{ID: 1, Name: "active-1"},
		{ID: 2, Name: "deleted", DeletedAt: &now},
		{ID: 3, Name: "active-2"},
	}

	got := visibleRecords(records)
	if len(got) != 2 {
		t.Fatalf("visibleRecords() len = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("visibleRecords() IDs = [%d %d], want [1 3]", got[0].ID, got[1].ID)
	}
}

