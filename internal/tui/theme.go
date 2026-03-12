package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/CheeziCrew/curd"
)

// App palette and derived styles.
var (
	AppPalette = curd.RaclettePalette
	Styles     = AppPalette.Styles()
)

// Backward-compat color aliases used by other files in this package.
var (
	colorBlack   = curd.ColorBg
	colorRed     = curd.ColorRed
	colorGreen   = curd.ColorGreen
	colorYellow  = curd.ColorYellow
	colorBlue    = curd.ColorBlue
	colorMagenta = curd.ColorMagenta
	colorCyan    = curd.ColorCyan
	colorWhite   = curd.ColorFg
	colorGray    = curd.ColorGray
	colorBrRed   = curd.ColorBrRed
	colorBrGreen = curd.ColorBrGreen
	colorBrYell  = curd.ColorBrYellow
	colorBrBlue  = curd.ColorBrBlue
	colorBrMag   = curd.ColorBrMag
	colorBrCyan  = curd.ColorBrCyan
	colorBrWhite = curd.ColorBrWhite
)

// Backward-compat style aliases.
var (
	titleStyle    = Styles.Title
	subtitleStyle = Styles.Subtitle
	selectedStyle = Styles.Selected
	dimStyle      = Styles.Dim
	successStyle  = Styles.SuccessStyle
	errorStyle    = Styles.FailStyle
	warnStyle     = lipgloss.NewStyle().Foreground(curd.ColorBrYellow)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(curd.ColorYellow).Padding(0, 2)
	activeBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(curd.ColorBrYellow).Padding(0, 2)
	helpStyle     = Styles.HelpMargin
)
