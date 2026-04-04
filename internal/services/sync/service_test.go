package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
	syncmocks "github.com/hydra13/gophkeeper/internal/services/sync/mocks"
)

func newTestService(t *testing.T, sr *syncmocks.SyncRepoMock, rr *syncmocks.RecordRepoMock) *Service {
	t.Helper()

	svc, err := NewService(sr, rr)
	require.NoError(t, err)

	return svc
}

func textChange(userID int64, name, deviceID string, baseRev int64) models.PendingChange {
	return models.PendingChange{
		Record: &models.Record{
			UserID:     userID,
			Type:       models.RecordTypeText,
			Name:       name,
			DeviceID:   deviceID,
			KeyVersion: 1,
			Payload:    models.TextPayload{Content: "hello"},
		},
		BaseRevision: baseRev,
	}
}

func TestNewService_NilSyncRepo(t *testing.T) {
	mc := minimock.NewController(t)
	rr := syncmocks.NewRecordRepoMock(mc)

	_, err := NewService(nil, rr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sync repository is required")
}

func TestNewService_NilRecordRepo(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)

	_, err := NewService(sr, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record repository is required")
}

func TestPush_CreateChange(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	rr.CreateRecordMock.Set(func(record *models.Record) error {
		require.Equal(t, int64(1), record.UserID)
		require.Equal(t, "my note", record.Name)
		require.Equal(t, int64(1), record.Revision)
		require.Equal(t, "device-1", record.DeviceID)
		require.Equal(t, models.RecordTypeText, record.Type)
		record.ID = 42
		return nil
	})
	sr.GetMaxRevisionMock.Expect(1).Return(0, nil)
	sr.CreateRevisionMock.Inspect(func(rev *models.RecordRevision) {
		require.Equal(t, int64(42), rev.RecordID)
		require.Equal(t, int64(1), rev.UserID)
		require.Equal(t, int64(1), rev.Revision)
		require.Equal(t, "device-1", rev.DeviceID)
	}).Return(nil)

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		textChange(1, "my note", "device-1", 0),
	})

	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(42), accepted[0].RecordID)
	require.Equal(t, int64(1), accepted[0].Revision)
}

func TestPush_CreateChange_CreateRecordError(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	sr.GetMaxRevisionMock.Expect(1).Return(0, nil)
	rr.CreateRecordMock.Return(errors.New("boom"))

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		textChange(1, "my note", "device-1", 0),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "create record")
	require.Nil(t, accepted)
	require.Nil(t, conflicts)
}

func TestPush_UpdateChange_MatchingRevision(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	existing := &models.Record{
		ID:         11,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "original",
		Revision:   1,
		DeviceID:   "device-1",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "before"},
	}

	rr.GetRecordMock.Expect(11).Return(existing, nil)
	sr.GetMaxRevisionMock.Expect(1).Return(1, nil)
	rr.UpdateRecordMock.Inspect(func(record *models.Record) {
		require.Equal(t, int64(11), record.ID)
		require.Equal(t, "updated", record.Name)
		require.Equal(t, int64(2), record.Revision)
		require.Equal(t, "device-2", record.DeviceID)
		payload, ok := record.Payload.(models.TextPayload)
		require.True(t, ok)
		require.Equal(t, "after", payload.Content)
	}).Return(nil)
	sr.CreateRevisionMock.Inspect(func(rev *models.RecordRevision) {
		require.Equal(t, int64(11), rev.RecordID)
		require.Equal(t, int64(2), rev.Revision)
		require.Equal(t, "device-2", rev.DeviceID)
	}).Return(nil)

	accepted, conflicts, err := svc.Push(1, "device-2", []models.PendingChange{
		{
			Record: &models.Record{
				ID:         11,
				UserID:     1,
				Type:       models.RecordTypeText,
				Name:       "updated",
				DeviceID:   "device-2",
				KeyVersion: 2,
				Payload:    models.TextPayload{Content: "after"},
			},
			BaseRevision: 1,
		},
	})

	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(2), accepted[0].Revision)
}

func TestPush_UpdateConflict_NonMatchingRevision(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	existing := &models.Record{
		ID:         11,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "server",
		Revision:   3,
		DeviceID:   "device-2",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "server"},
	}

	rr.GetRecordMock.Expect(11).Return(existing, nil)
	sr.CreateConflictMock.Inspect(func(conflict *models.SyncConflict) {
		require.Equal(t, int64(11), conflict.RecordID)
		require.Equal(t, int64(2), conflict.LocalRevision)
		require.Equal(t, int64(3), conflict.ServerRevision)
		require.NotNil(t, conflict.LocalRecord)
		require.Equal(t, "local", conflict.LocalRecord.Name)
		require.Equal(t, existing, conflict.ServerRecord)
	}).Return(nil)

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		{
			Record: &models.Record{
				ID:         11,
				UserID:     1,
				Type:       models.RecordTypeText,
				Name:       "local",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "local"},
			},
			BaseRevision: 2,
		},
	})

	require.NoError(t, err)
	require.Empty(t, accepted)
	require.Len(t, conflicts, 1)
	require.Equal(t, int64(11), conflicts[0].RecordID)
	require.False(t, conflicts[0].Resolved)
}

func TestPush_DeleteChange(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	record := &models.Record{ID: 21, UserID: 1, Revision: 4}

	rr.GetRecordMock.Expect(21).Return(record, nil)
	rr.DeleteRecordMock.Expect(21).Return(nil)
	sr.GetMaxRevisionMock.Expect(1).Return(4, nil)
	sr.CreateRevisionMock.Inspect(func(rev *models.RecordRevision) {
		require.Equal(t, int64(21), rev.RecordID)
		require.Equal(t, int64(5), rev.Revision)
		require.Equal(t, "device-1", rev.DeviceID)
	}).Return(nil)

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		{
			Record:       &models.Record{ID: 21},
			Deleted:      true,
			BaseRevision: 4,
		},
	})

	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(5), accepted[0].Revision)
}

func TestPush_DeleteAlreadyDeletedRecord(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	now := time.Now()
	rr.GetRecordMock.Expect(21).Return(&models.Record{
		ID:        21,
		UserID:    1,
		Revision:  4,
		DeletedAt: &now,
	}, nil)

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		{
			Record:       &models.Record{ID: 21},
			Deleted:      true,
			BaseRevision: 4,
		},
	})

	require.NoError(t, err)
	require.Empty(t, accepted)
	require.Empty(t, conflicts)
}

func TestPush_DeleteConflict(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	record := &models.Record{
		ID:         21,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "server-update",
		Revision:   4,
		DeviceID:   "device-2",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "server"},
	}

	rr.GetRecordMock.Expect(21).Return(record, nil)
	sr.CreateConflictMock.Inspect(func(conflict *models.SyncConflict) {
		require.Equal(t, int64(3), conflict.LocalRevision)
		require.Equal(t, int64(4), conflict.ServerRevision)
		require.Nil(t, conflict.LocalRecord)
		require.Equal(t, record, conflict.ServerRecord)
	}).Return(nil)

	accepted, conflicts, err := svc.Push(1, "device-1", []models.PendingChange{
		{
			Record:       &models.Record{ID: 21},
			Deleted:      true,
			BaseRevision: 3,
		},
	})

	require.NoError(t, err)
	require.Empty(t, accepted)
	require.Len(t, conflicts, 1)
}

func TestPull_WithRevisionsAndLimit(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	revisions := []models.RecordRevision{
		{RecordID: 10, UserID: 1, Revision: 1, DeviceID: "device-1"},
		{RecordID: 10, UserID: 1, Revision: 2, DeviceID: "device-2"},
		{RecordID: 20, UserID: 1, Revision: 3, DeviceID: "device-3"},
	}
	conflicts := []models.SyncConflict{{ID: 7, UserID: 1, RecordID: 20}}
	getRecordCalls := 0

	sr.GetRevisionsMock.Expect(1, 0).Return(revisions, nil)
	rr.GetRecordMock.Set(func(id int64) (*models.Record, error) {
		getRecordCalls++
		switch id {
		case 10:
			return &models.Record{ID: 10, UserID: 1, Type: models.RecordTypeText, Name: "first", DeviceID: "device-1", KeyVersion: 1, Payload: models.TextPayload{Content: "a"}}, nil
		case 20:
			return &models.Record{ID: 20, UserID: 1, Type: models.RecordTypeText, Name: "second", DeviceID: "device-1", KeyVersion: 1, Payload: models.TextPayload{Content: "b"}}, nil
		default:
			return nil, errors.New("unexpected id")
		}
	})
	sr.GetConflictsMock.Expect(1).Return(conflicts, nil)

	gotRevisions, records, gotConflicts, err := svc.Pull(1, "device-1", 0, 2)

	require.NoError(t, err)
	require.Len(t, gotRevisions, 2)
	require.Len(t, records, 1)
	require.Len(t, gotConflicts, 1)
	require.Equal(t, 1, getRecordCalls)
	require.Equal(t, int64(10), records[0].ID)
}

func TestGetConflicts(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	expected := []models.SyncConflict{
		{ID: 1, UserID: 1, RecordID: 10},
		{ID: 2, UserID: 1, RecordID: 20},
	}
	sr.GetConflictsMock.Expect(1).Return(expected, nil)

	conflicts, err := svc.GetConflicts(1)

	require.NoError(t, err)
	require.Equal(t, expected, conflicts)
}

func TestResolveConflict_LocalResolution(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	localRecord := &models.Record{
		Type:       models.RecordTypeText,
		Name:       "local-update",
		DeviceID:   "device-1",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "local"},
	}
	serverRecord := &models.Record{
		ID:         33,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "server-update",
		Revision:   4,
		DeviceID:   "device-2",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "server"},
	}

	sr.GetConflictsMock.Expect(1).Return([]models.SyncConflict{
		{
			ID:             7,
			UserID:         1,
			RecordID:       33,
			LocalRevision:  3,
			ServerRevision: 4,
			LocalRecord:    localRecord,
			ServerRecord:   serverRecord,
		},
	}, nil)
	sr.ResolveConflictMock.Expect(7, models.ConflictResolutionLocal).Return(nil)
	sr.GetMaxRevisionMock.Expect(1).Return(4, nil)
	rr.UpdateRecordMock.Inspect(func(record *models.Record) {
		require.Equal(t, int64(33), record.ID)
		require.Equal(t, int64(1), record.UserID)
		require.Equal(t, "local-update", record.Name)
		require.Equal(t, int64(5), record.Revision)
		require.Equal(t, "device-1", record.DeviceID)
	}).Return(nil)
	sr.CreateRevisionMock.Inspect(func(rev *models.RecordRevision) {
		require.Equal(t, int64(33), rev.RecordID)
		require.Equal(t, int64(5), rev.Revision)
		require.Equal(t, "device-1", rev.DeviceID)
	}).Return(nil)

	record, err := svc.ResolveConflict(1, 7, models.ConflictResolutionLocal)

	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "local-update", record.Name)
	require.Equal(t, int64(5), record.Revision)
}

func TestResolveConflict_ServerResolution(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	serverRecord := &models.Record{
		ID:         33,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "server-update",
		Revision:   4,
		DeviceID:   "device-2",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "server"},
	}

	sr.GetConflictsMock.Expect(1).Return([]models.SyncConflict{
		{
			ID:             7,
			UserID:         1,
			RecordID:       33,
			LocalRevision:  3,
			ServerRevision: 4,
			LocalRecord: &models.Record{
				Type:       models.RecordTypeText,
				Name:       "local-update",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "local"},
			},
			ServerRecord: serverRecord,
		},
	}, nil)
	sr.ResolveConflictMock.Expect(7, models.ConflictResolutionServer).Return(nil)

	record, err := svc.ResolveConflict(1, 7, models.ConflictResolutionServer)

	require.NoError(t, err)
	require.Equal(t, serverRecord, record)
}

func TestResolveConflict_NotFound(t *testing.T) {
	mc := minimock.NewController(t)
	sr := syncmocks.NewSyncRepoMock(mc)
	rr := syncmocks.NewRecordRepoMock(mc)
	svc := newTestService(t, sr, rr)

	sr.GetConflictsMock.Expect(1).Return(nil, nil)

	record, err := svc.ResolveConflict(1, 999, models.ConflictResolutionLocal)

	require.Error(t, err)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
	require.Nil(t, record)
}
