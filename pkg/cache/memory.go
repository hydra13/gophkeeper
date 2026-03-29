package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hydra13/gophkeeper/internal/models"
)

// FileStore — реализация Store с JSON-персистентностью на диск.
type FileStore struct {
	mu       sync.RWMutex
	dir      string
	records  *recordCacheImpl
	pending  *pendingQueueImpl
	transfers *transferStateImpl
	auth     *authStoreImpl
	syncState *syncStateImpl
}

// NewFileStore создаёт файловый кеш в указанной директории.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	s := &FileStore{
		dir: dir,
		records:  &recordCacheImpl{data: make(map[int64]models.Record)},
		pending:  &pendingQueueImpl{},
		transfers: &transferStateImpl{data: make(map[int64]Transfer)},
		auth:     &authStoreImpl{},
		syncState: &syncStateImpl{},
	}

	if err := s.load(); err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	return s, nil
}

func (s *FileStore) Records() RecordCache      { return s.records }
func (s *FileStore) Pending() PendingQueue      { return s.pending }
func (s *FileStore) Transfers() TransferState   { return s.transfers }
func (s *FileStore) Auth() AuthStore            { return s.auth }
func (s *FileStore) Sync() SyncState            { return s.syncState }

// Flush сохраняет все данные на диск.
func (s *FileStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked()
}

func (s *FileStore) flushLocked() error {
	type persistent struct {
		Records   map[int64]jsonRecord  `json:"records"`
		Pending   []jsonPendingOp       `json:"pending"`
		Transfers map[int64]Transfer    `json:"transfers"`
		Auth      *AuthData             `json:"auth,omitempty"`
		Sync      SyncData              `json:"sync"`
	}

	records := make(map[int64]jsonRecord, len(s.records.data))
	for id, rec := range s.records.data {
		jr, err := recordToJSON(rec)
		if err != nil {
			return fmt.Errorf("serialize record %d: %w", id, err)
		}
		records[id] = jr
	}

	pending := make([]jsonPendingOp, 0, len(s.pending.ops))
	for _, op := range s.pending.ops {
		jop, err := pendingOpToJSON(op)
		if err != nil {
			return fmt.Errorf("serialize pending op: %w", err)
		}
		pending = append(pending, jop)
	}

	data := persistent{
		Records:   records,
		Pending:   pending,
		Transfers: s.transfers.data,
		Auth:      s.auth.data,
		Sync:      s.syncState.data,
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	path := filepath.Join(s.dir, "cache.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return fmt.Errorf("write cache tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename cache: %w", err)
	}
	return nil
}

func (s *FileStore) load() error {
	path := filepath.Join(s.dir, "cache.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read cache: %w", err)
	}

	type persistent struct {
		Records   map[int64]jsonRecord  `json:"records"`
		Pending   []jsonPendingOp       `json:"pending"`
		Transfers map[int64]Transfer    `json:"transfers"`
		Auth      *AuthData             `json:"auth,omitempty"`
		Sync      SyncData              `json:"sync"`
	}

	var data persistent
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("unmarshal cache: %w", err)
	}

	if data.Records != nil {
		s.records.data = make(map[int64]models.Record, len(data.Records))
		for id, jr := range data.Records {
			rec, err := jsonToRecord(jr)
			if err != nil {
				return fmt.Errorf("deserialize record %d: %w", id, err)
			}
			s.records.data[id] = rec
		}
	}
	if data.Pending != nil {
		s.pending.ops = make([]PendingOp, 0, len(data.Pending))
		for _, jop := range data.Pending {
			op, err := jsonToPendingOp(jop)
			if err != nil {
				return fmt.Errorf("deserialize pending op: %w", err)
			}
			s.pending.ops = append(s.pending.ops, op)
		}
	}
	if data.Transfers != nil {
		s.transfers.data = data.Transfers
	}
	if data.Auth != nil {
		s.auth.data = data.Auth
	}
	s.syncState.data = data.Sync

	return nil
}

// --- RecordCache ---

type recordCacheImpl struct {
	data map[int64]models.Record
}

func (r *recordCacheImpl) Get(id int64) (*models.Record, bool) {
	rec, ok := r.data[id]
	if !ok {
		return nil, false
	}
	return &rec, true
}

func (r *recordCacheImpl) GetAll() []models.Record {
	result := make([]models.Record, 0, len(r.data))
	for _, rec := range r.data {
		result = append(result, rec)
	}
	return result
}

func (r *recordCacheImpl) Put(record *models.Record) {
	if record == nil {
		return
	}
	r.data[record.ID] = *record
}

func (r *recordCacheImpl) PutAll(records []models.Record) {
	r.data = make(map[int64]models.Record, len(records))
	for i := range records {
		r.data[records[i].ID] = records[i]
	}
}

func (r *recordCacheImpl) Delete(id int64) {
	delete(r.data, id)
}

func (r *recordCacheImpl) Clear() {
	r.data = make(map[int64]models.Record)
}

// --- PendingQueue ---

type pendingQueueImpl struct {
	ops []PendingOp
}

func (q *pendingQueueImpl) Enqueue(op PendingOp) error {
	q.ops = append(q.ops, op)
	return nil
}

func (q *pendingQueueImpl) DequeueAll() ([]PendingOp, error) {
	ops := q.ops
	q.ops = nil
	return ops, nil
}

func (q *pendingQueueImpl) Peek() ([]PendingOp, error) {
	result := make([]PendingOp, len(q.ops))
	copy(result, q.ops)
	return result, nil
}

func (q *pendingQueueImpl) Len() int {
	return len(q.ops)
}

func (q *pendingQueueImpl) Clear() {
	q.ops = nil
}

// --- TransferState ---

type transferStateImpl struct {
	data map[int64]Transfer
}

func (t *transferStateImpl) Save(tr Transfer) error {
	t.data[tr.ID] = tr
	return nil
}

func (t *transferStateImpl) Get(id int64) (Transfer, bool) {
	tr, ok := t.data[id]
	return tr, ok
}

func (t *transferStateImpl) GetByRecord(recordID int64) (Transfer, bool) {
	for _, tr := range t.data {
		if tr.RecordID == recordID {
			return tr, true
		}
	}
	return Transfer{}, false
}

func (t *transferStateImpl) Delete(id int64) {
	delete(t.data, id)
}

func (t *transferStateImpl) ListActive() []Transfer {
	var result []Transfer
	for _, tr := range t.data {
		if tr.Status == TransferStatusActive {
			result = append(result, tr)
		}
	}
	return result
}

func (t *transferStateImpl) Clear() {
	t.data = make(map[int64]Transfer)
}

// --- AuthStore ---

type authStoreImpl struct {
	data *AuthData
}

func (a *authStoreImpl) Get() (*AuthData, bool) {
	if a.data == nil {
		return nil, false
	}
	cp := *a.data
	return &cp, true
}

func (a *authStoreImpl) Set(data AuthData) error {
	a.data = &data
	return nil
}

func (a *authStoreImpl) Clear() {
	a.data = nil
}

// --- SyncState ---

type syncStateImpl struct {
	data SyncData
}

func (s *syncStateImpl) Get() SyncData {
	return s.data
}

func (s *syncStateImpl) SetLastRevision(rev int64) error {
	s.data.LastRevision = rev
	return nil
}
