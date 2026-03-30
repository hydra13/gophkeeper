package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	grpcc "github.com/hydra13/gophkeeper/pkg/apiclient/grpc"
	"github.com/hydra13/gophkeeper/pkg/cache"
	clientcore "github.com/hydra13/gophkeeper/pkg/clientcore"
)

// NewCore creates a shared client core for CLI and TUI frontends.
func NewCore() (*clientcore.ClientCore, func(), error) {
	ctx := context.Background()
	addr := DefaultServerAddr()
	certFile := DefaultTLSCertFile()

	if certFile == "" {
		return nil, nil, fmt.Errorf("TLS certificate is required: set GK_TLS_CERT_FILE or run from project root (configs/certs/dev.crt)")
	}

	client, err := grpcc.NewClient(ctx, grpcc.Config{
		Address:     addr,
		TLSCertFile: certFile,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to server: %w", err)
	}

	store, err := cache.NewFileStore(DefaultCacheDir())
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("init cache: %w", err)
	}

	core := clientcore.New(client, store, clientcore.Config{
		DeviceID: "client-" + Hostname(),
	})
	core.RestoreAuth()

	cleanup := func() {
		store.Flush()
		client.Close()
	}

	return core, cleanup, nil
}

func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gophkeeper", "cache")
}

func DefaultServerAddr() string {
	if v := os.Getenv("GK_GRPC_ADDRESS"); v != "" {
		return v
	}
	return "localhost:9090"
}

func DefaultTLSCertFile() string {
	if v := os.Getenv("GK_TLS_CERT_FILE"); v != "" {
		return v
	}
	if _, err := os.Stat("configs/certs/dev.crt"); err == nil {
		return "configs/certs/dev.crt"
	}
	return ""
}

func Hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}
