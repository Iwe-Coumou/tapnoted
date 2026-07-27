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
	case "replies":
		err = runReplies(args)
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

	// Split into groups so one long line (config set, with its flags)
	// doesn't force every short command to pad out to the same width.
	printUsageRows([][2]string{
		{`config set --url <url> --secret <secret>`, "Save connection details"},
		{`config show`, "Show saved config (secret masked)"},
	})
	fmt.Println()
	printUsageRows([][2]string{
		{`queue ["<message>"]`, "Queue a message"},
		{`status`, "Show the queue"},
		{`cancel [<index>]`, "Clear queue, or remove one by index"},
		{`add "<message>"`, "Add a message to the pool"},
		{`list`, "List the pool"},
		{`delete [<index>]`, "Remove a message"},
		{`reset`, "Clear the pool"},
		{`replies [list]`, "Show her replies"},
		{`replies clear [<index>]`, "Clear replies, or remove one by index"},
	})
}

func printUsageRows(rows [][2]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(w, "  tapnoted %s\t%s\n", row[0], row[1])
	}
	w.Flush()
}
