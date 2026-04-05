package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	require.Len(t, got, 2)
	require.Equal(t, int64(1), got[0].ID)
	require.Equal(t, int64(3), got[1].ID)
}
