package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// pickAction shows a single arrow-key menu for choosing what to do with a
// resource (messages/queue/songs/replies) when the command is run with no
// subcommand at all — the shared building block behind that behavior.
func pickAction(title string, opts ...huh.Option[string]) (string, error) {
	var action string
	err := huh.NewSelect[string]().
		Title(title).
		Options(opts...).
		Value(&action).
		Run()
	return action, err
}

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

// --- messages ---------------------------------------------------------

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

// runMessages manages the random-pick pool. With a subcommand
// (add/list/delete/reset) it acts directly — handy for scripting. With no
// subcommand it opens a menu, which is really just another way of reaching
// the same functions below.
func runMessages(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return messageAdd(args[1:])
		case "list":
			return messageList()
		case "delete":
			return messageDelete(args[1:])
		case "reset":
			return messageReset()
		default:
			return errors.New("usage: tapnoted messages <add|list|delete|reset> ...")
		}
	}

	action, err := pickAction("Messages",
		huh.NewOption("Add a message", "add"),
		huh.NewOption("List the pool", "list"),
		huh.NewOption("Delete a message", "delete"),
		huh.NewOption("Reset the pool", "reset"),
	)
	if err != nil {
		return err
	}

	switch action {
	case "add":
		return messageAdd(nil)
	case "list":
		return messageList()
	case "delete":
		return messageDelete(nil)
	case "reset":
		return messageReset()
	}
	return nil
}

// messageAdd adds a message. With text given, it adds directly. With none,
// it prompts for the text — this is what lets the interactive menu above
// just call messageAdd(nil) without duplicating any prompt logic.
func messageAdd(args []string) error {
	var text string
	if len(args) > 0 {
		text = args[0]
	} else {
		if err := huh.NewInput().Title("New message").Value(&text).Run(); err != nil {
			return err
		}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("message cannot be empty")
	}

	if _, err := request(http.MethodPost, "/messages", map[string]string{"message": text}); err != nil {
		return err
	}
	printSuccess("Added.")
	return nil
}

func messageList() error {
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

func messageDeleteByIndex(index int) error {
	if _, err := request(http.MethodDelete, "/messages", map[string]int{"index": index}); err != nil {
		return err
	}
	printSuccess("Deleted.")
	return nil
}

// messageDelete removes a message from the pool. With an index argument, it
// deletes directly. With no argument, it opens an interactive picker with a
// confirmation step, since deleting is harder to undo than adding.
func messageDelete(args []string) error {
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("index must be a number: %w", err)
		}
		return messageDeleteByIndex(index)
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

	return messageDeleteByIndex(selected)
}

func messageReset() error {
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

// --- queue --------------------------------------------------------------

func fetchQueue() ([]string, error) {
	data, err := request(http.MethodGet, "/queue", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Queue []string `json:"queue"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.Queue, nil
}

// runQueue manages the queue. With a subcommand (add/status/cancel) it acts
// directly. With none, it opens a menu.
func runQueue(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return queueAdd(args[1:])
		case "status":
			return queueStatus()
		case "cancel":
			return queueCancel(args[1:])
		default:
			return errors.New("usage: tapnoted queue <add|status|cancel> ...")
		}
	}

	action, err := pickAction("Queue",
		huh.NewOption("Add to the queue", "add"),
		huh.NewOption("Show the queue", "status"),
		huh.NewOption("Clear the queue", "cancel"),
	)
	if err != nil {
		return err
	}

	switch action {
	case "add":
		return queueAdd(nil)
	case "status":
		return queueStatus()
	case "cancel":
		return queueCancel(nil)
	}
	return nil
}

func queueAddMessage(message string) error {
	if _, err := request(http.MethodPost, "/queue", map[string]string{"message": message}); err != nil {
		return err
	}
	printSuccess("Queued.")
	return nil
}

// queueAdd adds a message to the queue. With an argument, it queues that
// text directly (fast path for scripting). With no argument, it opens an
// interactive picker: choose an existing pool message, or type a new one.
func queueAdd(args []string) error {
	if len(args) > 0 {
		return queueAddMessage(args[0])
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
		Title("Add which message to the queue?").
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
		return queueAddMessage(text)
	}

	return queueAddMessage(messages[selected])
}

func queueStatus() error {
	queue, err := fetchQueue()
	if err != nil {
		return err
	}

	if len(queue) == 0 {
		fmt.Println(dimStyle.Render("Nothing queued — next tap will be random."))
		return nil
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf("Queue (%d)", len(queue))))
	for i, m := range queue {
		fmt.Printf("%s  %s\n", dimStyle.Render(fmt.Sprintf("%2d", i)), m)
	}
	return nil
}

func queueCancelByIndex(index int) error {
	if _, err := request(http.MethodDelete, "/queue", map[string]int{"index": index}); err != nil {
		return err
	}
	printSuccess("Removed.")
	return nil
}

// queueCancel removes queued messages. With an index argument, it removes
// just that one entry directly. With no argument, it clears the whole
// queue, after confirming — a queue can hold several messages, so wiping
// it all is more consequential than removing one.
func queueCancel(args []string) error {
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("index must be a number: %w", err)
		}
		return queueCancelByIndex(index)
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title("This will clear the entire queue. Continue?").
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	if _, err := request(http.MethodDelete, "/queue", nil); err != nil {
		return err
	}
	printSuccess("Cleared.")
	return nil
}

// --- replies --------------------------------------------------------------

type reply struct {
	Text      string `json:"text"`
	RepliedTo string `json:"repliedTo"`
	Timestamp string `json:"timestamp"`
}

func fetchReplies() ([]reply, error) {
	data, err := request(http.MethodGet, "/replies", nil)
	if err != nil {
		return nil, err
	}
	var replies []reply
	if err := json.Unmarshal(data, &replies); err != nil {
		return nil, err
	}
	return replies, nil
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("Jan 2 15:04")
}

// runReplies shows what she's sent back, or clears it. With a subcommand
// (list/clear) it acts directly. With none, it opens a menu.
func runReplies(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "list":
			return repliesList()
		case "clear":
			return repliesClear(args[1:])
		default:
			return errors.New("usage: tapnoted replies <list|clear [<index>]> ...")
		}
	}

	action, err := pickAction("Replies",
		huh.NewOption("Show replies", "list"),
		huh.NewOption("Clear replies", "clear"),
	)
	if err != nil {
		return err
	}

	switch action {
	case "list":
		return repliesList()
	case "clear":
		return repliesClear(nil)
	}
	return nil
}

func repliesList() error {
	replies, err := fetchReplies()
	if err != nil {
		return err
	}

	if len(replies) == 0 {
		fmt.Println(dimStyle.Render("No replies yet."))
		return nil
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf("Replies (%d)", len(replies))))
	for i, r := range replies {
		fmt.Printf("%s  %s\n", dimStyle.Render(fmt.Sprintf("%2d", i)), formatTimestamp(r.Timestamp))
		if r.RepliedTo != "" {
			fmt.Println(dimStyle.Render("     re: ") + r.RepliedTo)
		}
		fmt.Println("     " + r.Text)
	}
	return nil
}

func repliesClear(args []string) error {
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("index must be a number: %w", err)
		}
		if _, err := request(http.MethodDelete, "/replies", map[string]int{"index": index}); err != nil {
			return err
		}
		printSuccess("Removed.")
		return nil
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title("This will clear all replies. Continue?").
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	if _, err := request(http.MethodDelete, "/replies", nil); err != nil {
		return err
	}
	printSuccess("Cleared.")
	return nil
}

// --- songs --------------------------------------------------------------

type song struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

func (s song) label() string {
	if s.Title != "" {
		return s.Title
	}
	return s.URL
}

func fetchSongs() ([]song, error) {
	data, err := request(http.MethodGet, "/songs", nil)
	if err != nil {
		return nil, err
	}
	var songs []song
	if err := json.Unmarshal(data, &songs); err != nil {
		return nil, err
	}
	return songs, nil
}

// runSongs manages the curated song pool (plain Spotify track links, not a
// live playlist lookup). With a subcommand (add/list/delete/reset) it acts
// directly. With none, it opens a menu.
func runSongs(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return songAdd(args[1:])
		case "list":
			return songList()
		case "delete":
			return songDelete(args[1:])
		case "reset":
			return songReset()
		default:
			return errors.New(`usage: tapnoted songs <add <url> ["title"]|list|delete [<index>]|reset>`)
		}
	}

	action, err := pickAction("Songs",
		huh.NewOption("Add a song", "add"),
		huh.NewOption("List songs", "list"),
		huh.NewOption("Delete a song", "delete"),
		huh.NewOption("Clear all songs", "reset"),
	)
	if err != nil {
		return err
	}

	switch action {
	case "add":
		return songAdd(nil)
	case "list":
		return songList()
	case "delete":
		return songDelete(nil)
	case "reset":
		return songReset()
	}
	return nil
}

// songAdd adds a song. With a URL (and optional title) given, it adds
// directly. With no arguments at all, it prompts for both — used by the
// interactive menu above via songAdd(nil).
func songAdd(args []string) error {
	if len(args) == 0 {
		var urlInput, title string
		if err := huh.NewInput().Title("Spotify track URL").Value(&urlInput).Run(); err != nil {
			return err
		}
		if err := huh.NewInput().Title("Title (optional)").Value(&title).Run(); err != nil {
			return err
		}
		args = []string{urlInput}
		if title != "" {
			args = append(args, title)
		}
	}

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return errors.New(`usage: tapnoted songs add <spotify-track-url> ["title"]`)
	}

	body := map[string]string{"url": args[0]}
	if len(args) > 1 {
		body["title"] = args[1]
	}

	if _, err := request(http.MethodPost, "/songs", body); err != nil {
		return err
	}
	printSuccess("Added.")
	return nil
}

func songList() error {
	songs, err := fetchSongs()
	if err != nil {
		return err
	}

	if len(songs) == 0 {
		fmt.Println(dimStyle.Render("No songs yet."))
		return nil
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf("Songs (%d)", len(songs))))
	for i, s := range songs {
		fmt.Printf("%s  %s\n", dimStyle.Render(fmt.Sprintf("%2d", i)), s.label())
	}
	return nil
}

func songDeleteByIndex(index int) error {
	if _, err := request(http.MethodDelete, "/songs", map[string]int{"index": index}); err != nil {
		return err
	}
	printSuccess("Deleted.")
	return nil
}

func songDelete(args []string) error {
	if len(args) > 0 {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("index must be a number: %w", err)
		}
		return songDeleteByIndex(index)
	}

	songs, err := fetchSongs()
	if err != nil {
		return err
	}
	if len(songs) == 0 {
		fmt.Println(dimStyle.Render("No songs — nothing to delete."))
		return nil
	}

	opts := make([]huh.Option[int], len(songs))
	for i, s := range songs {
		opts[i] = huh.NewOption(fmt.Sprintf("%d: %s", i, s.label()), i)
	}

	var selected int
	if err := huh.NewSelect[int]().
		Title("Pick a song to delete").
		Options(opts...).
		Value(&selected).
		Run(); err != nil {
		return err
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Delete %q?", songs[selected].label())).
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	return songDeleteByIndex(selected)
}

func songReset() error {
	var confirm bool
	if err := huh.NewConfirm().
		Title("This will permanently clear all songs. Continue?").
		Value(&confirm).
		Run(); err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Cancelled.")
		return nil
	}

	if _, err := request(http.MethodPut, "/songs", map[string][]song{"songs": {}}); err != nil {
		return err
	}
	printSuccess("Songs cleared.")
	return nil
}
