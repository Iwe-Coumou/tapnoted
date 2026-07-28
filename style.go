package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Faint(true)
	headerStyle  = lipgloss.NewStyle().Bold(true)
	// noticeStyle is for "something worth your attention" (e.g. new replies
	// waiting) — distinct from successStyle, which means "an action you took
	// just completed."
	noticeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
)

func printSuccess(msg string) {
	fmt.Println(successStyle.Render("✓") + " " + msg)
}

func printErr(err error) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗")+" "+err.Error())
}
