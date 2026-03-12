package screens

import (
	"github.com/CheeziCrew/raclette/internal/maven"
	"github.com/CheeziCrew/curd"
)

// Screen identifies the active view.
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenRunner
	ScreenRepoSelect
	ScreenPrompt
)

// NavigateMsg tells the root model to switch screens.
type NavigateMsg struct {
	Screen Screen
}

// RunCommandMsg tells the app a command was selected from the menu.
type RunCommandMsg struct {
	Command maven.Command
}

// PromptDoneMsg carries the collected user inputs back to the app.
type PromptDoneMsg struct {
	Inputs map[string]string
}

// AllDoneMsg signals all commands are finished.
type AllDoneMsg struct{}

// Re-export shared types.
type BackToMenuMsg = curd.BackToMenuMsg
type RepoSelectDoneMsg = curd.RepoSelectDoneMsg
type RepoInfo = curd.RepoInfo
