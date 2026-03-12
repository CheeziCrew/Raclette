package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/CheeziCrew/curd"
)

// Common key checks — keeps Update methods clean.

func isBack(msg tea.KeyPressMsg) bool {
	return curd.IsBack(msg)
}

func isEnter(msg tea.KeyPressMsg) bool {
	return curd.IsEnter(msg)
}

func isUp(msg tea.KeyPressMsg) bool {
	return curd.IsUp(msg)
}

func isDown(msg tea.KeyPressMsg) bool {
	return curd.IsDown(msg)
}

func isToggle(msg tea.KeyPressMsg) bool {
	return curd.IsToggle(msg)
}

func isSelectAll(msg tea.KeyPressMsg) bool {
	return curd.IsSelectAll(msg)
}
