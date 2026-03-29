package grpc

import (
	"context"
	"fmt"
	"io"

	apiclient "github.com/hydra13/gophkeeper/pkg/apiclient"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client — gRPC-реализация Transport.
type Client struct {
	authConn    *grpc.ClientConn
	authClient  pbv1.AuthServiceClient
	dataClient  pbv1.DataServiceClient
	syncClient  pbv1.SyncServiceClient
	uploadClient pbv1.UploadsServiceClient
	accessToken string
}

// Config — параметры подключения gRPC-клиента.
type Config struct {
	Address      string
	TLSCertFile  string // пустая строка = insecure
	AccessToken  string
}

// NewClient создаёт gRPC-клиент и подключается к серверу.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	var opts []grpc.DialOption

	if cfg.TLSCertFile != "" {
		creds, err := credentials.NewClientTLSFromFile(cfg.TLSCertFile, "")
		if err != nil {
			return nil, fmt.Errorf("load TLS cert: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	c := &Client{
		authConn:     conn,
		authClient:   pbv1.NewAuthServiceClient(conn),
		dataClient:   pbv1.NewDataServiceClient(conn),
		syncClient:   pbv1.NewSyncServiceClient(conn),
		uploadClient: pbv1.NewUploadsServiceClient(conn),
		accessToken:  cfg.AccessToken,
	}

	return c, nil
}

// Close закрывает gRPC-соединение.
func (c *Client) Close() error {
	if c.authConn != nil {
		return c.authConn.Close()
	}
	return nil
}

// SetAccessToken устанавливает access-токен для авторизации запросов.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// authContext возвращает контекст с access-токеном в metadata.
func (c *Client) authContext(ctx context.Context) context.Context {
	if c.accessToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.accessToken)
}

// Register регистрирует нового пользователя.
func (c *Client) Register(ctx context.Context, email, password string) (int64, error) {
	resp, err := c.authClient.Register(ctx, &pbv1.RegisterRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return 0, fmt.Errorf("grpc register: %w", err)
	}
	return resp.UserId, nil
}

// Login аутентифицирует пользователя.
func (c *Client) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	resp, err := c.authClient.Login(ctx, &pbv1.LoginRequest{
		Email:      email,
		Password:   password,
		DeviceId:   deviceID,
		DeviceName: deviceName,
		ClientType: clientType,
	})
	if err != nil {
		return "", "", fmt.Errorf("grpc login: %w", err)
	}
	c.accessToken = resp.AccessToken
	return resp.AccessToken, resp.RefreshToken, nil
}

// Refresh обновляет пару токенов.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	resp, err := c.authClient.Refresh(ctx, &pbv1.RefreshRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return "", "", fmt.Errorf("grpc refresh: %w", err)
	}
	c.accessToken = resp.AccessToken
	return resp.AccessToken, resp.RefreshToken, nil
}

// Logout отзывает текущую сессию.
func (c *Client) Logout(ctx context.Context) error {
	_, err := c.authClient.Logout(c.authContext(ctx), &pbv1.LogoutRequest{})
	if err != nil {
		return fmt.Errorf("grpc logout: %w", err)
	}
	c.accessToken = ""
	return nil
}

// CreateRecord создаёт запись на сервере.
func (c *Client) CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	pbRecord := domainToProtoRecord(record)
	pbPayload := domainToProtoPayload(record)

	req := &pbv1.CreateRecordRequest{
		Type:          pbRecord.Type,
		Name:          pbRecord.Name,
		Metadata:      pbRecord.Metadata,
		DeviceId:      pbRecord.DeviceId,
		KeyVersion:    pbRecord.KeyVersion,
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

	resp, err := c.dataClient.CreateRecord(c.authContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("grpc create record: %w", err)
	}
	return protoToDomainRecord(resp.Record), nil
}

// GetRecord получает запись по ID.
func (c *Client) GetRecord(ctx context.Context, id int64) (*models.Record, error) {
	resp, err := c.dataClient.GetRecord(c.authContext(ctx), &pbv1.GetRecordRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("grpc get record: %w", err)
	}
	return protoToDomainRecord(resp.Record), nil
}

// ListRecords получает список записей.
func (c *Client) ListRecords(ctx context.Context, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	resp, err := c.dataClient.ListRecords(c.authContext(ctx), &pbv1.ListRecordsRequest{
		Type:           domainToProtoRecordType(recordType),
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc list records: %w", err)
	}

	records := make([]models.Record, 0, len(resp.Records))
	for _, r := range resp.Records {
		records = append(records, *protoToDomainRecord(r))
	}
	return records, nil
}

// UpdateRecord обновляет запись.
func (c *Client) UpdateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	pbRecord := domainToProtoRecord(record)
	pbPayload := domainToProtoPayloadForUpdate(record)

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

	resp, err := c.dataClient.UpdateRecord(c.authContext(ctx), req)
	if err != nil {
		return nil, fmt.Errorf("grpc update record: %w", err)
	}
	return protoToDomainRecord(resp.Record), nil
}

// DeleteRecord удаляет запись.
func (c *Client) DeleteRecord(ctx context.Context, id int64) error {
	_, err := c.dataClient.DeleteRecord(c.authContext(ctx), &pbv1.DeleteRecordRequest{
		Id:       id,
		DeviceId: "", // будет заполнен на уровне clientcore
	})
	if err != nil {
		return fmt.Errorf("grpc delete record: %w", err)
	}
	return nil
}

// Pull получает изменения с сервера.
func (c *Client) Pull(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
	resp, err := c.syncClient.Pull(c.authContext(ctx), &pbv1.PullRequest{
		SinceRevision: sinceRevision,
		DeviceId:      deviceID,
		Limit:         limit,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc pull: %w", err)
	}

	result := &apiclient.PullResult{
		HasMore:      resp.HasMore,
		NextRevision: resp.NextRevision,
	}

	for _, r := range resp.Records {
		result.Records = append(result.Records, *protoToDomainRecord(r))
	}
	for _, conflict := range resp.Conflicts {
		result.Conflicts = append(result.Conflicts, protoToConflictInfo(conflict))
	}

	return result, nil
}

// Push отправляет локальные изменения на сервер.
func (c *Client) Push(ctx context.Context, changes []apiclient.PendingChange, deviceID string) (*apiclient.PushResult, error) {
	pbChanges := make([]*pbv1.PendingChange, 0, len(changes))
	for _, ch := range changes {
		pbChanges = append(pbChanges, &pbv1.PendingChange{
			Record:       domainToProtoRecord(ch.Record),
			Deleted:      ch.Deleted,
			BaseRevision: ch.BaseRevision,
		})
	}

	resp, err := c.syncClient.Push(c.authContext(ctx), &pbv1.PushRequest{
		Changes: pbChanges,
		DeviceId: deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc push: %w", err)
	}

	result := &apiclient.PushResult{}
	for _, a := range resp.Accepted {
		result.Accepted = append(result.Accepted, apiclient.AcceptedChange{
			RecordID: a.RecordId,
			Revision: a.Revision,
			DeviceID: a.DeviceId,
		})
	}
	for _, conflict := range resp.Conflicts {
		result.Conflicts = append(result.Conflicts, protoToConflictInfo(conflict))
	}

	return result, nil
}

// CreateUploadSession создаёт сессию загрузки.
func (c *Client) CreateUploadSession(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	resp, err := c.uploadClient.CreateUploadSession(c.authContext(ctx), &pbv1.CreateUploadSessionRequest{
		RecordId:    recordID,
		TotalChunks: totalChunks,
		ChunkSize:   chunkSize,
		TotalSize:   totalSize,
		KeyVersion:  keyVersion,
	})
	if err != nil {
		return 0, fmt.Errorf("grpc create upload session: %w", err)
	}
	return resp.UploadId, nil
}

// UploadChunk загружает чанк данных.
func (c *Client) UploadChunk(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
	stream, err := c.uploadClient.UploadChunk(c.authContext(ctx))
	if err != nil {
		return fmt.Errorf("grpc upload chunk stream: %w", err)
	}

	err = stream.Send(&pbv1.UploadChunkRequest{
		UploadId:   uploadID,
		ChunkIndex: chunkIndex,
		Data:       data,
	})
	if err != nil {
		return fmt.Errorf("grpc upload chunk send: %w", err)
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("grpc upload chunk close: %w", err)
	}
	return nil
}

// GetUploadStatus получает статус upload-сессии.
func (c *Client) GetUploadStatus(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error) {
	resp, err := c.uploadClient.GetUploadStatus(c.authContext(ctx), &pbv1.GetUploadStatusRequest{
		UploadId: uploadID,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc get upload status: %w", err)
	}
	return &apiclient.UploadStatus{
		UploadID:       resp.UploadId,
		Status:         resp.Status.String(),
		TotalChunks:    resp.TotalChunks,
		ReceivedChunks: resp.ReceivedChunks,
		MissingChunks:  resp.MissingChunks,
	}, nil
}

// CreateDownloadSession создаёт сессию скачивания.
func (c *Client) CreateDownloadSession(ctx context.Context, recordID int64) (int64, int64, error) {
	resp, err := c.uploadClient.CreateDownloadSession(c.authContext(ctx), &pbv1.CreateDownloadSessionRequest{
		RecordId: recordID,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("grpc create download session: %w", err)
	}
	return resp.DownloadId, resp.TotalChunks, nil
}

// DownloadChunk скачивает чанк данных.
func (c *Client) DownloadChunk(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
	stream, err := c.uploadClient.DownloadChunk(c.authContext(ctx), &pbv1.DownloadChunkRequest{
		DownloadId: downloadID,
		ChunkIndex: chunkIndex,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc download chunk: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("grpc download chunk recv: %w", err)
	}
	return resp.Data, nil
}

// ConfirmChunk подтверждает получение чанка.
func (c *Client) ConfirmChunk(ctx context.Context, downloadID, chunkIndex int64) error {
	_, err := c.uploadClient.ConfirmChunk(c.authContext(ctx), &pbv1.ConfirmChunkRequest{
		DownloadId: downloadID,
		ChunkIndex: chunkIndex,
	})
	if err != nil {
		return fmt.Errorf("grpc confirm chunk: %w", err)
	}
	return nil
}

// GetDownloadStatus получает статус download-сессии.
func (c *Client) GetDownloadStatus(ctx context.Context, downloadID int64) (*apiclient.DownloadStatus, error) {
	resp, err := c.uploadClient.GetDownloadStatus(c.authContext(ctx), &pbv1.GetDownloadStatusRequest{
		DownloadId: downloadID,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc get download status: %w", err)
	}
	return &apiclient.DownloadStatus{
		DownloadID:      resp.DownloadId,
		Status:          resp.Status.String(),
		TotalChunks:     resp.TotalChunks,
		ConfirmedChunks: resp.ConfirmedChunks,
		RemainingChunks: resp.RemainingChunks,
	}, nil
}

// --- Конвертация доменных моделей <-> protobuf ---

func domainToProtoRecordType(rt models.RecordType) pbv1.RecordType {
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

func protoToDomainRecordType(rt pbv1.RecordType) models.RecordType {
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

func domainToProtoRecord(r *models.Record) *pbv1.Record {
	if r == nil {
		return nil
	}
	pb := &pbv1.Record{
		Id:             r.ID,
		UserId:         r.UserID,
		Type:           domainToProtoRecordType(r.Type),
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

	if r.Payload != nil {
		switch p := r.Payload.(type) {
		case models.LoginPayload:
			pb.Payload = &pbv1.Record_Login{Login: &pbv1.LoginPayload{Login: p.Login, Password: p.Password}}
		case models.TextPayload:
			pb.Payload = &pbv1.Record_Text{Text: &pbv1.TextPayload{Content: p.Content}}
		case models.BinaryPayload:
			pb.Payload = &pbv1.Record_Binary{Binary: &pbv1.BinaryPayload{}}
		case models.CardPayload:
			pb.Payload = &pbv1.Record_Card{Card: &pbv1.CardPayload{
				Number:     p.Number,
				HolderName: p.HolderName,
				ExpiryDate: p.ExpiryDate,
				Cvv:        p.CVV,
			}}
		}
	}
	return pb
}

func domainToProtoPayload(r *models.Record) interface{} {
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

func domainToProtoPayloadForUpdate(r *models.Record) interface{} {
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

func protoToDomainRecord(pb *pbv1.Record) *models.Record {
	if pb == nil {
		return nil
	}
	r := &models.Record{
		ID:             pb.Id,
		UserID:         pb.UserId,
		Type:           protoToDomainRecordType(pb.Type),
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

func protoToConflictInfo(pb *pbv1.SyncConflict) apiclient.SyncConflictInfo {
	info := apiclient.SyncConflictInfo{
		ID:             pb.Id,
		RecordID:       pb.RecordId,
		LocalRevision:  pb.LocalRevision,
		ServerRevision: pb.ServerRevision,
		Resolved:       pb.Resolved,
	}
	if pb.LocalRecord != nil {
		rec := protoToDomainRecord(pb.LocalRecord)
		info.LocalRecord = rec
	}
	if pb.ServerRecord != nil {
		rec := protoToDomainRecord(pb.ServerRecord)
		info.ServerRecord = rec
	}
	return info
}
