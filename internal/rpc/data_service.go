package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DataService gRPC имплементация.
type DataService struct {
	pbv1.UnimplementedDataServiceServer
}

// NewDataService создаёт заглушку DataService.
func NewDataService() *DataService {
	return &DataService{}
}

func (s *DataService) CreateRecord(context.Context, *pbv1.CreateRecordRequest) (*pbv1.CreateRecordResponse, error) {
	return nil, status.Error(codes.Unimplemented, "data service not implemented")
}

func (s *DataService) ListRecords(context.Context, *pbv1.ListRecordsRequest) (*pbv1.ListRecordsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "data service not implemented")
}

func (s *DataService) GetRecord(context.Context, *pbv1.GetRecordRequest) (*pbv1.GetRecordResponse, error) {
	return nil, status.Error(codes.Unimplemented, "data service not implemented")
}

func (s *DataService) UpdateRecord(context.Context, *pbv1.UpdateRecordRequest) (*pbv1.UpdateRecordResponse, error) {
	return nil, status.Error(codes.Unimplemented, "data service not implemented")
}

func (s *DataService) DeleteRecord(context.Context, *pbv1.DeleteRecordRequest) (*pbv1.DeleteRecordResponse, error) {
	return nil, status.Error(codes.Unimplemented, "data service not implemented")
}
