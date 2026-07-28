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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

// errBack is returned by a leaf action (add/delete/reset/...) when the user
// pressed Esc partway through it. It's not a real error — callers use it to
// decide whether to return to the enclosing menu instead of exiting.
var errBack = errors.New("back")

// escKeyMap adds Esc to huh's default "quit this form" binding (which is
// otherwise Ctrl+C only), so Esc can be used to back out of a prompt.
func escKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	return km
}

// selectFrom shows a single-field select menu. ok is false (err nil) if the
// user backed out with Esc/Ctrl+C, distinct from a real error.
func selectFrom[T comparable](title string, opts ...huh.Option[T]) (value T, ok bool, err error) {
	field := huh.NewSelect[T]().Title(title).Options(opts...).Value(&value)
	runErr := huh.NewForm(huh.NewGroup(field)).WithShowHelp(false).WithKeyMap(escKeyMap()).Run()
	if errors.Is(runErr, huh.ErrUserAborted) {
		return value, false, nil
	}
	if runErr != nil {
		return value, false, runErr
	}
	return value, true, nil
}

func confirmPrompt(title string) (value bool, ok bool, err error) {
	field := huh.NewConfirm().Title(title).Value(&value)
	runErr := huh.NewForm(huh.NewGroup(field)).WithShowHelp(false).WithKeyMap(escKeyMap()).Run()
	if errors.Is(runErr, huh.ErrUserAborted) {
		return false, false, nil
	}
	if runErr != nil {
		return false, false, runErr
	}
	return value, true, nil
}

func inputPrompt(title string) (value string, ok bool, err error) {
	field := huh.NewInput().Title(title).Value(&value)
	runErr := huh.NewForm(huh.NewGroup(field)).WithShowHelp(false).WithKeyMap(escKeyMap()).Run()
	if errors.Is(runErr, huh.ErrUserAborted) {
		return "", false, nil
	}
	if runErr != nil {
		return "", false, runErr
	}
	return value, true, nil
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

// runMessages manages the random-pick pool. With a subcommand it acts
// directly. With none, it shows a menu; Esc on the menu exits, Esc inside
// whichever action was chosen returns to this same menu.
func runMessages(args []string) error {
	if len(args) > 0 {
		var err error
		switch args[0] {
		case "add":
			err = messageAdd(args[1:])
		case "list":
			err = messageList()
		case "delete":
			err = messageDelete(args[1:])
		case "reset":
			err = messageReset()
		default:
			return errors.New("usage: tapnoted messages <add|list|delete|reset> ...")
		}
		return backOrErr(err)
	}

	for {
		action, ok, err := selectFrom("Messages",
			huh.NewOption("Add a message", "add"),
			huh.NewOption("List the pool", "list"),
			huh.NewOption("Delete a message", "delete"),
			huh.NewOption("Reset the pool", "reset"),
		)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		switch action {
		case "add":
			err = messageAdd(nil)
		case "list":
			return messageList()
		case "delete":
			err = messageDelete(nil)
		case "reset":
			err = messageReset()
		}
		if errors.Is(err, errBack) {
			continue
		}
		return err
	}
}

// backOrErr turns errBack into a clean "Cancelled." (no scary error text)
// for the direct-subcommand entry point, where there's no menu to return to.
func backOrErr(err error) error {
	if errors.Is(err, errBack) {
		fmt.Println("Cancelled.")
		return nil
	}
	return err
}

// messageAdd adds a message. With text given, it adds directly. With none,
// it prompts for the text — this is what lets the interactive menu above
// just call messageAdd(nil) without duplicating any prompt logic.
func messageAdd(args []string) error {
	var text string
	if len(args) > 0 {
		text = args[0]
	} else {
		value, ok, err := inputPrompt("New message")
		if err != nil {
			return err
		}
		if !ok {
			return errBack
		}
		text = value
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

	selected, ok, err := selectFrom("Pick a message to delete", opts...)
	if err != nil {
		return err
	}
	if !ok {
		return errBack
	}

	confirm, ok, err := confirmPrompt(fmt.Sprintf("Delete %q?", messages[selected]))
	if err != nil {
		return err
	}
	if !ok || !confirm {
		return errBack
	}

	return messageDeleteByIndex(selected)
}

func messageReset() error {
	confirm, ok, err := confirmPrompt("This will permanently clear the entire message pool. Continue?")
	if err != nil {
		return err
	}
	if !ok || !confirm {
		return errBack
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

// runQueue manages the queue. With a subcommand it acts directly. With
// none, it shows a menu (same Esc behavior as runMessages).
func runQueue(args []string) error {
	if len(args) > 0 {
		var err error
		switch args[0] {
		case "add":
			err = queueAdd(args[1:])
		case "status":
			err = queueStatus()
		case "cancel":
			err = queueCancel(args[1:])
		default:
			return errors.New("usage: tapnoted queue <add|status|cancel> ...")
		}
		return backOrErr(err)
	}

	for {
		action, ok, err := selectFrom("Queue",
			huh.NewOption("Add to the queue", "add"),
			huh.NewOption("Show the queue", "status"),
			huh.NewOption("Clear the queue", "cancel"),
		)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		switch action {
		case "add":
			err = queueAdd(nil)
		case "status":
			return queueStatus()
		case "cancel":
			err = queueCancel(nil)
		}
		if errors.Is(err, errBack) {
			continue
		}
		return err
	}
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

	selected, ok, err := selectFrom("Add which message to the queue?", opts...)
	if err != nil {
		return err
	}
	if !ok {
		return errBack
	}

	if selected == newEntry {
		text, ok, err := inputPrompt("New message")
		if err != nil {
			return err
		}
		if !ok {
			return errBack
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

	confirm, ok, err := confirmPrompt("This will clear the entire queue. Continue?")
	if err != nil {
		return err
	}
	if !ok || !confirm {
		return errBack
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

// runReplies shows what she's sent back. There's nothing to choose — viewing
// them is the only action, and it clears them as a side effect (view-once,
// same idea as GET /message destructively popping the queue) — so no menu,
// no subcommands, just show them.
func runReplies(args []string) error {
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

// fetchReplyCount is a safe, non-destructive peek at how many replies are
// waiting — used by `overview`, which must not accidentally consume them
// just by checking in.
func fetchReplyCount() (int, error) {
	data, err := request(http.MethodGet, "/replies/count", nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	return result.Count, nil
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
// live playlist lookup). With a subcommand it acts directly. With none, it
// shows a menu.
func runSongs(args []string) error {
	if len(args) > 0 {
		var err error
		switch args[0] {
		case "add":
			err = songAdd(args[1:])
		case "list":
			err = songList()
		case "delete":
			err = songDelete(args[1:])
		case "reset":
			err = songReset()
		default:
			return errors.New(`usage: tapnoted songs <add <url> ["title"]|list|delete [<index>]|reset>`)
		}
		return backOrErr(err)
	}

	for {
		action, ok, err := selectFrom("Songs",
			huh.NewOption("Add a song", "add"),
			huh.NewOption("List songs", "list"),
			huh.NewOption("Delete a song", "delete"),
			huh.NewOption("Clear all songs", "reset"),
		)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		switch action {
		case "add":
			err = songAdd(nil)
		case "list":
			return songList()
		case "delete":
			err = songDelete(nil)
		case "reset":
			err = songReset()
		}
		if errors.Is(err, errBack) {
			continue
		}
		return err
	}
}

// songAdd adds a song. With a URL (and optional title) given, it adds
// directly. With no arguments at all, it prompts for both — used by the
// interactive menu above via songAdd(nil).
func songAdd(args []string) error {
	if len(args) == 0 {
		urlInput, ok, err := inputPrompt("Spotify track URL")
		if err != nil {
			return err
		}
		if !ok {
			return errBack
		}

		title, ok, err := inputPrompt("Title (optional)")
		if err != nil {
			return err
		}
		if !ok {
			return errBack
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

	selected, ok, err := selectFrom("Pick a song to delete", opts...)
	if err != nil {
		return err
	}
	if !ok {
		return errBack
	}

	confirm, ok, err := confirmPrompt(fmt.Sprintf("Delete %q?", songs[selected].label()))
	if err != nil {
		return err
	}
	if !ok || !confirm {
		return errBack
	}

	return songDeleteByIndex(selected)
}

func songReset() error {
	confirm, ok, err := confirmPrompt("This will permanently clear all songs. Continue?")
	if err != nil {
		return err
	}
	if !ok || !confirm {
		return errBack
	}

	if _, err := request(http.MethodPut, "/songs", map[string][]song{"songs": {}}); err != nil {
		return err
	}
	printSuccess("Songs cleared.")
	return nil
}

// --- overview -------------------------------------------------------------

// runOverview prints a quick, at-a-glance summary across every resource.
// The reply count is fetched via the non-destructive /replies/count
// endpoint — checking in here must never silently consume pending replies.
func runOverview() error {
	queue, err := fetchQueue()
	if err != nil {
		return err
	}
	messages, err := fetchMessages()
	if err != nil {
		return err
	}
	songs, err := fetchSongs()
	if err != nil {
		return err
	}
	replyCount, err := fetchReplyCount()
	if err != nil {
		return err
	}

	fmt.Println(headerStyle.Render("Overview"))
	fmt.Printf("  Queue:    %d waiting\n", len(queue))
	fmt.Printf("  Pool:     %d messages\n", len(messages))
	fmt.Printf("  Songs:    %d\n", len(songs))
	if replyCount == 0 {
		fmt.Println(dimStyle.Render("  Replies:  none waiting"))
	} else {
		fmt.Println(noticeStyle.Render(fmt.Sprintf("  Replies:  %d new — run `tapnoted replies` to view", replyCount)))
	}
	return nil
}
