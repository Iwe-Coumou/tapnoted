package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"

	"github.com/charmbracelet/huh"
)

func runConfig(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: tapnoted config <set|show> ...")
	}

	switch args[0] {
	case "set":
		return runConfigSet(args[1:])
	case "show":
		return runConfigShow()
	default:
		return errors.New("usage: tapnoted config <set|show> ...")
	}
}

func runConfigSet(args []string) error {
	fs := flag.NewFlagSet("config set", flag.ExitOnError)
	url := fs.String("url", "", "Worker URL, e.g. https://tapnote.icoumou.workers.dev")
	secret := fs.String("secret", "", "ADMIN_SECRET")
	fs.Parse(args)

	if *url == "" || *secret == "" {
		return errors.New("both --url and --secret are required")
	}

	if err := saveConfig(config{URL: *url, Secret: *secret}); err != nil {
		return err
	}

	path, _ := configPath()
	printSuccess("Saved config to " + path)
	return nil
}

func runConfigShow() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Println(dimStyle.Render("No config found at " + path))
		fmt.Println(dimStyle.Render("Run `tapnoted config set --url <url> --secret <secret>` to create one."))
		return nil
	}

	fmt.Println(headerStyle.Render("Config file:") + " " + path)
	fmt.Println(headerStyle.Render("URL:") + "         " + cfg.URL)
	fmt.Println(headerStyle.Render("Secret:") + "      " + maskSecret(cfg.Secret))
	return nil
}

// maskSecret shows just enough of a secret to sanity-check it's the right
// one, without printing the whole thing into terminal scrollback.
func maskSecret(secret string) string {
	if len(secret) <= 6 {
		return fmt.Sprintf("(set, %d characters)", len(secret))
	}
	return fmt.Sprintf("%s...%s (%d characters)", secret[:3], secret[len(secret)-3:], len(secret))
}

func fetchMessages() ([]string, error) {
	data, err := request(http.MethodGet, "/messages", nil)
	if err != nil {
		return nil, err
	}
	var messages []string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func queueMessage(message string) error {
	if _, err := request(http.MethodPost, "/queue", map[string]string{"message": message}); err != nil {
		return err
	}
	printSuccess("Queued.")
	return nil
}

// runQueue queues a message for the next tap. With an argument, it queues
// that text directly (fast path for scripting). With no argument, it opens
// an interactive picker: choose an existing pool message, or type a new one.
func runQueue(args []string) error {
	if len(args) > 0 {
		return queueMessage(args[0])
	}

	messages, err := fetchMessages()
	if err != nil {
		return err
	}

	const newEntry = -1
	opts := make([]huh.Option[int], 0, len(messages)+1)
	opts = append(opts, huh.NewOption("✎ Type a new message...", newEntry))
	for i, m := range messages {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%d: %s", i, m), i))
	}

	var selected int
	if err := huh.NewSelect[int]().
		Title("Queue which message for the next tap?").
		Options(opts...).
		Value(&selected).
		Run(); err != nil {
		return err
	}

	if selected == newEntry {
		var text string
		if err := huh.NewInput().Title("New message").Value(&text).Run(); err != nil {
			return err
		}
		return queueMessage(text)
	}

	return queueMessage(messages[selected])
}

func runStatus() error {
	data, err := request(http.MethodGet, "/queue", nil)
	if err != nil {
		return err
	}

	var result struct {
		Queued *string `json:"queued"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	if result.Queued == nil {
		fmt.Println(dimStyle.Render("Nothing queued — next tap will be random."))
	} else {
		fmt.Println(headerStyle.Render("Queued:") + " " + *result.Queued)
	}
	return nil
}

func runCancel() error {
	if _, err := request(http.MethodDelete, "/queue", nil); err != nil {
		return err
	}
	printSuccess("Cleared.")
	return nil
}

func runAdd(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: tapnoted add "<message>"`)
	}
	if _, err := request(http.MethodPost, "/messages", map[string]string{"message": args[0]}); err != nil {
		return err
	}
	printSuccess("Added.")
	return nil
}

func runList() error {
	messages, err := fetchMessages()
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println(dimStyle.Render("Pool is empty."))
		return nil
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf("Message pool (%d)", len(messages))))
	for i, m := range messages {
		fmt.Printf("%s  %s\n", dimStyle.Render(fmt.Sprintf("%2d", i)), m)
	}
	return nil
}

func deleteByIndex(index int) error {
	if _, err := request(http.MethodDelete, "/messages", map[string]int{"index": index}); err != nil {
		return err
	}
	printSuccess("Deleted.")
	return nil
}

// runDelete removes a message from the pool. With an index argument, it
// deletes directly. With no argument, it opens an interactive picker with a
// confirmation step, since deleting is harder to undo than adding.
func runDelete(args []string) error {
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("index must be a number: %w", err)
		}
		return deleteByIndex(index)
	}

	messages, err := fetchMessages()
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		fmt.Println(dimStyle.Render("Pool is empty — nothing to delete."))
		return nil
	}

	opts := make([]huh.Option[int], len(messages))
	for i, m := range messages {
		opts[i] = huh.NewOption(fmt.Sprintf("%d: %s", i, m), i)
	}

	var selected int
	if err := huh.NewSelect[int]().
		Title("Pick a message to delete").
		Options(opts...).
		Value(&selected).
		Run(); err != nil {
		return err
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Delete %q?", messages[selected])).
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	return deleteByIndex(selected)
}

func runReset() error {
	var confirm bool
	if err := huh.NewConfirm().
		Title("This will permanently clear the entire message pool. Continue?").
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	if _, err := request(http.MethodPut, "/messages", map[string][]string{"messages": {}}); err != nil {
		return err
	}
	printSuccess("Pool reset.")
	return nil
}
