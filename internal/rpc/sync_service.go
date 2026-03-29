package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// SyncUseCase определяет операции синхронизации для gRPC слоя.
type SyncUseCase interface {
	Push(userID int64, deviceID string, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
	Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
	GetConflicts(userID int64) ([]models.SyncConflict, error)
	ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error)
}

// PendingChange описывает одно локальное изменение для push.
type PendingChange struct {
	Record        *models.Record
	Deleted       bool
	BaseRevision  int64
}

// SyncService gRPC имплементация.
type SyncService struct {
	pbv1.UnimplementedSyncServiceServer
	useCase SyncUseCase
	log     zerolog.Logger
}

// NewSyncService создаёт SyncService с зависимостями.
func NewSyncService(useCase SyncUseCase, log zerolog.Logger) *SyncService {
	return &SyncService{
		useCase: useCase,
		log:     log,
	}
}

func (s *SyncService) Push(ctx context.Context, req *pbv1.PushRequest) (*pbv1.PushResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}
	if len(req.Changes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "changes are required")
	}

	changes := make([]PendingChange, 0, len(req.Changes))
	for _, pc := range req.Changes {
		if pc.Record == nil {
			return nil, status.Error(codes.InvalidArgument, "record is required in pending change")
		}
		changes = append(changes, PendingChange{
			Record:       protoRecordToDomain(pc.Record),
			Deleted:      pc.Deleted,
			BaseRevision: pc.BaseRevision,
		})
	}

	accepted, conflicts, err := s.useCase.Push(userID, req.DeviceId, changes)
	if err != nil {
		return nil, mapSyncError(err)
	}

	resp := &pbv1.PushResponse{
		Accepted:  domainRevisionsToProto(accepted),
		Conflicts: domainConflictsToProto(conflicts),
	}
	return resp, nil
}

func (s *SyncService) Pull(ctx context.Context, req *pbv1.PullRequest) (*pbv1.PullResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}

	limit := int64(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}

	revisions, records, conflicts, err := s.useCase.Pull(userID, req.DeviceId, req.SinceRevision, limit)
	if err != nil {
		return nil, mapSyncError(err)
	}

	hasMore := int64(len(revisions)) == limit
	var nextRevision int64
	if len(revisions) > 0 {
		nextRevision = revisions[len(revisions)-1].Revision
	}

	resp := &pbv1.PullResponse{
		Changes:      domainRevisionsToProto(revisions),
		Records:      domainRecordsToProto(records),
		HasMore:      hasMore,
		NextRevision: nextRevision,
		Conflicts:    domainConflictsToProto(conflicts),
	}
	return resp, nil
}

func (s *SyncService) GetConflicts(ctx context.Context, req *pbv1.GetConflictsRequest) (*pbv1.GetConflictsResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	conflicts, err := s.useCase.GetConflicts(userID)
	if err != nil {
		return nil, mapSyncError(err)
	}

	return &pbv1.GetConflictsResponse{
		Conflicts: domainConflictsToProto(conflicts),
	}, nil
}

func (s *SyncService) ResolveConflict(ctx context.Context, req *pbv1.ResolveConflictRequest) (*pbv1.ResolveConflictResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.ConflictId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "conflict_id is required")
	}

	resolution := protoResolutionToDomain(req.Resolution)
	if resolution == "" {
		return nil, status.Error(codes.InvalidArgument, "resolution is required")
	}

	record, err := s.useCase.ResolveConflict(userID, req.ConflictId, resolution)
	if err != nil {
		return nil, mapSyncError(err)
	}

	return &pbv1.ResolveConflictResponse{
		Record: domainRecordToProto(record),
	}, nil
}

func protoRecordToDomain(pb *pbv1.Record) *models.Record {
	if pb == nil {
		return nil
	}
	r := &models.Record{
		ID:             pb.Id,
		UserID:         pb.UserId,
		Type:           protoTypeToDomain(pb.Type),
		Name:           pb.Name,
		Metadata:       pb.Metadata,
		Revision:       pb.Revision,
		DeviceID:       pb.DeviceId,
		KeyVersion:     pb.KeyVersion,
		PayloadVersion: pb.PayloadVersion,
		CreatedAt:      protoToTime(pb.CreatedAt),
		UpdatedAt:      protoToTime(pb.UpdatedAt),
	}
	if pb.DeletedAt != nil {
		t := protoToTime(pb.DeletedAt)
		r.DeletedAt = &t
	}

	switch p := pb.Payload.(type) {
	case *pbv1.Record_Login:
		r.Payload = models.LoginPayload{Login: p.Login.GetLogin(), Password: p.Login.GetPassword()}
	case *pbv1.Record_Text:
		r.Payload = models.TextPayload{Content: p.Text.GetContent()}
	case *pbv1.Record_Binary:
		r.Payload = models.BinaryPayload{}
	case *pbv1.Record_Card:
		r.Payload = models.CardPayload{
			Number: p.Card.GetNumber(), HolderName: p.Card.GetHolderName(),
			ExpiryDate: p.Card.GetExpiryDate(), CVV: p.Card.GetCvv(),
		}
	}

	return r
}

func protoResolutionToDomain(r pbv1.ConflictResolution) string {
	switch r {
	case pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL:
		return models.ConflictResolutionLocal
	case pbv1.ConflictResolution_CONFLICT_RESOLUTION_SERVER:
		return models.ConflictResolutionServer
	default:
		return ""
	}
}

func domainResolutionToProto(r string) pbv1.ConflictResolution {
	switch r {
	case models.ConflictResolutionLocal:
		return pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL
	case models.ConflictResolutionServer:
		return pbv1.ConflictResolution_CONFLICT_RESOLUTION_SERVER
	default:
		return pbv1.ConflictResolution_CONFLICT_RESOLUTION_UNSPECIFIED
	}
}

func domainRevisionsToProto(revs []models.RecordRevision) []*pbv1.RecordRevision {
	result := make([]*pbv1.RecordRevision, 0, len(revs))
	for _, rev := range revs {
		result = append(result, &pbv1.RecordRevision{
			Id:       rev.ID,
			RecordId: rev.RecordID,
			UserId:   rev.UserID,
			Revision: rev.Revision,
			DeviceId: rev.DeviceID,
		})
	}
	return result
}

func domainRecordsToProto(records []models.Record) []*pbv1.Record {
	result := make([]*pbv1.Record, 0, len(records))
	for i := range records {
		result = append(result, domainRecordToProto(&records[i]))
	}
	return result
}

func domainConflictsToProto(conflicts []models.SyncConflict) []*pbv1.SyncConflict {
	result := make([]*pbv1.SyncConflict, 0, len(conflicts))
	for _, c := range conflicts {
		pb := &pbv1.SyncConflict{
			Id:             c.ID,
			UserId:         c.UserID,
			RecordId:       c.RecordID,
			LocalRevision:  c.LocalRevision,
			ServerRevision: c.ServerRevision,
			Resolved:       c.Resolved,
			Resolution:     domainResolutionToProto(c.Resolution),
		}
		if c.LocalRecord != nil {
			pb.LocalRecord = domainRecordToProto(c.LocalRecord)
		}
		if c.ServerRecord != nil {
			pb.ServerRecord = domainRecordToProto(c.ServerRecord)
		}
		result = append(result, pb)
	}
	return result
}

func protoToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// mapSyncError маппит доменные ошибки синхронизации в gRPC status codes.
func mapSyncError(err error) error {
	switch {
	case errors.Is(err, models.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrRevisionConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, models.ErrConflictAlreadyResolved):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrInvalidConflictResolution):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrAlreadyDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrEmptyRecordName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrEmptyDeviceID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrInvalidRecordType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrNilPayload):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrInvalidKeyVersion):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
