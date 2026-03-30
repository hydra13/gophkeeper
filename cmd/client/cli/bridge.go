package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hydra13/gophkeeper/cmd/client/cli/commands"
	"github.com/hydra13/gophkeeper/internal/models"
	grpcc "github.com/hydra13/gophkeeper/pkg/apiclient/grpc"
	"github.com/hydra13/gophkeeper/pkg/cache"
	clientcore "github.com/hydra13/gophkeeper/pkg/clientcore"
	"golang.org/x/term"
)

var newCoreFunc = defaultNewCore

func defaultNewCore() (*clientcore.ClientCore, func(), error) {
	ctx := context.Background()
	addr := defaultServerAddr()
	certFile := defaultTLSCertFile()

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

	store, err := cache.NewFileStore(defaultCacheDir())
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("init cache: %w", err)
	}

	core := clientcore.New(client, store, clientcore.Config{
		DeviceID: "cli-" + hostname(),
	})

	core.RestoreAuth()

	cleanup := func() {
		store.Flush()
		client.Close()
	}

	return core, cleanup, nil
}

func newCore() (*clientcore.ClientCore, func(), error) {
	return newCoreFunc()
}

var fatalFunc = defaultFatal

func defaultFatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func fatal(err error) {
	fatalFunc(err)
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gophkeeper", "cache")
}

func defaultServerAddr() string {
	if v := os.Getenv("GK_GRPC_ADDRESS"); v != "" {
		return v
	}
	return "localhost:9090"
}

func defaultTLSCertFile() string {
	if v := os.Getenv("GK_TLS_CERT_FILE"); v != "" {
		return v
	}
	if _, err := os.Stat("configs/certs/dev.crt"); err == nil {
		return "configs/certs/dev.crt"
	}
	return ""
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

var readPasswordFunc = defaultReadPassword

func defaultReadPassword(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fatal(fmt.Errorf("read password: %w", err))
	}
	return string(b)
}

func readPassword(prompt string) string {
	return readPasswordFunc(prompt)
}

var readLineFunc = defaultReadLine

func defaultReadLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	var line string
	if _, err := fmt.Scanln(&line); err != nil {
		fatal(fmt.Errorf("read input: %w", err))
	}
	return strings.TrimSpace(line)
}

func readLine(prompt string) string {
	return readLineFunc(prompt)
}

func cliRunner() *commands.Runner {
	return commands.New(commands.Dependencies{
		NewCore:      newCore,
		Fatal:        fatal,
		ReadPassword: readPassword,
		ReadLine:     readLine,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	})
}

func printUsage() {
	cliRunner().PrintUsage()
}

func extractMetadata(args []string) (metadata string, found bool, rest []string) {
	return commands.ExtractMetadata(args)
}

func parseRecordType(s string) (models.RecordType, error) {
	return commands.ParseRecordType(s)
}

func printRecord(rec *models.Record) {
	commands.PrintRecord(os.Stdout, rec)
}

func printRecordShort(r models.Record) {
	commands.PrintRecordShort(os.Stdout, r)
}

func buildPayload(recordType models.RecordType, data string) models.RecordPayload {
	return cliRunner().BuildPayload(recordType, data)
}

func runRegister(args []string) {
	cliRunner().RunRegister(args)
}

func runLogin(args []string) {
	cliRunner().RunLogin(args)
}

func runLogout() {
	cliRunner().RunLogout()
}

func runList(args []string) {
	cliRunner().RunList(args)
}

func runGet(args []string) {
	cliRunner().RunGet(args)
}

func runAdd(args []string) {
	cliRunner().RunAdd(args)
}

func runUpdate(args []string) {
	cliRunner().RunUpdate(args)
}

func runDelete(args []string) {
	cliRunner().RunDelete(args)
}

func runSync() {
	cliRunner().RunSync()
}
