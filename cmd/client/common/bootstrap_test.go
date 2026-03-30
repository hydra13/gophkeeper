package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultServerAddrUsesEnv(t *testing.T) {
	t.Setenv("GK_GRPC_ADDRESS", "example.org:9999")

	if got := DefaultServerAddr(); got != "example.org:9999" {
		t.Fatalf("DefaultServerAddr() = %q, want %q", got, "example.org:9999")
	}
}

func TestDefaultTLSCertFileUsesEnv(t *testing.T) {
	t.Setenv("GK_TLS_CERT_FILE", "/tmp/custom.crt")

	if got := DefaultTLSCertFile(); got != "/tmp/custom.crt" {
		t.Fatalf("DefaultTLSCertFile() = %q, want %q", got, "/tmp/custom.crt")
	}
}

func TestDefaultTLSCertFileFindsProjectCert(t *testing.T) {
	t.Setenv("GK_TLS_CERT_FILE", "")

	dir := t.TempDir()
	certDir := filepath.Join(dir, "configs", "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "dev.crt"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd(): %v", err)
	}
	defer os.Chdir(oldWD)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(): %v", err)
	}

	if got := DefaultTLSCertFile(); got != "configs/certs/dev.crt" {
		t.Fatalf("DefaultTLSCertFile() = %q, want %q", got, "configs/certs/dev.crt")
	}
}
