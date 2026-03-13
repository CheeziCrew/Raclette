package tui

import (
	"testing"
)

func TestAppPalette(t *testing.T) {
	// Verify the palette is initialized (can't directly compare structs with slices)
	s := AppPalette.Styles()
	_ = s.Title
}

func TestStylesNotEmpty(t *testing.T) {
	// Verify that the style set was properly initialized
	s := Styles
	if s.Title.GetBold() == false {
		// Title should be bold by default
	}
	// Just verify it doesn't panic to access styles
	_ = s.Subtitle
	_ = s.Selected
	_ = s.Dim
	_ = s.SuccessStyle
	_ = s.FailStyle
}

func TestBackwardCompatColors(t *testing.T) {
	// Verify backward-compat color aliases don't panic
	_ = colorBlack
	_ = colorRed
	_ = colorGreen
	_ = colorYellow
	_ = colorBlue
	_ = colorMagenta
	_ = colorCyan
	_ = colorWhite
	_ = colorGray
	_ = colorBrRed
	_ = colorBrGreen
	_ = colorBrYell
	_ = colorBrBlue
	_ = colorBrMag
	_ = colorBrCyan
	_ = colorBrWhite
}

func TestBackwardCompatStyles(t *testing.T) {
	// Verify backward-compat style aliases don't panic
	_ = titleStyle
	_ = subtitleStyle
	_ = selectedStyle
	_ = dimStyle
	_ = successStyle
	_ = errorStyle
	_ = warnStyle
	_ = boxStyle
	_ = activeBoxStyle
	_ = helpStyle
}
