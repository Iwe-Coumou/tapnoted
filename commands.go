package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"
)

func runConfig(args []string) error {
	if len(args) == 0 || args[0] != "set" {
		return errors.New("usage: tapnoted config set --url <url> --secret <secret>")
	}

	fs := flag.NewFlagSet("config set", flag.ExitOnError)
	url := fs.String("url", "", "Worker URL, e.g. https://tapnote.icoumou.workers.dev")
	secret := fs.String("secret", "", "ADMIN_SECRET")
	fs.Parse(args[1:])

	if *url == "" || *secret == "" {
		return errors.New("both --url and --secret are required")
	}

	if err := saveConfig(config{URL: *url, Secret: *secret}); err != nil {
		return err
	}

	path, _ := configPath()
	fmt.Println("Saved config to", path)
	return nil
}

func runQueue(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: tapnoted queue "<message>"`)
	}
	if _, err := request(http.MethodPost, "/queue", map[string]string{"message": args[0]}); err != nil {
		return err
	}
	fmt.Println("Queued.")
	return nil
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
		fmt.Println("Nothing queued — next tap will be random.")
	} else {
		fmt.Println("Queued:", *result.Queued)
	}
	return nil
}

func runCancel() error {
	if _, err := request(http.MethodDelete, "/queue", nil); err != nil {
		return err
	}
	fmt.Println("Cleared.")
	return nil
}

func runAdd(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: tapnoted add "<message>"`)
	}
	if _, err := request(http.MethodPost, "/messages", map[string]string{"message": args[0]}); err != nil {
		return err
	}
	fmt.Println("Added.")
	return nil
}

func runList() error {
	data, err := request(http.MethodGet, "/messages", nil)
	if err != nil {
		return err
	}

	var messages []string
	if err := json.Unmarshal(data, &messages); err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println("Pool is empty.")
		return nil
	}

	for i, m := range messages {
		fmt.Printf("%d: %s\n", i, m)
	}
	return nil
}

func runDelete(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: tapnoted delete <index>")
	}

	index, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("index must be a number: %w", err)
	}

	if _, err := request(http.MethodDelete, "/messages", map[string]int{"index": index}); err != nil {
		return err
	}
	fmt.Println("Deleted.")
	return nil
}

func runReset() error {
	fmt.Print("This will permanently clear the entire message pool. Continue? (y/N): ")
	var answer string
	fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	if _, err := request(http.MethodPut, "/messages", map[string][]string{"messages": {}}); err != nil {
		return err
	}
	fmt.Println("Pool reset.")
	return nil
}
