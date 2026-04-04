package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

type RecordUseCase interface {
	CreateRecord(record *models.Record) error
	GetRecord(id int64) (*models.Record, error)
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64, deviceID string) error
}

// DataService реализует gRPC-ручки для записей.
type DataService struct {
	pbv1.UnimplementedDataServiceServer
	records RecordUseCase
	log     zerolog.Logger
}

// NewDataService создаёт DataService.
func NewDataService(records RecordUseCase, log zerolog.Logger) *DataService {
	return &DataService{
		records: records,
		log:     log,
	}
}

func (s *DataService) CreateRecord(ctx context.Context, req *pbv1.CreateRecordRequest) (*pbv1.CreateRecordResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}

	recordType, payload, err := protoPayloadToDomain(req.GetPayload())
	if err != nil {
		return nil, err
	}

	record := &models.Record{
		UserID:         userID,
		Type:           recordType,
		Name:           req.Name,
		Metadata:       req.Metadata,
		DeviceID:       req.DeviceId,
		PayloadVersion: req.PayloadVersion,
		Payload:        payload,
	}

	if err := s.records.CreateRecord(record); err != nil {
		return nil, mapRecordError(err)
	}

	return &pbv1.CreateRecordResponse{
		Record: domainRecordToProto(record),
	}, nil
}

func (s *DataService) GetRecord(ctx context.Context, req *pbv1.GetRecordRequest) (*pbv1.GetRecordResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	record, err := s.records.GetRecord(req.Id)
	if err != nil {
		return nil, mapRecordError(err)
	}

	if record.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	return &pbv1.GetRecordResponse{
		Record: domainRecordToProto(record),
	}, nil
}

func (s *DataService) ListRecords(ctx context.Context, req *pbv1.ListRecordsRequest) (*pbv1.ListRecordsResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	recordType := protoTypeToDomain(req.GetType())
	records, err := s.records.ListRecords(userID, recordType, req.GetIncludeDeleted())
	if err != nil {
		return nil, mapRecordError(err)
	}

	pbRecords := make([]*pbv1.Record, 0, len(records))
	for i := range records {
		pbRecords = append(pbRecords, domainRecordToProto(&records[i]))
	}

	return &pbv1.ListRecordsResponse{Records: pbRecords}, nil
}

func (s *DataService) UpdateRecord(ctx context.Context, req *pbv1.UpdateRecordRequest) (*pbv1.UpdateRecordResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}
	if req.Revision <= 0 {
		return nil, status.Error(codes.InvalidArgument, "revision is required")
	}

	existing, err := s.records.GetRecord(req.Id)
	if err != nil {
		return nil, mapRecordError(err)
	}

	if existing.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	if existing.IsDeleted() {
		return nil, status.Error(codes.FailedPrecondition, "record is deleted")
	}

	currentRevision := existing.Revision
	if req.Revision <= currentRevision {
		return nil, status.Error(codes.Aborted, "revision conflict")
	}

	payload, err := protoPayloadToDomainForType(existing.Type, req.GetPayload())
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Metadata = req.Metadata
	existing.DeviceID = req.DeviceId
	existing.KeyVersion = req.KeyVersion
	existing.PayloadVersion = req.PayloadVersion
	existing.Payload = payload

	if err := existing.BumpRevision(req.Revision, req.DeviceId); err != nil {
		return nil, mapRecordError(err)
	}

	if err := s.records.UpdateRecord(existing); err != nil {
		return nil, mapRecordError(err)
	}

	return &pbv1.UpdateRecordResponse{
		Record: domainRecordToProto(existing),
	}, nil
}

func (s *DataService) DeleteRecord(ctx context.Context, req *pbv1.DeleteRecordRequest) (*pbv1.DeleteRecordResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}

	record, err := s.records.GetRecord(req.Id)
	if err != nil {
		if errors.Is(err, models.ErrRecordNotFound) {
			return &pbv1.DeleteRecordResponse{}, nil
		}
		return nil, mapRecordError(err)
	}

	if record.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}
	if record.IsDeleted() {
		return &pbv1.DeleteRecordResponse{}, nil
	}

	if err := s.records.DeleteRecord(req.Id, req.DeviceId); err != nil {
		return nil, mapRecordError(err)
	}

	return &pbv1.DeleteRecordResponse{}, nil
}

func userIDFromContext(ctx context.Context) (int64, error) {
	userID, ok := middlewares.UserIDFromContext(ctx)
	if !ok || userID <= 0 {
		return 0, status.Error(codes.Unauthenticated, "authorization required")
	}
	return userID, nil
}

func protoPayloadToDomain(payload interface{}) (models.RecordType, models.RecordPayload, error) {
	switch p := payload.(type) {
	case *pbv1.CreateRecordRequest_Login:
		return models.RecordTypeLogin, models.LoginPayload{Login: p.Login.GetLogin(), Password: p.Login.GetPassword()}, nil
	case *pbv1.CreateRecordRequest_Text:
		return models.RecordTypeText, models.TextPayload{Content: p.Text.GetContent()}, nil
	case *pbv1.CreateRecordRequest_Binary:
		return models.RecordTypeBinary, models.BinaryPayload{}, nil
	case *pbv1.CreateRecordRequest_Card:
		card := models.CardPayload{
			Number: p.Card.GetNumber(), HolderName: p.Card.GetHolderName(),
			ExpiryDate: p.Card.GetExpiryDate(), CVV: p.Card.GetCvv(),
		}
		if err := card.Validate(); err != nil {
			return "", nil, status.Errorf(codes.InvalidArgument, "card validation failed: %s", err.Error())
		}
		return models.RecordTypeCard, card, nil
	default:
		return "", nil, status.Error(codes.InvalidArgument, "payload is required")
	}
}

func protoPayloadToDomainForType(rt models.RecordType, payload interface{}) (models.RecordPayload, error) {
	switch p := payload.(type) {
	case *pbv1.UpdateRecordRequest_Login:
		if rt != models.RecordTypeLogin {
			return nil, status.Error(codes.InvalidArgument, "payload type does not match record type")
		}
		return models.LoginPayload{Login: p.Login.GetLogin(), Password: p.Login.GetPassword()}, nil
	case *pbv1.UpdateRecordRequest_Text:
		if rt != models.RecordTypeText {
			return nil, status.Error(codes.InvalidArgument, "payload type does not match record type")
		}
		return models.TextPayload{Content: p.Text.GetContent()}, nil
	case *pbv1.UpdateRecordRequest_Binary:
		if rt != models.RecordTypeBinary {
			return nil, status.Error(codes.InvalidArgument, "payload type does not match record type")
		}
		return models.BinaryPayload{}, nil
	case *pbv1.UpdateRecordRequest_Card:
		if rt != models.RecordTypeCard {
			return nil, status.Error(codes.InvalidArgument, "payload type does not match record type")
		}
		card := models.CardPayload{
			Number: p.Card.GetNumber(), HolderName: p.Card.GetHolderName(),
			ExpiryDate: p.Card.GetExpiryDate(), CVV: p.Card.GetCvv(),
		}
		if err := card.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "card validation failed: %s", err.Error())
		}
		return card, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "payload is required")
	}
}

func domainRecordToProto(r *models.Record) *pbv1.Record {
	if r == nil {
		return nil
	}
	pb := &pbv1.Record{
		Id:             r.ID,
		UserId:         r.UserID,
		Type:           domainTypeToProto(r.Type),
		Name:           r.Name,
		Metadata:       r.Metadata,
		Revision:       r.Revision,
		DeviceId:       r.DeviceID,
		KeyVersion:     r.KeyVersion,
		PayloadVersion: r.PayloadVersion,
		CreatedAt:      timeToProto(r.CreatedAt),
		UpdatedAt:      timeToProto(r.UpdatedAt),
	}

	if r.DeletedAt != nil {
		pb.DeletedAt = timeToProto(*r.DeletedAt)
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
			Number: p.Number, HolderName: p.HolderName,
			ExpiryDate: p.ExpiryDate, Cvv: p.CVV,
		}}
	}

	return pb
}

func domainTypeToProto(rt models.RecordType) pbv1.RecordType {
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

func protoTypeToDomain(rt pbv1.RecordType) models.RecordType {
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

func timeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func mapRecordError(err error) error {
	switch {
	case errors.Is(err, models.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrAlreadyDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrRevisionConflict):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, models.ErrRevisionNotMonotonic):
		return status.Error(codes.InvalidArgument, err.Error())
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
	case errors.Is(err, models.ErrInvalidPayloadVersion):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrInvalidUserID):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
