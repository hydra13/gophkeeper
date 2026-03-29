package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	grpcc "github.com/hydra13/gophkeeper/pkg/apiclient/grpc"
	"github.com/hydra13/gophkeeper/pkg/cache"
	clientcore "github.com/hydra13/gophkeeper/pkg/clientcore"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "login":
		runLogin(args)
	case "register":
		runRegister(args)
	case "logout":
		runLogout()
	case "list":
		runList(args)
	case "get":
		runGet(args)
	case "save":
		runSave(args)
	case "delete":
		runDelete(args)
	case "sync":
		runSync()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  register <email> <password>")
	fmt.Fprintln(os.Stderr, "  login    <email> <password>")
	fmt.Fprintln(os.Stderr, "  logout")
	fmt.Fprintln(os.Stderr, "  list     [type]            (login|text|binary|card)")
	fmt.Fprintln(os.Stderr, "  get      <id>")
	fmt.Fprintln(os.Stderr, "  save     <type> <name> <data>")
	fmt.Fprintln(os.Stderr, "  delete   <id>")
	fmt.Fprintln(os.Stderr, "  sync")
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
	return "localhost:4400"
}

func newCore() (*clientcore.ClientCore, func(), error) {
	ctx := context.Background()
	addr := defaultServerAddr()

	client, err := grpcc.NewClient(ctx, grpcc.Config{
		Address: addr,
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

	// Восстанавливаем авторизацию из кеша
	core.RestoreAuth()

	cleanup := func() {
		store.Flush()
		client.Close()
	}

	return core, cleanup, nil
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func runLogin(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli login <email> <password>")
		os.Exit(1)
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Login(ctx, args[0], args[1]); err != nil {
		fatal(err)
	}

	fmt.Println("logged in successfully")
}

func runRegister(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli register <email> <password>")
		os.Exit(1)
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Register(ctx, args[0], args[1]); err != nil {
		fatal(err)
	}

	fmt.Println("registered successfully")
}

func runLogout() {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.Logout(ctx); err != nil {
		fatal(err)
	}

	fmt.Println("logged out")
}

func runList(args []string) {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var recordType models.RecordType
	if len(args) > 0 {
		recordType = models.RecordType(args[0])
	}

	records, err := core.ListRecords(ctx, recordType)
	if err != nil {
		fatal(err)
	}

	if len(records) == 0 {
		fmt.Println("no records")
		return
	}

	for _, r := range records {
		fmt.Printf("%d\t%s\t%s\trev=%d\n", r.ID, r.Type, r.Name, r.Revision)
	}
}

func runGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli get <id>")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rec, err := core.GetRecord(ctx, id)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("ID:       %d\n", rec.ID)
	fmt.Printf("Type:     %s\n", rec.Type)
	fmt.Printf("Name:     %s\n", rec.Name)
	fmt.Printf("Revision: %d\n", rec.Revision)
	fmt.Printf("Payload:  %v\n", rec.Payload)
}

func runSave(args []string) {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli save <type> <name> <data>")
		os.Exit(1)
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recordType := models.RecordType(args[0])
	name := args[1]
	data := args[2]

	var payload models.RecordPayload
	switch recordType {
	case models.RecordTypeLogin:
		payload = models.LoginPayload{Login: name, Password: data}
	case models.RecordTypeText:
		payload = models.TextPayload{Content: data}
	case models.RecordTypeCard:
		payload = models.CardPayload{Number: data}
	default:
		fatal(fmt.Errorf("unsupported type: %s (use login, text, card)", recordType))
	}

	rec := &models.Record{
		Type:    recordType,
		Name:    name,
		Payload: payload,
	}

	result, err := core.SaveRecord(ctx, rec)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("saved: id=%d rev=%d\n", result.ID, result.Revision)
}

func runDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gophkeeper-cli delete <id>")
		os.Exit(1)
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fatal(fmt.Errorf("invalid id: %w", err))
	}

	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := core.DeleteRecord(ctx, id); err != nil {
		fatal(err)
	}

	fmt.Println("deleted")
}

func runSync() {
	core, cleanup, err := newCore()
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := core.SyncNow(ctx); err != nil {
		fatal(err)
	}

	fmt.Println("synced")
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
