package main

import (
	"fmt"
	"os"
	"text/tabwriter"
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
	fmt.Println("tapnoted — manage your tapnote gift")
	fmt.Println()
	fmt.Println("Usage:")

	rows := [][2]string{
		{`config set --url <url> --secret <secret>`, "Save connection details"},
		{`config show`, "Show the saved config (secret partially masked)"},
		{`queue ["<message>"]`, "Add a message to the queue (interactive picker if omitted)"},
		{`status`, "List what's currently queued, in order"},
		{`cancel [<index>]`, "Clear the whole queue (asks to confirm), or remove one item by index"},
		{`add "<message>"`, "Add a message to the random pool"},
		{`list`, "List all messages in the pool"},
		{`delete [<index>]`, "Remove a message (interactive picker if index omitted)"},
		{`reset`, "Clear the entire pool (asks to confirm)"},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(w, "  tapnoted %s\t%s\n", row[0], row[1])
	}
	w.Flush()
}
