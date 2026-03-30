package main

import (
	"fmt"
	"os"

	"github.com/hydra13/gophkeeper/pkg/buildinfo"
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
	case "add":
		runAdd(args)
	case "update":
		runUpdate(args)
	case "delete":
		runDelete(args)
	case "sync":
		runSync()
	case "version":
		fmt.Printf("gophkeeper-cli %s\n", buildinfo.Short())
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
