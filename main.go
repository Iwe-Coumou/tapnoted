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
	case "messages":
		err = runMessages(args)
	case "queue":
		err = runQueue(args)
	case "songs":
		err = runSongs(args)
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
		{`messages [add|list|delete|reset]`, "Manage the message pool"},
		{`queue [add|status|cancel]`, "Manage the queue"},
		{`songs [add|list|delete|reset]`, "Manage songs"},
		{`replies [list|clear]`, "View or clear her replies"},
	})
	fmt.Println()
	fmt.Println("Run any of those with no subcommand for an interactive menu, e.g. `tapnoted songs`.")
	fmt.Println("Or give the subcommand directly for a fast path, e.g. `tapnoted songs add <url> \"title\"`.")
}

func printUsageRows(rows [][2]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, row := range rows {
		fmt.Fprintf(w, "  tapnoted %s\t%s\n", row[0], row[1])
	}
	w.Flush()
}
