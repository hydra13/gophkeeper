package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/cache"
	"github.com/hydra13/gophkeeper/pkg/clientcore"
	"github.com/hydra13/gophkeeper/pkg/clientcore/mocks"
)

// testEnv holds test environment with injected mock core.
type testEnv struct {
	core      *clientcore.ClientCore
	transport *mocks.MockTransport
	store     cache.Store
	cleanup   func()

	fatalErr error
	mu       sync.Mutex

	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

// setupTestEnv creates a test environment with mock transport and injected core factory.
// It replaces newCoreFunc, fatalFunc, readPasswordFunc, readLineFunc so that
// CLI entrypoints (runRegister, runLogin, etc.) can be called directly.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	store, err := cache.NewFileStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	transport := &mocks.MockTransport{}
	core := clientcore.New(transport, store, clientcore.Config{
		DeviceID: "test-device",
	})

	env := &testEnv{
		core:      core,
		transport: transport,
		store:     store,
		stdout:    &bytes.Buffer{},
		stderr:    &bytes.Buffer{},
		cleanup: func() {
			_ = store.Flush()
		},
	}

	// Inject mock core factory
	origNewCore := newCoreFunc
	origFatal := fatalFunc
	origReadPassword := readPasswordFunc
	origReadLine := readLineFunc

	newCoreFunc = func() (*clientcore.ClientCore, func(), error) {
		return env.core, env.cleanup, nil
	}
	fatalFunc = func(err error) {
		env.mu.Lock()
		env.fatalErr = err
		env.mu.Unlock()
		panic("fatal:" + err.Error())
	}
	readPasswordFunc = func(prompt string) string {
		return "test-password"
	}
	readLineFunc = func(prompt string) string {
		return "test-input"
	}

	t.Cleanup(func() {
		newCoreFunc = origNewCore
		fatalFunc = origFatal
		readPasswordFunc = origReadPassword
		readLineFunc = origReadLine
	})

	return env
}

// lastFatal returns the last fatal error captured by the test env.
func (e *testEnv) lastFatal() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.fatalErr
}

// captureOutput redirects stdout and stderr during fn execution.
func (e *testEnv) captureOutput(fn func()) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	e.stdout.Reset()
	e.stderr.Reset()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var buf bytes.Buffer
		buf.ReadFrom(rOut)
		e.stdout.Write(buf.Bytes())
	}()
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(rErr)
		e.stderr.Write(buf.Bytes())
	}()

	fn()

	wOut.Close()
	wErr.Close()
	<-done
}

// --- Smoke tests: direct entrypoint calls ---

func TestCLIRunRegister(t *testing.T) {
	env := setupTestEnv(t)

	env.captureOutput(func() {
		defer func() {
			recover() // catch fatal panic
		}()
		runRegister([]string{"user@example.com"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "registered successfully") {
		t.Fatalf("expected 'registered successfully' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunLogin(t *testing.T) {
	env := setupTestEnv(t)

	env.captureOutput(func() {
		defer func() { recover() }()
		runLogin([]string{"user@example.com"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "logged in successfully") {
		t.Fatalf("expected 'logged in successfully' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunLogout(t *testing.T) {
	env := setupTestEnv(t)
	// Login first so logout has auth
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.captureOutput(func() {
		defer func() { recover() }()
		runLogout()
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "logged out") {
		t.Fatalf("expected 'logged out' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunAddText(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.captureOutput(func() {
		defer func() { recover() }()
		// add text <name> <data>
		runAdd([]string{"text", "my note", "hello world"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	out := env.stdout.String()
	if !strings.Contains(out, "added:") {
		t.Fatalf("expected 'added:' in stdout, got: %s", out)
	}
}

func TestCLIRunAddLogin(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	// readLineFunc returns "test-input" for all prompts, readPasswordFunc returns "test-password"
	env.captureOutput(func() {
		defer func() { recover() }()
		// add login <name> — interactive prompts via readLine/readPassword
		runAdd([]string{"login", "my site"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "added:") {
		t.Fatalf("expected 'added:' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunAddBinary(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	// Create a temp file for binary upload
	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmpFile, []byte("binary content"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		// add binary <name> <file-path>
		runAdd([]string{"binary", "file.bin", tmpFile})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "added:") {
		t.Fatalf("expected 'added:' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunList(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	// Add a record so list is not empty
	rec := &models.Record{
		Type:    models.RecordTypeText,
		Name:    "list test",
		Payload: models.TextPayload{Content: "content"},
	}
	_, _ = env.core.SaveRecord(ctx, rec)

	// Mock ListRecords to return the saved record
	env.transport.ListRecordsFunc = func(ctx context.Context, rt models.RecordType, includeDeleted bool) ([]models.Record, error) {
		return []models.Record{
			{ID: 1, Type: models.RecordTypeText, Name: "list test", Revision: 1},
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runList(nil)
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	out := env.stdout.String()
	if !strings.Contains(out, "list test") {
		t.Fatalf("expected 'list test' in stdout, got: %s", out)
	}
}

func TestCLIRunListEmpty(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.captureOutput(func() {
		defer func() { recover() }()
		runList(nil)
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "no records") {
		t.Fatalf("expected 'no records' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunGet(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeLogin,
			Name:     "test site",
			Payload:  models.LoginPayload{Login: "user", Password: "pass"},
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runGet([]string{"42"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	out := env.stdout.String()
	if !strings.Contains(out, "test site") {
		t.Fatalf("expected 'test site' in stdout, got: %s", out)
	}
	if !strings.Contains(out, "ID:       42") {
		t.Fatalf("expected 'ID:       42' in stdout, got: %s", out)
	}
}

func TestCLIRunGetBinary(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeBinary,
			Name:     "file.bin",
			Payload:  models.BinaryPayload{},
			Revision: 1,
		}, nil
	}
	env.transport.CreateDownloadSessionFunc = func(ctx context.Context, recordID int64) (int64, int64, error) {
		return 1, 1, nil
	}
	env.transport.DownloadChunkFunc = func(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
		return []byte("file-content"), nil
	}

	outPath := filepath.Join(t.TempDir(), "output.bin")

	env.captureOutput(func() {
		defer func() { recover() }()
		runGet([]string{"10", outPath})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "saved") {
		t.Fatalf("expected 'saved' in stdout, got: %s", env.stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != "file-content" {
		t.Fatalf("expected 'file-content', got '%s'", string(data))
	}
}

func TestCLIRunUpdate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeLogin,
			Name:     "old-name",
			Payload:  models.LoginPayload{Login: "user", Password: "old"},
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		// update <id> <name> — prompts for payload via readLine/readPassword
		runUpdate([]string{"5", "new-name"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "updated:") {
		t.Fatalf("expected 'updated:' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunUpdateBinary(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeBinary,
			Name:     "old-name",
			Payload:  models.BinaryPayload{},
			Revision: 1,
		}, nil
	}

	tmpFile := filepath.Join(t.TempDir(), "update.bin")
	if err := os.WriteFile(tmpFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runUpdate([]string{"5", "new-name", tmpFile})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "updated:") {
		t.Fatalf("expected 'updated:' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunDelete(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.captureOutput(func() {
		defer func() { recover() }()
		runDelete([]string{"1"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "deleted") {
		t.Fatalf("expected 'deleted' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunSync(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.captureOutput(func() {
		defer func() { recover() }()
		runSync()
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if !strings.Contains(env.stdout.String(), "synced") {
		t.Fatalf("expected 'synced' in stdout, got: %s", env.stdout.String())
	}
}

func TestCLIRunRegisterError(t *testing.T) {
	env := setupTestEnv(t)
	env.transport.RegisterFunc = func(ctx context.Context, email, password string) (int64, error) {
		return 0, fmt.Errorf("server error")
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runRegister([]string{"user@example.com"})
	})

	if env.lastFatal() == nil {
		t.Fatal("expected fatal error")
	}
	if !strings.Contains(env.lastFatal().Error(), "server error") {
		t.Fatalf("expected 'server error' in fatal, got: %v", env.lastFatal())
	}
}

func TestCLIRunGetInvalidID(t *testing.T) {
	env := setupTestEnv(t)

	env.captureOutput(func() {
		defer func() { recover() }()
		runGet([]string{"not-a-number"})
	})

	if env.lastFatal() == nil {
		t.Fatal("expected fatal error for invalid id")
	}
}

func TestCLIRunDeleteInvalidID(t *testing.T) {
	env := setupTestEnv(t)

	env.captureOutput(func() {
		defer func() { recover() }()
		runDelete([]string{"abc"})
	})

	if env.lastFatal() == nil {
		t.Fatal("expected fatal error for invalid id")
	}
}

// --- Binary build tests (subprocess level) ---

func TestCLIBinaryBuildVersion(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run version: %v", err)
	}

	out := strings.TrimSpace(string(output))
	if !strings.Contains(out, "dev") && !strings.Contains(out, "(") {
		t.Fatalf("version output doesn't look right: %s", out)
	}
}

func TestCLIBinaryUnknownCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin, "foobar")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected 'unknown command' in stderr, got: %s", stderr.String())
	}
}

func TestCLIBinaryNoArgs(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit with no args")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Fatalf("expected usage in stderr, got: %s", stderr.String())
	}
}

func TestCLIRunAddMissingArgs(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin, "add")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for add without args")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Fatalf("expected usage in stderr, got: %s", stderr.String())
	}
}

func TestCLIRunGetMissingArgs(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin, "get")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for get without args")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Fatalf("expected usage in stderr, got: %s", stderr.String())
	}
}

func TestCLIRunDeleteMissingArgs(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gophkeeper-cli")
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	cmd := exec.Command(bin, "delete")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for delete without args")
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Fatalf("expected usage in stderr, got: %s", stderr.String())
	}
}

// --- Unit tests for helper functions ---

func TestParseRecordType(t *testing.T) {
	tests := []struct {
		input    string
		expected models.RecordType
		wantErr  bool
	}{
		{"login", models.RecordTypeLogin, false},
		{"text", models.RecordTypeText, false},
		{"binary", models.RecordTypeBinary, false},
		{"card", models.RecordTypeCard, false},
		{"LOGIN", models.RecordTypeLogin, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRecordType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestPrintRecordLogin(t *testing.T) {
	rec := &models.Record{
		ID:       1,
		Type:     models.RecordTypeLogin,
		Name:     "test site",
		Revision: 5,
		Payload:  models.LoginPayload{Login: "user", Password: "pass"},
	}
	printRecord(rec)
}

func TestPrintRecordCard(t *testing.T) {
	rec := &models.Record{
		ID:       2,
		Type:     models.RecordTypeCard,
		Name:     "my card",
		Revision: 3,
		Payload: models.CardPayload{
			Number:     "4111111111111111",
			HolderName: "John Doe",
			ExpiryDate: "12/30",
			CVV:        "123",
		},
	}
	printRecord(rec)
}

func TestPrintRecordDeleted(t *testing.T) {
	now := time.Now()
	rec := &models.Record{
		ID:        3,
		Type:      models.RecordTypeText,
		Name:      "old note",
		Revision:  1,
		Payload:   models.TextPayload{Content: "hello"},
		DeletedAt: &now,
	}
	printRecord(rec)
}

func TestPrintRecordShort(t *testing.T) {
	rec := models.Record{
		ID:       10,
		Type:     models.RecordTypeBinary,
		Name:     "file.bin",
		Revision: 2,
		Payload:  models.BinaryPayload{Data: []byte{1, 2, 3}},
	}
	printRecordShort(rec)
}

// --- buildPayload tests ---

func TestBuildPayloadText(t *testing.T) {
	p := buildPayload(models.RecordTypeText, "hello world")
	tp, ok := p.(models.TextPayload)
	if !ok {
		t.Fatal("expected TextPayload")
	}
	if tp.Content != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", tp.Content)
	}
}

func TestBuildPayloadCard(t *testing.T) {
	p := buildPayload(models.RecordTypeCard, "4111111111111111")
	cp, ok := p.(models.CardPayload)
	if !ok {
		t.Fatal("expected CardPayload")
	}
	if cp.Number != "4111111111111111" {
		t.Fatalf("expected card number '4111111111111111', got '%s'", cp.Number)
	}
}

func TestBuildPayloadBinary(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	content := []byte("binary test content")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	p := buildPayload(models.RecordTypeBinary, tmpFile)
	bp, ok := p.(models.BinaryPayload)
	if !ok {
		t.Fatal("expected BinaryPayload")
	}
	if string(bp.Data) != string(content) {
		t.Fatalf("expected '%s', got '%s'", string(content), string(bp.Data))
	}
}

// --- extractMetadata tests ---

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantMeta  string
		wantFound bool
		wantRest  []string
	}{
		{
			name:      "no metadata flag",
			args:      []string{"text", "my note", "hello"},
			wantMeta:  "",
			wantFound: false,
			wantRest:  []string{"text", "my note", "hello"},
		},
		{
			name:      "metadata at end",
			args:      []string{"text", "my note", "--metadata", "some info"},
			wantMeta:  "some info",
			wantFound: true,
			wantRest:  []string{"text", "my note"},
		},
		{
			name:      "metadata in middle",
			args:      []string{"text", "--metadata", "info", "my note", "hello"},
			wantMeta:  "info",
			wantFound: true,
			wantRest:  []string{"text", "my note", "hello"},
		},
		{
			name:      "empty metadata value",
			args:      []string{"text", "my note", "--metadata", ""},
			wantMeta:  "",
			wantFound: true,
			wantRest:  []string{"text", "my note"},
		},
		{
			name:      "metadata flag at end without value",
			args:      []string{"text", "my note", "--metadata"},
			wantMeta:  "",
			wantFound: true,
			wantRest:  []string{"text", "my note"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, found, rest := extractMetadata(tt.args)
			if meta != tt.wantMeta {
				t.Fatalf("metadata: got %q, want %q", meta, tt.wantMeta)
			}
			if found != tt.wantFound {
				t.Fatalf("found: got %v, want %v", found, tt.wantFound)
			}
			if len(rest) != len(tt.wantRest) {
				t.Fatalf("rest length: got %v, want %v", rest, tt.wantRest)
			}
			for i := range rest {
				if rest[i] != tt.wantRest[i] {
					t.Fatalf("rest[%d]: got %q, want %q", i, rest[i], tt.wantRest[i])
				}
			}
		})
	}
}

// --- Metadata integration tests ---

func TestCLIRunAddTextWithMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	var savedRecord *models.Record
	env.transport.CreateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       100,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runAdd([]string{"text", "my note", "hello world", "--metadata", "important note"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "important note" {
		t.Fatalf("expected metadata 'important note', got %q", savedRecord.Metadata)
	}
}

func TestCLIRunAddTextWithoutMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	var savedRecord *models.Record
	env.transport.CreateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       101,
			Type:     rec.Type,
			Name:     rec.Name,
			Payload:  rec.Payload,
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runAdd([]string{"text", "my note", "hello world"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "" {
		t.Fatalf("expected empty metadata, got %q", savedRecord.Metadata)
	}
}

func TestCLIRunUpdateMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeText,
			Name:     "old-name",
			Metadata: "old meta",
			Payload:  models.TextPayload{Content: "content"},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runUpdate([]string{"5", "new-name", "new content", "--metadata", "new meta"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "new meta" {
		t.Fatalf("expected metadata 'new meta', got %q", savedRecord.Metadata)
	}
}

func TestCLIRunUpdateClearMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeText,
			Name:     "old-name",
			Metadata: "old meta",
			Payload:  models.TextPayload{Content: "content"},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runUpdate([]string{"5", "new-name", "new content", "--metadata", ""})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "" {
		t.Fatalf("expected empty metadata after clear, got %q", savedRecord.Metadata)
	}
}

func TestCLIRunUpdatePreservesMetadataWithoutFlag(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeText,
			Name:     "old-name",
			Metadata: "preserved meta",
			Payload:  models.TextPayload{Content: "content"},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runUpdate([]string{"5", "new-name", "new content"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "preserved meta" {
		t.Fatalf("expected preserved metadata, got %q", savedRecord.Metadata)
	}
}

func TestCLIRunGetWithMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeLogin,
			Name:     "test site",
			Metadata: "my metadata value",
			Payload:  models.LoginPayload{Login: "user", Password: "pass"},
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runGet([]string{"42"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	out := env.stdout.String()
	if !strings.Contains(out, "Metadata: my metadata value") {
		t.Fatalf("expected 'Metadata: my metadata value' in stdout, got: %s", out)
	}
}

func TestCLIRunAddLoginWithMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	var savedRecord *models.Record
	env.transport.CreateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       200,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runAdd([]string{"login", "my site", "--metadata", "work credentials"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "work credentials" {
		t.Fatalf("expected metadata 'work credentials', got %q", savedRecord.Metadata)
	}
}

func TestCLIRunAddCardWithMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	var savedRecord *models.Record
	env.transport.CreateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       300,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		runAdd([]string{"card", "my card", "4111111111111111", "--metadata", "main card"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "main card" {
		t.Fatalf("expected metadata 'main card', got %q", savedRecord.Metadata)
	}
}

// --- Metadata-only update tests ---

func TestCLIRunUpdateMetadataOnly(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeText,
			Name:     "old-name",
			Metadata: "old meta",
			Payload:  models.TextPayload{Content: "original content"},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		// update <id> <name> --metadata <text> — no data, no payload change
		runUpdate([]string{"5", "old-name", "--metadata", "new meta only"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "new meta only" {
		t.Fatalf("expected metadata 'new meta only', got %q", savedRecord.Metadata)
	}
	// Verify payload was NOT changed
	tp, ok := savedRecord.Payload.(models.TextPayload)
	if !ok {
		t.Fatal("expected TextPayload")
	}
	if tp.Content != "original content" {
		t.Fatalf("payload should be preserved, got %q", tp.Content)
	}
}

func TestCLIRunUpdateMetadataOnlyBinary(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeBinary,
			Name:     "old-name",
			Metadata: "old meta",
			Payload:  models.BinaryPayload{Data: []byte("original binary")},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		// update <id> <name> --metadata <text> — no file path, binary payload preserved
		runUpdate([]string{"5", "old-name", "--metadata", "binary meta"})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "binary meta" {
		t.Fatalf("expected metadata 'binary meta', got %q", savedRecord.Metadata)
	}
	// Verify payload was NOT changed (no upload should happen)
	bp, ok := savedRecord.Payload.(models.BinaryPayload)
	if !ok {
		t.Fatal("expected BinaryPayload")
	}
	if string(bp.Data) != "original binary" {
		t.Fatalf("payload should be preserved, got %q", string(bp.Data))
	}
}

func TestCLIRunUpdateClearMetadataOnly(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	env.transport.GetRecordFunc = func(ctx context.Context, id int64) (*models.Record, error) {
		return &models.Record{
			ID:       id,
			Type:     models.RecordTypeText,
			Name:     "old-name",
			Metadata: "old meta",
			Payload:  models.TextPayload{Content: "original content"},
			Revision: 1,
		}, nil
	}

	var savedRecord *models.Record
	env.transport.UpdateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		savedRecord = rec
		return &models.Record{
			ID:       rec.ID,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: rec.Revision + 1,
		}, nil
	}

	env.captureOutput(func() {
		defer func() { recover() }()
		// update <id> <name> --metadata "" — clear metadata, no payload change
		runUpdate([]string{"5", "old-name", "--metadata", ""})
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	if savedRecord.Metadata != "" {
		t.Fatalf("expected empty metadata after clear, got %q", savedRecord.Metadata)
	}
	tp, ok := savedRecord.Payload.(models.TextPayload)
	if !ok {
		t.Fatal("expected TextPayload")
	}
	if tp.Content != "original content" {
		t.Fatalf("payload should be preserved, got %q", tp.Content)
	}
}

// --- Metadata display tests ---

func TestPrintRecordWithMetadata(t *testing.T) {
	rec := &models.Record{
		ID:       1,
		Type:     models.RecordTypeText,
		Name:     "test note",
		Revision: 5,
		Metadata: "some metadata",
		Payload:  models.TextPayload{Content: "hello"},
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecord(rec)

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Metadata: some metadata") {
		t.Fatalf("expected 'Metadata: some metadata' in output, got: %s", output)
	}
}

func TestPrintRecordWithEmptyMetadata(t *testing.T) {
	rec := &models.Record{
		ID:       2,
		Type:     models.RecordTypeText,
		Name:     "test note",
		Revision: 5,
		Metadata: "",
		Payload:  models.TextPayload{Content: "hello"},
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecord(rec)

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Metadata:") {
		t.Fatalf("expected 'Metadata:' line in output even when empty, got: %s", output)
	}
}

func TestPrintRecordWithMultilineMetadata(t *testing.T) {
	rec := &models.Record{
		ID:       3,
		Type:     models.RecordTypeText,
		Name:     "test note",
		Revision: 5,
		Metadata: "line1\nline2\nline3",
		Payload:  models.TextPayload{Content: "hello"},
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecord(rec)

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Metadata: line1\nline2\nline3") {
		t.Fatalf("expected multiline metadata preserved, got: %s", output)
	}
}

func TestPrintRecordShortWithMetadata(t *testing.T) {
	rec := models.Record{
		ID:       10,
		Type:     models.RecordTypeText,
		Name:     "file.txt",
		Revision: 2,
		Metadata: "some info",
		Payload:  models.TextPayload{Content: "data"},
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecordShort(rec)

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "some info") {
		t.Fatalf("expected metadata in short output, got: %s", output)
	}
}

func TestPrintRecordShortWithoutMetadata(t *testing.T) {
	rec := models.Record{
		ID:       11,
		Type:     models.RecordTypeText,
		Name:     "file.txt",
		Revision: 2,
		Payload:  models.TextPayload{Content: "data"},
	}

	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecordShort(rec)

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	output := buf.String()

	if strings.Contains(output, "\t\t") {
		// Should not have extra tabs for metadata
		t.Fatalf("unexpected extra tab for empty metadata, got: %q", output)
	}
}

func TestCLIRunListWithMetadata(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_ = env.core.Login(ctx, "user@example.com", "test-password")

	// Save a record with metadata through the transport mock
	env.transport.CreateRecordFunc = func(ctx context.Context, rec *models.Record) (*models.Record, error) {
		return &models.Record{
			ID:       1,
			Type:     rec.Type,
			Name:     rec.Name,
			Metadata: rec.Metadata,
			Payload:  rec.Payload,
			Revision: 1,
		}, nil
	}
	env.transport.ListRecordsFunc = func(ctx context.Context, rt models.RecordType, includeDeleted bool) ([]models.Record, error) {
		return []models.Record{
			{ID: 1, Type: models.RecordTypeText, Name: "list test", Revision: 1, Metadata: "important info"},
		}, nil
	}

	// Save to cache so ListRecords (which reads from cache) returns it
	env.store.Records().Put(&models.Record{
		ID:       1,
		Type:     models.RecordTypeText,
		Name:     "list test",
		Revision: 1,
		Metadata: "important info",
		Payload:  models.TextPayload{Content: "content"},
	})

	env.captureOutput(func() {
		defer func() { recover() }()
		runList(nil)
	})

	if env.lastFatal() != nil {
		t.Fatalf("unexpected fatal: %v", env.lastFatal())
	}
	out := env.stdout.String()
	if !strings.Contains(out, "important info") {
		t.Fatalf("expected metadata 'important info' in list output, got: %s", out)
	}
}
