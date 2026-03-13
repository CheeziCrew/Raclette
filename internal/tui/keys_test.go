package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestIsBack(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
		want bool
	}{
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}, true},
		{"q", tea.KeyPressMsg{Code: 'q'}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBack(tt.msg); got != tt.want {
				t.Errorf("isBack = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEnter(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	if !isEnter(msg) {
		t.Error("expected isEnter = true for enter key")
	}
	msg2 := tea.KeyPressMsg{Code: 'a'}
	if isEnter(msg2) {
		t.Error("expected isEnter = false for 'a'")
	}
}

func TestIsUp(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyUp}
	if !isUp(msg) {
		t.Error("expected isUp = true for up key")
	}
	msgK := tea.KeyPressMsg{Code: 'k'}
	if !isUp(msgK) {
		t.Error("expected isUp = true for 'k'")
	}
}

func TestIsDown(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	if !isDown(msg) {
		t.Error("expected isDown = true for down key")
	}
	msgJ := tea.KeyPressMsg{Code: 'j'}
	if !isDown(msgJ) {
		t.Error("expected isDown = true for 'j'")
	}
}

func TestIsToggle(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeySpace}
	if !isToggle(msg) {
		t.Error("expected isToggle = true for space")
	}
}

func TestIsSelectAll(t *testing.T) {
	msg := tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	if !isSelectAll(msg) {
		t.Error("expected isSelectAll = true for ctrl+a")
	}
}
