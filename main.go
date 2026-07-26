package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "config":
		err = runConfig(args)
	case "queue":
		err = runQueue(args)
	case "status":
		err = runStatus()
	case "cancel":
		err = runCancel()
	case "add":
		err = runAdd(args)
	case "list":
		err = runList()
	case "delete":
		err = runDelete(args)
	case "reset":
		err = runReset()
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`tapnoted — manage your tapnote gift

Usage:
  tapnoted config set --url <url> --secret <secret>   Save connection details
  tapnoted queue "<message>"                          Queue a message for the next tap
  tapnoted status                                     Show what's currently queued
  tapnoted cancel                                     Clear a queued message
  tapnoted add "<message>"                            Add a message to the random pool
  tapnoted list                                        List all messages in the pool
  tapnoted delete <index>                              Remove a message by its list index
  tapnoted reset                                       Clear the entire pool (asks to confirm)`)
}
