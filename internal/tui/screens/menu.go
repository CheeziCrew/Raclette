package screens

import (
	"github.com/CheeziCrew/raclette/internal/maven"
	"github.com/CheeziCrew/curd"
)

// MenuModel wraps curd.MenuModel for raclette.
type MenuModel = curd.MenuModel

// NewMenu creates a fresh menu model.
func NewMenu() curd.MenuModel {
	cmds := maven.Commands()
	items := make([]curd.MenuItem, len(cmds))
	for i, cmd := range cmds {
		items[i] = curd.MenuItem{
			Icon:    cmd.Icon,
			Name:    cmd.Name,
			Command: cmd.Name,
			Desc:    cmd.Description,
		}
	}

	return curd.NewMenuModel(curd.MenuConfig{
		Banner: []string{
			"                __    __  __     ",
			"  _______ _____/ /__ / /_/ /____",
			" / __/ _ `/ __/ / -_) __/ __/ -_)",
			"/_/  \\_,_/\\__/_/\\__/\\__/\\__/\\__/",
		},
		Tagline: "melt through your dept44 repos",
		Palette: curd.RaclettePalette,
		Items:   items,
	})
}
