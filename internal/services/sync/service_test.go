package sync

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	"github.com/hydra13/gophkeeper/internal/models"
)

// ---------------------------------------------------------------------------
// Mock repos
// ---------------------------------------------------------------------------

type mockSyncRepo struct {
	revisions []models.RecordRevision
	conflicts []models.SyncConflict
	maxRev    int64
	nextID    int64
	// error hooks
	getRevisionsErr    error
	createRevisionErr  error
	getMaxRevisionErr  error
	getConflictsErr    error
	createConflictErr  error
	resolveConflictErr error
}

func newMockSyncRepo() *mockSyncRepo {
	return &mockSyncRepo{}
}

func (m *mockSyncRepo) GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error) {
	if m.getRevisionsErr != nil {
		return nil, m.getRevisionsErr
	}
	var result []models.RecordRevision
	for _, r := range m.revisions {
		if r.Revision > sinceRevision {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockSyncRepo) CreateRevision(rev *models.RecordRevision) error {
	if m.createRevisionErr != nil {
		return m.createRevisionErr
	}
	m.nextID++
	rev.ID = m.nextID
	m.maxRev = rev.Revision
	m.revisions = append(m.revisions, *rev)
	return nil
}

func (m *mockSyncRepo) GetMaxRevision(userID int64) (int64, error) {
	if m.getMaxRevisionErr != nil {
		return 0, m.getMaxRevisionErr
	}
	return m.maxRev, nil
}

func (m *mockSyncRepo) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	if m.getConflictsErr != nil {
		return nil, m.getConflictsErr
	}
	return m.conflicts, nil
}

func (m *mockSyncRepo) CreateConflict(conflict *models.SyncConflict) error {
	if m.createConflictErr != nil {
		return m.createConflictErr
	}
	m.nextID++
	conflict.ID = m.nextID
	m.conflicts = append(m.conflicts, *conflict)
	return nil
}

func (m *mockSyncRepo) ResolveConflict(conflictID int64, resolution string) error {
	if m.resolveConflictErr != nil {
		return m.resolveConflictErr
	}
	for i := range m.conflicts {
		if m.conflicts[i].ID == conflictID {
			m.conflicts[i].Resolved = true
			m.conflicts[i].Resolution = resolution
			break
		}
	}
	return nil
}

type mockRecordRepo struct {
	records map[int64]*models.Record
	nextID  int64
	// error hooks
	createErr error
	getErr    error
	updateErr error
	deleteErr error
}

func newMockRecordRepo() *mockRecordRepo {
	return &mockRecordRepo{
		records: make(map[int64]*models.Record),
		nextID:  1,
	}
}

func (m *mockRecordRepo) CreateRecord(record *models.Record) error {
	if m.createErr != nil {
		return m.createErr
	}
	record.ID = m.nextID
	m.nextID++
	clone := *record
	m.records[clone.ID] = &clone
	return nil
}

func (m *mockRecordRepo) GetRecord(id int64) (*models.Record, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	r, ok := m.records[id]
	if !ok {
		return nil, fmt.Errorf("record %d not found", id)
	}
	clone := *r
	return &clone, nil
}

func (m *mockRecordRepo) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	var result []models.Record
	for _, r := range m.records {
		if r.UserID == userID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRecordRepo) UpdateRecord(record *models.Record) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	clone := *record
	m.records[clone.ID] = &clone
	return nil
}

func (m *mockRecordRepo) DeleteRecord(id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	r, ok := m.records[id]
	if !ok {
		return fmt.Errorf("record %d not found", id)
	}
	now := time.Now()
	r.DeletedAt = &now
	r.UpdatedAt = now
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(t *testing.T, sr *mockSyncRepo, rr *mockRecordRepo) *Service {
	t.Helper()
	if sr == nil {
		sr = newMockSyncRepo()
	}
	if rr == nil {
		rr = newMockRecordRepo()
	}
	svc, err := NewService(sr, rr)
	require.NoError(t, err)
	return svc
}

func textChange(userID int64, name, deviceID string, baseRev int64) sync_push_v1_post.PendingChange {
	return sync_push_v1_post.PendingChange{
		Record: recordscommon.RecordDTO{
			UserID:     userID,
			Type:       "text",
			Name:       name,
			DeviceID:   deviceID,
			KeyVersion: 1,
			Payload:    map[string]interface{}{"content": "hello"},
		},
		BaseRevision: baseRev,
	}
}

// createAndReturnRecordID is a helper that creates a record via Push and returns the record ID.
// The stored record will have Revision equal to the value set by pushCreate (which is 0,
// since pushCreate does not set a revision on the Record before CreateRecord).
// The accepted revision number (from record_revisions) will be 1.
func createAndReturnRecordID(t *testing.T, svc *Service, rr *mockRecordRepo) (recordID int64) {
	t.Helper()
	accepted, _, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "original", "device-1", 0),
	})
	require.NoError(t, err)
	require.Len(t, accepted, 1)
	return accepted[0].RecordID
}

// ---------------------------------------------------------------------------
// Tests: NewService
// ---------------------------------------------------------------------------

func TestNewService_NilSyncRepo(t *testing.T) {
	rr := newMockRecordRepo()
	_, err := NewService(nil, rr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sync repository is required")
}

func TestNewService_NilRecordRepo(t *testing.T) {
	sr := newMockSyncRepo()
	_, err := NewService(sr, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record repository is required")
}

// ---------------------------------------------------------------------------
// Tests: Push - Create
// ---------------------------------------------------------------------------

func TestPush_CreateChange(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	accepted, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "my note", "device-1", 0),
	})
	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(1), accepted[0].Revision)
	require.Equal(t, int64(1), accepted[0].RecordID)

	// Record should be persisted
	record, err := rr.GetRecord(accepted[0].RecordID)
	require.NoError(t, err)
	require.Equal(t, "my note", record.Name)
	require.Equal(t, models.RecordTypeText, record.Type)
}

// ---------------------------------------------------------------------------
// Tests: Push - Update
// ---------------------------------------------------------------------------

func TestPush_UpdateChange_MatchingRevision(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create a record: pushCreate stores record with Revision=0,
	// then creates a revision entry with Revision=1.
	recordID := createAndReturnRecordID(t, svc, rr)

	// The stored record has Revision=0. Update with BaseRevision=0 to match.
	updateChange := sync_push_v1_post.PendingChange{
		Record: recordscommon.RecordDTO{
			ID:         recordID,
			UserID:     1,
			Type:       "text",
			Name:       "updated",
			DeviceID:   "device-1",
			KeyVersion: 1,
			Payload:    map[string]interface{}{"content": "new"},
		},
		BaseRevision: 0, // matches the stored record's Revision
	}
	accepted, conflicts, err := svc.Push(1, "device-2", []sync_push_v1_post.PendingChange{updateChange})
	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(2), accepted[0].Revision)

	// Record should be updated
	record, err := rr.GetRecord(recordID)
	require.NoError(t, err)
	require.Equal(t, "updated", record.Name)
	require.Equal(t, int64(2), record.Revision)
}

func TestPush_UpdateConflict_NonMatchingRevision(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create
	recordID := createAndReturnRecordID(t, svc, rr)

	// First update with matching revision (BaseRevision=0 matches stored Revision=0)
	_, _, err := svc.Push(1, "device-2", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "server-update", DeviceID: "device-2",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "server"},
			},
			BaseRevision: 0,
		},
	})
	require.NoError(t, err)

	// Now stored record has Revision=2 (after BumpRevision).
	// Second update with stale BaseRevision=0 triggers conflict.
	_, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "local-update", DeviceID: "device-1",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "local"},
			},
			BaseRevision: 0, // stale: record is now at revision 2
		},
	})
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	require.Equal(t, recordID, conflicts[0].RecordID)
	require.False(t, conflicts[0].Resolved)
}

// ---------------------------------------------------------------------------
// Tests: Push - Delete
// ---------------------------------------------------------------------------

func TestPush_DeleteChange(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	recordID := createAndReturnRecordID(t, svc, rr)

	// Delete with BaseRevision=0 (matches stored record Revision=0)
	accepted, _, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record:       recordscommon.RecordDTO{ID: recordID},
			Deleted:      true,
			BaseRevision: 0,
		},
	})
	require.NoError(t, err)
	require.Len(t, accepted, 1)

	// Record should be soft-deleted
	record, err := rr.GetRecord(recordID)
	require.NoError(t, err)
	require.NotNil(t, record.DeletedAt)
}

func TestPush_DeleteAlreadyDeletedRecord(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	recordID := createAndReturnRecordID(t, svc, rr)

	// First delete (BaseRevision=0 matches, or BaseRevision=0 triggers delete without conflict check when 0)
	_, _, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record:  recordscommon.RecordDTO{ID: recordID},
			Deleted: true,
		},
	})
	require.NoError(t, err)

	// Second delete of already-deleted record: no error, no accepted, no conflicts
	accepted2, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record:  recordscommon.RecordDTO{ID: recordID},
			Deleted: true,
		},
	})
	require.NoError(t, err)
	require.Empty(t, accepted2)
	require.Empty(t, conflicts)
}

// ---------------------------------------------------------------------------
// Tests: Pull
// ---------------------------------------------------------------------------

func TestPull_WithRevisions(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create two records
	_, _, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "first", "device-1", 0),
	})
	require.NoError(t, err)

	_, _, err = svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "second", "device-1", 0),
	})
	require.NoError(t, err)

	// Pull from 0: should get all revisions
	revs, records, _, err := svc.Pull(1, "device-1", 0, 50)
	require.NoError(t, err)
	require.Len(t, revs, 2)
	require.Len(t, records, 2)
}

func TestPull_WithLimit(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create two records
	_, _, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "first", "device-1", 0),
	})
	require.NoError(t, err)

	_, _, err = svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		textChange(1, "second", "device-1", 0),
	})
	require.NoError(t, err)

	// Pull with limit=1: should get only 1 revision
	revs, _, _, err := svc.Pull(1, "device-1", 0, 1)
	require.NoError(t, err)
	require.Len(t, revs, 1)

	// Pull with sinceRevision=1: should skip revision 1
	revs2, _, _, err := svc.Pull(1, "device-1", 1, 50)
	require.NoError(t, err)
	require.Len(t, revs2, 1)
}

// ---------------------------------------------------------------------------
// Tests: GetConflicts
// ---------------------------------------------------------------------------

func TestGetConflicts(t *testing.T) {
	sr := newMockSyncRepo()
	sr.conflicts = []models.SyncConflict{
		{ID: 1, UserID: 1, RecordID: 10, Resolved: false},
		{ID: 2, UserID: 1, RecordID: 20, Resolved: false},
	}
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	conflicts, err := svc.GetConflicts(1)
	require.NoError(t, err)
	require.Len(t, conflicts, 2)
}

// ---------------------------------------------------------------------------
// Tests: ResolveConflict
// ---------------------------------------------------------------------------

func TestResolveConflict_LocalResolution(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create a record
	recordID := createAndReturnRecordID(t, svc, rr)

	// Update from device-2 with matching revision (BaseRevision=0)
	_, _, err := svc.Push(1, "device-2", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "server-update", DeviceID: "device-2",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "server"},
			},
			BaseRevision: 0,
		},
	})
	require.NoError(t, err)

	// Now record has Revision=2. Conflict from device-1 with stale BaseRevision=0.
	_, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "local-update", DeviceID: "device-1",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "local"},
			},
			BaseRevision: 0, // stale
		},
	})
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Resolve in favor of local
	record, err := svc.ResolveConflict(1, conflicts[0].ID, models.ConflictResolutionLocal)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "local-update", record.Name)
}

func TestResolveConflict_ServerResolution(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create a record
	recordID := createAndReturnRecordID(t, svc, rr)

	// Update from device-2
	_, _, err := svc.Push(1, "device-2", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "server-update", DeviceID: "device-2",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "server"},
			},
			BaseRevision: 0,
		},
	})
	require.NoError(t, err)

	// Conflict from device-1
	_, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "local-update", DeviceID: "device-1",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "local"},
			},
			BaseRevision: 0, // stale
		},
	})
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Resolve in favor of server
	record, err := svc.ResolveConflict(1, conflicts[0].ID, models.ConflictResolutionServer)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "server-update", record.Name)
}

func TestResolveConflict_NotFound(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	_, err := svc.ResolveConflict(1, 999, models.ConflictResolutionLocal)
	require.Error(t, err)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestPush_DeleteConflict(t *testing.T) {
	sr := newMockSyncRepo()
	rr := newMockRecordRepo()
	svc := newTestService(t, sr, rr)

	// Create a record (gets Revision=1 after push)
	recordID := createAndReturnRecordID(t, svc, rr)

	// Update from device-2 (record now has Revision=2)
	_, _, err := svc.Push(1, "device-2", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "server-update", DeviceID: "device-2",
				KeyVersion: 1, Payload: map[string]interface{}{"content": "server"},
			},
			BaseRevision: 0,
		},
	})
	require.NoError(t, err)

	// Delete from device-1 with stale BaseRevision=1 -> should create delete conflict
	_, conflicts, err := svc.Push(1, "device-1", []sync_push_v1_post.PendingChange{
		{
			Record: recordscommon.RecordDTO{
				ID: recordID, UserID: 1, Type: "text",
				Name: "deleted-record", DeviceID: "device-1",
				KeyVersion: 1,
			},
			Deleted:       true,
			BaseRevision:  1, // stale — record is at revision 2
		},
	})
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
}

func TestStrVal(t *testing.T) {
	require.Equal(t, "", strVal(nil))
	require.Equal(t, "hello", strVal("hello"))
	require.Equal(t, "", strVal(42))
	require.Equal(t, "", strVal([]string{}))
}
