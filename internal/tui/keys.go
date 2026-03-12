package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Common key checks — keeps Update methods clean.

func isBack(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("esc")))
}

func isEnter(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("enter")))
}

func isUp(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("up", "k")))
}

func isDown(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("down", "j")))
}

func isToggle(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("space")))
}

func isSelectAll(msg tea.KeyPressMsg) bool {
	return key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+a")))
}
