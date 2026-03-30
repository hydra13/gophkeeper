package rpc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// stubServer embeds all Unimplemented*Server types to satisfy every
// service interface required by NewServer in a single struct.
type stubServer struct {
	pbv1.UnimplementedAuthServiceServer
	pbv1.UnimplementedDataServiceServer
	pbv1.UnimplementedSyncServiceServer
	pbv1.UnimplementedUploadsServiceServer
	pbv1.UnimplementedHealthServiceServer
}

func TestNewServer(t *testing.T) {
	s := stubServer{}

	srv := NewServer(
		s,
		s,
		s,
		s,
		s,
	)
	require.NotNil(t, srv)
}

func TestRegister_RegistersAllServices(t *testing.T) {
	s := stubServer{}

	srv := NewServer(
		s,
		s,
		s,
		s,
		s,
	)

	g := grpc.NewServer()
	srv.Register(g)

	info := g.GetServiceInfo()

	expected := map[string]string{
		"gophkeeper.v1.AuthService":   "auth",
		"gophkeeper.v1.DataService":   "data",
		"gophkeeper.v1.SyncService":   "sync",
		"gophkeeper.v1.UploadsService": "uploads",
		"gophkeeper.v1.HealthService": "health",
	}

	for serviceName, label := range expected {
		t.Run(label, func(t *testing.T) {
			si, ok := info[serviceName]
			require.True(t, ok, "service %s not registered", serviceName)
			require.NotEmpty(t, si.Methods, "service %s has no methods", serviceName)
		})
	}

	// Verify total count — only the 5 services above plus the grpc health
	// v1 that may be auto-registered. The exact count should be at least 5.
	require.GreaterOrEqual(t, len(info), 5)
}

func TestRegister_AllServiceMethodsPresent(t *testing.T) {
	s := stubServer{}

	srv := NewServer(s, s, s, s, s)
	g := grpc.NewServer()
	srv.Register(g)

	info := g.GetServiceInfo()

	// AuthService methods.
	authMethods := info["gophkeeper.v1.AuthService"].Methods
	requireMethodRegistered(t, authMethods, "Register")
	requireMethodRegistered(t, authMethods, "Login")
	requireMethodRegistered(t, authMethods, "Refresh")
	requireMethodRegistered(t, authMethods, "Logout")

	// DataService methods.
	dataMethods := info["gophkeeper.v1.DataService"].Methods
	requireMethodRegistered(t, dataMethods, "CreateRecord")
	requireMethodRegistered(t, dataMethods, "GetRecord")
	requireMethodRegistered(t, dataMethods, "ListRecords")
	requireMethodRegistered(t, dataMethods, "UpdateRecord")
	requireMethodRegistered(t, dataMethods, "DeleteRecord")

	// SyncService methods.
	syncMethods := info["gophkeeper.v1.SyncService"].Methods
	requireMethodRegistered(t, syncMethods, "Pull")
	requireMethodRegistered(t, syncMethods, "Push")
	requireMethodRegistered(t, syncMethods, "GetConflicts")
	requireMethodRegistered(t, syncMethods, "ResolveConflict")

	// HealthService methods.
	healthMethods := info["gophkeeper.v1.HealthService"].Methods
	requireMethodRegistered(t, healthMethods, "HealthCheck")

	// UploadsService: streams are in the Streams field, regular methods in Methods.
	uploadsMethods := info["gophkeeper.v1.UploadsService"].Methods
	requireMethodRegistered(t, uploadsMethods, "CreateUploadSession")
	requireMethodRegistered(t, uploadsMethods, "GetUploadStatus")
	requireMethodRegistered(t, uploadsMethods, "CreateDownloadSession")
	requireMethodRegistered(t, uploadsMethods, "ConfirmChunk")
	requireMethodRegistered(t, uploadsMethods, "GetDownloadStatus")
}

func requireMethodRegistered(t *testing.T, methods []grpc.MethodInfo, name string) {
	t.Helper()
	for _, m := range methods {
		if m.Name == name {
			return
		}
	}
	t.Errorf("method %s not registered", name)
}
