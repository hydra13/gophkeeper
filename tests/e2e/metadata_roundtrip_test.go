//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	apiclient "github.com/hydra13/gophkeeper/pkg/apiclient"
	"github.com/hydra13/gophkeeper/pkg/cache"
	"github.com/hydra13/gophkeeper/pkg/clientcore"
)

const bufSize = 1024 * 1024

// ---------------------------------------------------------------------------
// In-memory repository for e2e tests
// ---------------------------------------------------------------------------

type e2eRecordRepo struct {
	lastID  int64
	records map[int64]*models.Record
}

func newE2ERecordRepo() *e2eRecordRepo {
	return &e2eRecordRepo{records: make(map[int64]*models.Record)}
}

func (r *e2eRecordRepo) CreateRecord(record *models.Record) error {
	r.lastID++
	record.ID = r.lastID
	record.Revision = 1
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	copied := *record
	r.records[record.ID] = &copied
	return nil
}

func (r *e2eRecordRepo) GetRecord(id int64) (*models.Record, error) {
	rec, ok := r.records[id]
	if !ok {
		return nil, models.ErrRecordNotFound
	}
	copied := *rec
	return &copied, nil
}

func (r *e2eRecordRepo) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	var result []models.Record
	for _, rec := range r.records {
		if rec.UserID != userID {
			continue
		}
		if !includeDeleted && rec.DeletedAt != nil {
			continue
		}
		if recordType != "" && rec.Type != recordType {
			continue
		}
		result = append(result, *rec)
	}
	return result, nil
}

func (r *e2eRecordRepo) UpdateRecord(record *models.Record) error {
	if _, ok := r.records[record.ID]; !ok {
		return models.ErrRecordNotFound
	}
	record.UpdatedAt = time.Now()
	copied := *record
	r.records[record.ID] = &copied
	return nil
}

func (r *e2eRecordRepo) DeleteRecord(id int64) error {
	rec, ok := r.records[id]
	if !ok {
		return models.ErrRecordNotFound
	}
	now := time.Now()
	rec.DeletedAt = &now
	return nil
}

// e2eSyncRepo — minimal sync repository for push/pull testing
type e2eSyncRepo struct {
	revLastID int64
	revisions []models.RecordRevision
}

func newE2ESyncRepo() *e2eSyncRepo {
	return &e2eSyncRepo{}
}

func (s *e2eSyncRepo) GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error) {
	var result []models.RecordRevision
	for _, rev := range s.revisions {
		if rev.UserID == userID && rev.Revision > sinceRevision {
			result = append(result, rev)
		}
	}
	return result, nil
}

func (s *e2eSyncRepo) CreateRevision(rev *models.RecordRevision) error {
	s.revLastID++
	rev.ID = s.revLastID
	s.revisions = append(s.revisions, *rev)
	return nil
}

func (s *e2eSyncRepo) GetMaxRevision(userID int64) (int64, error) {
	var max int64
	for _, rev := range s.revisions {
		if rev.UserID == userID && rev.Revision > max {
			max = rev.Revision
		}
	}
	return max, nil
}

func (s *e2eSyncRepo) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return nil, nil
}

func (s *e2eSyncRepo) CreateConflict(conflict *models.SyncConflict) error {
	return nil
}

func (s *e2eSyncRepo) ResolveConflict(conflictID int64, resolution string) error {
	return nil
}

func (s *e2eSyncRepo) GetConflictByID(conflictID int64) (*models.SyncConflict, error) {
	return nil, models.ErrRecordNotFound
}

func (s *e2eSyncRepo) UpdateConflict(conflict *models.SyncConflict) error {
	return nil
}

// ---------------------------------------------------------------------------
// Test environment setup
// ---------------------------------------------------------------------------

type e2eTestEnv struct {
	lis      *bufconn.Listener
	server   *grpc.Server
	conn     *grpc.ClientConn
	repo     *e2eRecordRepo
	syncRepo *e2eSyncRepo
	client   *e2eTransportClient
	core     *clientcore.ClientCore
	store    cache.Store
	cleanup  func()
}

func setupE2E(t *testing.T) *e2eTestEnv {
	t.Helper()

	repo := newE2ERecordRepo()
	syncRepo := newE2ESyncRepo()
	log := zerolog.Nop()

	// Create real services
	recordUseCase := &e2eRecordUseCase{repo: repo}
	syncUseCase := &e2eSyncUseCase{repo: repo, syncRepo: syncRepo}

	// Create gRPC services
	dataService := rpc.NewDataService(recordUseCase, log)
	syncService := rpc.NewSyncService(syncUseCase, log)

	// Auth service that accepts any token and returns userID=1
	authService := &e2eAuthService{}

	// Health service
	healthService := rpc.NewHealthService()

	// Uploads service stub
	uploadsService := &e2eUploadsService{}

	rpcServer := rpc.NewServer(authService, dataService, syncService, uploadsService, healthService)

	lis := bufconn.Listen(bufSize)

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			e2eAuthInterceptor,
		),
	)
	rpcServer.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("server serve: %v", err)
		}
	}()

	// Create client connection
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	// Create transport client
	client := &e2eTransportClient{
		conn:        conn,
		dataClient:  pbv1.NewDataServiceClient(conn),
		syncClient:  pbv1.NewSyncServiceClient(conn),
		accessToken: "test-token",
	}

	// Create cache store
	dir := t.TempDir()
	store, err := cache.NewFileStore(dir)
	require.NoError(t, err)

	// Create clientcore
	core := clientcore.New(client, store, clientcore.Config{
		DeviceID: "e2e-test-device",
	})

	// Login so clientcore is authenticated (needed for online operations)
	err = core.Login(context.Background(), "e2e@test.com", "password")
	require.NoError(t, err)

	env := &e2eTestEnv{
		lis:      lis,
		server:   s,
		conn:     conn,
		repo:     repo,
		syncRepo: syncRepo,
		client:   client,
		core:     core,
		store:    store,
		cleanup: func() {
			_ = conn.Close()
			s.GracefulStop()
		},
	}

	t.Cleanup(env.cleanup)
	return env
}

// ---------------------------------------------------------------------------
// gRPC Transport client using bufconn (implements apiclient.Transport)
// ---------------------------------------------------------------------------

type e2eTransportClient struct {
	conn        *grpc.ClientConn
	dataClient  pbv1.DataServiceClient
	syncClient  pbv1.SyncServiceClient
	accessToken string
}

func (c *e2eTransportClient) authCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.accessToken)
}

func (c *e2eTransportClient) Register(ctx context.Context, email, password string) (int64, error) {
	return 1, nil
}

func (c *e2eTransportClient) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	c.accessToken = "test-access-token"
	return "test-access-token", "test-refresh-token", nil
}

func (c *e2eTransportClient) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	return "new-access-token", "new-refresh-token", nil
}

func (c *e2eTransportClient) Logout(ctx context.Context) error {
	c.accessToken = ""
	return nil
}

func (c *e2eTransportClient) SetAccessToken(token string) {
	c.accessToken = token
}

func (c *e2eTransportClient) CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	pbRecord := domainToPb(record)
	pbPayload := domainToPbPayloadCreate(record)

	req := &pbv1.CreateRecordRequest{
		Type:           pbRecord.Type,
		Name:           pbRecord.Name,
		Metadata:       pbRecord.Metadata,
		DeviceId:       pbRecord.DeviceId,
		KeyVersion:     pbRecord.KeyVersion,
		PayloadVersion: pbRecord.PayloadVersion,
	}

	switch p := pbPayload.(type) {
	case *pbv1.CreateRecordRequest_Login:
		req.Payload = p
	case *pbv1.CreateRecordRequest_Text:
		req.Payload = p
	case *pbv1.CreateRecordRequest_Binary:
		req.Payload = p
	case *pbv1.CreateRecordRequest_Card:
		req.Payload = p
	}

	resp, err := c.dataClient.CreateRecord(c.authCtx(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("create record: %w", err)
	}
	return pbToDomain(resp.Record), nil
}

func (c *e2eTransportClient) GetRecord(ctx context.Context, id int64) (*models.Record, error) {
	resp, err := c.dataClient.GetRecord(c.authCtx(ctx), &pbv1.GetRecordRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("get record: %w", err)
	}
	return pbToDomain(resp.Record), nil
}

func (c *e2eTransportClient) ListRecords(ctx context.Context, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	resp, err := c.dataClient.ListRecords(c.authCtx(ctx), &pbv1.ListRecordsRequest{
		Type:           domainToPbType(recordType),
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	records := make([]models.Record, 0, len(resp.Records))
	for _, r := range resp.Records {
		records = append(records, *pbToDomain(r))
	}
	return records, nil
}

func (c *e2eTransportClient) UpdateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	pbRecord := domainToPb(record)
	pbPayload := domainToPbPayloadUpdate(record)

	req := &pbv1.UpdateRecordRequest{
		Id:             pbRecord.Id,
		Name:           pbRecord.Name,
		Metadata:       pbRecord.Metadata,
		DeviceId:       pbRecord.DeviceId,
		KeyVersion:     pbRecord.KeyVersion,
		PayloadVersion: pbRecord.PayloadVersion,
		Revision:       pbRecord.Revision,
	}

	switch p := pbPayload.(type) {
	case *pbv1.UpdateRecordRequest_Login:
		req.Payload = p
	case *pbv1.UpdateRecordRequest_Text:
		req.Payload = p
	case *pbv1.UpdateRecordRequest_Binary:
		req.Payload = p
	case *pbv1.UpdateRecordRequest_Card:
		req.Payload = p
	}

	resp, err := c.dataClient.UpdateRecord(c.authCtx(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("update record: %w", err)
	}
	return pbToDomain(resp.Record), nil
}

func (c *e2eTransportClient) DeleteRecord(ctx context.Context, id int64, deviceID string) error {
	_, err := c.dataClient.DeleteRecord(c.authCtx(ctx), &pbv1.DeleteRecordRequest{
		Id:       id,
		DeviceId: deviceID,
	})
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	return nil
}

func (c *e2eTransportClient) Pull(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
	resp, err := c.syncClient.Pull(c.authCtx(ctx), &pbv1.PullRequest{
		SinceRevision: sinceRevision,
		DeviceId:      deviceID,
		Limit:         limit,
	})
	if err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}
	result := &apiclient.PullResult{
		HasMore:      resp.HasMore,
		NextRevision: resp.NextRevision,
	}
	for _, r := range resp.Records {
		result.Records = append(result.Records, *pbToDomain(r))
	}
	return result, nil
}

func (c *e2eTransportClient) Push(ctx context.Context, changes []apiclient.PendingChange, deviceID string) (*apiclient.PushResult, error) {
	pbChanges := make([]*pbv1.PendingChange, 0, len(changes))
	for _, ch := range changes {
		pbChanges = append(pbChanges, &pbv1.PendingChange{
			Record:       domainToPb(ch.Record),
			Deleted:      ch.Deleted,
			BaseRevision: ch.BaseRevision,
		})
	}
	resp, err := c.syncClient.Push(c.authCtx(ctx), &pbv1.PushRequest{
		Changes:  pbChanges,
		DeviceId: deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("push: %w", err)
	}
	result := &apiclient.PushResult{}
	for _, a := range resp.Accepted {
		result.Accepted = append(result.Accepted, apiclient.AcceptedChange{
			RecordID: a.RecordId,
			Revision: a.Revision,
			DeviceID: a.DeviceId,
		})
	}
	return result, nil
}

// Stub methods for uploads (not needed for metadata tests)

func (c *e2eTransportClient) CreateUploadSession(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	return 0, nil
}
func (c *e2eTransportClient) UploadChunk(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
	return nil
}
func (c *e2eTransportClient) GetUploadStatus(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error) {
	return nil, nil
}
func (c *e2eTransportClient) CreateDownloadSession(ctx context.Context, recordID int64) (int64, int64, error) {
	return 0, 0, nil
}
func (c *e2eTransportClient) DownloadChunk(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
	return nil, nil
}
func (c *e2eTransportClient) ConfirmChunk(ctx context.Context, downloadID, chunkIndex int64) error {
	return nil
}
func (c *e2eTransportClient) GetDownloadStatus(ctx context.Context, downloadID int64) (*apiclient.DownloadStatus, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Protobuf <-> Domain conversion helpers (reuse same patterns as grpc/client.go)
// ---------------------------------------------------------------------------

func domainToPb(r *models.Record) *pbv1.Record {
	if r == nil {
		return nil
	}
	pb := &pbv1.Record{
		Id:             r.ID,
		UserId:         r.UserID,
		Type:           domainToPbType(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		DeviceId:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
	}
	if r.DeletedAt != nil {
		pb.DeletedAt = timestamppb.New(*r.DeletedAt)
	}
	if !r.CreatedAt.IsZero() {
		pb.CreatedAt = timestamppb.New(r.CreatedAt)
	}
	if !r.UpdatedAt.IsZero() {
		pb.UpdatedAt = timestamppb.New(r.UpdatedAt)
	}
	switch p := r.Payload.(type) {
	case models.LoginPayload:
		pb.Payload = &pbv1.Record_Login{Login: &pbv1.LoginPayload{Login: p.Login, Password: p.Password}}
	case models.TextPayload:
		pb.Payload = &pbv1.Record_Text{Text: &pbv1.TextPayload{Content: p.Content}}
	case models.BinaryPayload:
		pb.Payload = &pbv1.Record_Binary{Binary: &pbv1.BinaryPayload{}}
	case models.CardPayload:
		pb.Payload = &pbv1.Record_Card{Card: &pbv1.CardPayload{
			Number: p.Number, HolderName: p.HolderName, ExpiryDate: p.ExpiryDate, Cvv: p.CVV,
		}}
	}
	return pb
}

func pbToDomain(pb *pbv1.Record) *models.Record {
	if pb == nil {
		return nil
	}
	r := &models.Record{
		ID:             pb.Id,
		UserID:         pb.UserId,
		Type:           pbTypeToDomain(pb.Type),
		Name:           pb.Name,
		Metadata:       pb.Metadata,
		Revision:       pb.Revision,
		DeviceID:       pb.DeviceId,
		KeyVersion:     pb.KeyVersion,
		PayloadVersion: pb.PayloadVersion,
	}
	if pb.DeletedAt != nil {
		t := pb.DeletedAt.AsTime()
		r.DeletedAt = &t
	}
	if pb.CreatedAt != nil {
		r.CreatedAt = pb.CreatedAt.AsTime()
	}
	if pb.UpdatedAt != nil {
		r.UpdatedAt = pb.UpdatedAt.AsTime()
	}
	switch p := pb.Payload.(type) {
	case *pbv1.Record_Login:
		r.Payload = models.LoginPayload{Login: p.Login.Login, Password: p.Login.Password}
	case *pbv1.Record_Text:
		r.Payload = models.TextPayload{Content: p.Text.Content}
	case *pbv1.Record_Binary:
		r.Payload = models.BinaryPayload{}
	case *pbv1.Record_Card:
		r.Payload = models.CardPayload{
			Number: p.Card.Number, HolderName: p.Card.HolderName,
			ExpiryDate: p.Card.ExpiryDate, CVV: p.Card.Cvv,
		}
	}
	return r
}

func domainToPbType(rt models.RecordType) pbv1.RecordType {
	switch rt {
	case models.RecordTypeLogin:
		return pbv1.RecordType_RECORD_TYPE_LOGIN
	case models.RecordTypeText:
		return pbv1.RecordType_RECORD_TYPE_TEXT
	case models.RecordTypeBinary:
		return pbv1.RecordType_RECORD_TYPE_BINARY
	case models.RecordTypeCard:
		return pbv1.RecordType_RECORD_TYPE_CARD
	default:
		return pbv1.RecordType_RECORD_TYPE_UNSPECIFIED
	}
}

func pbTypeToDomain(rt pbv1.RecordType) models.RecordType {
	switch rt {
	case pbv1.RecordType_RECORD_TYPE_LOGIN:
		return models.RecordTypeLogin
	case pbv1.RecordType_RECORD_TYPE_TEXT:
		return models.RecordTypeText
	case pbv1.RecordType_RECORD_TYPE_BINARY:
		return models.RecordTypeBinary
	case pbv1.RecordType_RECORD_TYPE_CARD:
		return models.RecordTypeCard
	default:
		return ""
	}
}

func domainToPbPayloadCreate(r *models.Record) interface{} {
	if r.Payload == nil {
		return nil
	}
	switch p := r.Payload.(type) {
	case models.LoginPayload:
		return &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: p.Login, Password: p.Password}}
	case models.TextPayload:
		return &pbv1.CreateRecordRequest_Text{Text: &pbv1.TextPayload{Content: p.Content}}
	case models.BinaryPayload:
		return &pbv1.CreateRecordRequest_Binary{Binary: &pbv1.BinaryPayload{}}
	case models.CardPayload:
		return &pbv1.CreateRecordRequest_Card{Card: &pbv1.CardPayload{
			Number: p.Number, HolderName: p.HolderName, ExpiryDate: p.ExpiryDate, Cvv: p.CVV,
		}}
	}
	return nil
}

func domainToPbPayloadUpdate(r *models.Record) interface{} {
	if r.Payload == nil {
		return nil
	}
	switch p := r.Payload.(type) {
	case models.LoginPayload:
		return &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: p.Login, Password: p.Password}}
	case models.TextPayload:
		return &pbv1.UpdateRecordRequest_Text{Text: &pbv1.TextPayload{Content: p.Content}}
	case models.BinaryPayload:
		return &pbv1.UpdateRecordRequest_Binary{Binary: &pbv1.BinaryPayload{}}
	case models.CardPayload:
		return &pbv1.UpdateRecordRequest_Card{Card: &pbv1.CardPayload{
			Number: p.Number, HolderName: p.HolderName, ExpiryDate: p.ExpiryDate, Cvv: p.CVV,
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Auth interceptor for e2e: sets userID=1 for any non-empty token
// ---------------------------------------------------------------------------

func e2eAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return handler(ctx, req)
	}
	values := md.Get("authorization")
	if len(values) > 0 && values[0] != "" {
		ctx = middlewares.ContextWithUserID(ctx, 1)
	}
	return handler(ctx, req)
}

// ---------------------------------------------------------------------------
// In-memory service implementations
// ---------------------------------------------------------------------------

// e2eRecordUseCase implements rpc.RecordUseCase using in-memory repo
type e2eRecordUseCase struct {
	repo *e2eRecordRepo
}

func (u *e2eRecordUseCase) CreateRecord(record *models.Record) error {
	return u.repo.CreateRecord(record)
}

func (u *e2eRecordUseCase) GetRecord(id int64) (*models.Record, error) {
	return u.repo.GetRecord(id)
}

func (u *e2eRecordUseCase) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return u.repo.ListRecords(userID, recordType, includeDeleted)
}

func (u *e2eRecordUseCase) UpdateRecord(record *models.Record) error {
	return u.repo.UpdateRecord(record)
}

func (u *e2eRecordUseCase) DeleteRecord(id int64, deviceID string) error {
	return u.repo.DeleteRecord(id)
}

// e2eSyncUseCase implements rpc.SyncUseCase
type e2eSyncUseCase struct {
	repo     *e2eRecordRepo
	syncRepo *e2eSyncRepo
}

func (u *e2eSyncUseCase) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	var revisions []models.RecordRevision
	for _, ch := range changes {
		if ch.Record == nil {
			continue
		}
		rec := *ch.Record
		rec.UserID = userID
		rec.DeviceID = deviceID

		if ch.Deleted {
			_ = u.repo.DeleteRecord(rec.ID)
		} else if rec.ID == 0 {
			_ = u.repo.CreateRecord(&rec)
		} else {
			_ = u.repo.UpdateRecord(&rec)
		}

		maxRev, _ := u.syncRepo.GetMaxRevision(userID)
		rev := &models.RecordRevision{
			UserID:   userID,
			RecordID: rec.ID,
			Revision: maxRev + 1,
			DeviceID: deviceID,
		}
		_ = u.syncRepo.CreateRevision(rev)
		revisions = append(revisions, *rev)
	}
	return revisions, nil, nil
}

func (u *e2eSyncUseCase) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	revs, err := u.syncRepo.GetRevisions(userID, sinceRevision)
	if err != nil {
		return nil, nil, nil, err
	}

	seen := make(map[int64]bool)
	var records []models.Record
	for _, rev := range revs {
		if !seen[rev.RecordID] {
			rec, err := u.repo.GetRecord(rev.RecordID)
			if err == nil {
				records = append(records, *rec)
			}
			seen[rev.RecordID] = true
		}
	}

	return revs, records, nil, nil
}

func (u *e2eSyncUseCase) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return nil, nil
}

func (u *e2eSyncUseCase) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	return nil, nil
}

// e2eAuthService — stub auth service for gRPC
type e2eAuthService struct {
	pbv1.UnimplementedAuthServiceServer
}

func (e *e2eAuthService) Register(ctx context.Context, req *pbv1.RegisterRequest) (*pbv1.RegisterResponse, error) {
	return &pbv1.RegisterResponse{UserId: 1}, nil
}

func (e *e2eAuthService) Login(ctx context.Context, req *pbv1.LoginRequest) (*pbv1.LoginResponse, error) {
	return &pbv1.LoginResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}, nil
}

func (e *e2eAuthService) Refresh(ctx context.Context, req *pbv1.RefreshRequest) (*pbv1.RefreshResponse, error) {
	return &pbv1.RefreshResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}, nil
}

func (e *e2eAuthService) Logout(ctx context.Context, req *pbv1.LogoutRequest) (*pbv1.LogoutResponse, error) {
	return &pbv1.LogoutResponse{}, nil
}

// e2eUploadsService — stub uploads service
type e2eUploadsService struct {
	pbv1.UnimplementedUploadsServiceServer
}

// ===========================================================================
// E2E TESTS
// ===========================================================================

// TestMetadataE2E_CreateAndGet проверяет полный roundtrip metadata
// через gRPC: clientcore -> gRPC transport -> server -> repository -> обратно.
func TestMetadataE2E_CreateAndGet(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	rec := &models.Record{
		Type:     models.RecordTypeText,
		Name:     "my secret note",
		Metadata: "important metadata for e2e",
		Payload:  models.TextPayload{Content: "secret content"},
		DeviceID: "e2e-test-device",
	}

	// Create record with metadata through clientcore
	result, err := env.core.SaveRecord(ctx, rec)
	require.NoError(t, err)
	require.NotZero(t, result.ID)
	require.Equal(t, "important metadata for e2e", result.Metadata, "metadata must survive create roundtrip")

	// Get record back through clientcore
	got, err := env.core.GetRecord(ctx, result.ID)
	require.NoError(t, err)
	require.Equal(t, "important metadata for e2e", got.Metadata, "metadata must survive get roundtrip")

	// Verify payload is also intact
	tp, ok := got.Payload.(models.TextPayload)
	require.True(t, ok)
	require.Equal(t, "secret content", tp.Content)
}

// TestMetadataE2E_UpdateMetadata проверяет обновление metadata через gRPC roundtrip.
func TestMetadataE2E_UpdateMetadata(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	// Create record with initial metadata
	rec := &models.Record{
		Type:     models.RecordTypeLogin,
		Name:     "my login",
		Metadata: "initial metadata",
		Payload:  models.LoginPayload{Login: "user@example.com", Password: "secret"},
		DeviceID: "e2e-test-device",
	}

	result, err := env.core.SaveRecord(ctx, rec)
	require.NoError(t, err)
	require.Equal(t, "initial metadata", result.Metadata)

	// Update metadata (bump revision for optimistic concurrency)
	result.Metadata = "updated metadata value"
	result.Name = "my login updated"
	result.Revision++

	updated, err := env.core.SaveRecord(ctx, result)
	require.NoError(t, err)
	require.Equal(t, "updated metadata value", updated.Metadata, "metadata must be updated")

	// Verify via Get
	got, err := env.core.GetRecord(ctx, result.ID)
	require.NoError(t, err)
	require.Equal(t, "updated metadata value", got.Metadata, "metadata must persist after update")
}

// TestMetadataE2E_ClearMetadata проверяет очистку metadata.
func TestMetadataE2E_ClearMetadata(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	rec := &models.Record{
		Type:     models.RecordTypeText,
		Name:     "temp note",
		Metadata: "will be cleared",
		Payload:  models.TextPayload{Content: "data"},
		DeviceID: "e2e-test-device",
	}

	result, err := env.core.SaveRecord(ctx, rec)
	require.NoError(t, err)
	require.Equal(t, "will be cleared", result.Metadata)

	// Clear metadata (bump revision for optimistic concurrency)
	result.Metadata = ""
	result.Revision++
	updated, err := env.core.SaveRecord(ctx, result)
	require.NoError(t, err)
	require.Equal(t, "", updated.Metadata, "metadata must be cleared")

	// Verify via Get
	got, err := env.core.GetRecord(ctx, result.ID)
	require.NoError(t, err)
	require.Equal(t, "", got.Metadata, "metadata must remain cleared")
}

// TestMetadataE2E_ListRecords проверяет что metadata корректно возвращается при list.
func TestMetadataE2E_ListRecords(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	// Create two records with different metadata
	rec1 := &models.Record{
		Type:     models.RecordTypeText,
		Name:     "note 1",
		Metadata: "first metadata",
		Payload:  models.TextPayload{Content: "content 1"},
		DeviceID: "e2e-test-device",
	}
	rec2 := &models.Record{
		Type:     models.RecordTypeText,
		Name:     "note 2",
		Metadata: "second metadata",
		Payload:  models.TextPayload{Content: "content 2"},
		DeviceID: "e2e-test-device",
	}

	r1, err := env.core.SaveRecord(ctx, rec1)
	require.NoError(t, err)
	r2, err := env.core.SaveRecord(ctx, rec2)
	require.NoError(t, err)

	// Save to cache so ListRecords can find them
	env.store.Records().Put(r1)
	env.store.Records().Put(r2)

	// List all records
	records, err := env.core.ListRecords(ctx, "")
	require.NoError(t, err)
	require.Len(t, records, 2)

	// Verify metadata is present in list results
	metaMap := map[int64]string{r1.ID: "first metadata", r2.ID: "second metadata"}
	for _, r := range records {
		expected, ok := metaMap[r.ID]
		require.True(t, ok, "unexpected record ID %d", r.ID)
		require.Equal(t, expected, r.Metadata, "metadata must match for record %d", r.ID)
	}
}

// TestMetadataE2E_SyncPushPull проверяет что metadata не теряется при sync.
func TestMetadataE2E_SyncPushPull(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	// Create a record directly in the repository (simulating server-side creation)
	rec := &models.Record{
		UserID:   1,
		Type:     models.RecordTypeCard,
		Name:     "visa card",
		Metadata: "sync metadata test",
		Payload: models.CardPayload{
			Number:     "4111111111111111",
			HolderName: "John Doe",
			ExpiryDate: "12/30",
			CVV:        "123",
		},
		DeviceID:   "e2e-test-device",
		KeyVersion: 1,
	}
	err := env.repo.CreateRecord(rec)
	require.NoError(t, err)

	// Create a revision for this record
	rev := &models.RecordRevision{
		UserID:   1,
		RecordID: rec.ID,
		Revision: 1,
		DeviceID: "e2e-test-device",
	}
	err = env.syncRepo.CreateRevision(rev)
	require.NoError(t, err)

	// Pull from server via sync transport
	pullResult, err := env.client.Pull(ctx, 0, "e2e-test-device", 100)
	require.NoError(t, err)
	require.Len(t, pullResult.Records, 1)

	// Verify metadata survived sync pull
	pulled := pullResult.Records[0]
	require.Equal(t, "sync metadata test", pulled.Metadata, "metadata must survive sync pull")
	require.Equal(t, "visa card", pulled.Name)

	// Verify payload
	cp, ok := pulled.Payload.(models.CardPayload)
	require.True(t, ok)
	require.Equal(t, "4111111111111111", cp.Number)
	require.Equal(t, "John Doe", cp.HolderName)
}

// TestMetadataE2E_MultilineMetadata проверяет многострочную metadata.
func TestMetadataE2E_MultilineMetadata(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	multilineMeta := "line1\nline2\nline3 with special chars: &<>\"'"

	rec := &models.Record{
		Type:     models.RecordTypeText,
		Name:     "multiline meta",
		Metadata: multilineMeta,
		Payload:  models.TextPayload{Content: "data"},
		DeviceID: "e2e-test-device",
	}

	result, err := env.core.SaveRecord(ctx, rec)
	require.NoError(t, err)
	require.Equal(t, multilineMeta, result.Metadata)

	got, err := env.core.GetRecord(ctx, result.ID)
	require.NoError(t, err)
	require.Equal(t, multilineMeta, got.Metadata, "multiline metadata must survive roundtrip")
}

// TestMetadataE2E_AllRecordTypes проверяет metadata для всех типов записей.
func TestMetadataE2E_AllRecordTypes(t *testing.T) {
	env := setupE2E(t)
	ctx := context.Background()

	tests := []struct {
		name string
		rec  *models.Record
		meta string
	}{
		{
			name: "login with metadata",
			rec: &models.Record{
				Type: models.RecordTypeLogin, Name: "login1",
				Payload:  models.LoginPayload{Login: "u", Password: "p"},
				DeviceID: "e2e-test-device",
			},
			meta: "login metadata",
		},
		{
			name: "text with metadata",
			rec: &models.Record{
				Type: models.RecordTypeText, Name: "text1",
				Payload:  models.TextPayload{Content: "hello"},
				DeviceID: "e2e-test-device",
			},
			meta: "text metadata",
		},
		{
			name: "card with metadata",
			rec: &models.Record{
				Type: models.RecordTypeCard, Name: "card1",
				Payload:  models.CardPayload{Number: "4111111111111111", HolderName: "Test", ExpiryDate: "12/30", CVV: "123"},
				DeviceID: "e2e-test-device",
			},
			meta: "card metadata",
		},
		{
			name: "binary with metadata",
			rec: &models.Record{
				Type: models.RecordTypeBinary, Name: "bin1",
				Payload:  models.BinaryPayload{},
				DeviceID: "e2e-test-device",
			},
			meta: "binary metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.rec.Metadata = tt.meta
			result, err := env.core.SaveRecord(ctx, tt.rec)
			require.NoError(t, err)
			require.Equal(t, tt.meta, result.Metadata, "metadata must survive create for %s", tt.name)

			got, err := env.core.GetRecord(ctx, result.ID)
			require.NoError(t, err)
			require.Equal(t, tt.meta, got.Metadata, "metadata must survive get for %s", tt.name)
		})
	}
}
