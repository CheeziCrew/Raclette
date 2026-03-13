package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/CheeziCrew/raclette/internal/cli"
	"github.com/CheeziCrew/raclette/internal/tui"
)

func main() {
	// CLI mode if any args are provided.
	if len(os.Args) > 1 {
		if err := cli.BuildCLI().Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode (default).
	p := tea.NewProgram(tui.New())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
