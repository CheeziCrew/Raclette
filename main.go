package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/CheeziCrew/raclette/internal/tui"
)

func main() {
	p := tea.NewProgram(tui.New())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
