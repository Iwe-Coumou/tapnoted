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
		err = runCancel(args)
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
		printErr(err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`tapnoted — manage your tapnote gift

Usage:
  tapnoted config set --url <url> --secret <secret>   Save connection details
  tapnoted config show                                Show the saved config (secret partially masked)
  tapnoted queue ["<message>"]                        Add a message to the queue (interactive picker if omitted)
  tapnoted status                                     List what's currently queued, in order
  tapnoted cancel [<index>]                            Clear the whole queue (asks to confirm), or remove one item by index
  tapnoted add "<message>"                            Add a message to the random pool
  tapnoted list                                        List all messages in the pool
  tapnoted delete [<index>]                            Remove a message (interactive picker if index omitted)
  tapnoted reset                                       Clear the entire pool (asks to confirm)`)
}
