package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/raclette/internal/ops"
)

// ── Styled formatters for scan/transform output ─────────────────────
// These live in the TUI layer (not ops) because they use lipgloss styles.

var styles = curd.RaclettePalette.Styles()

// depVersion returns the version for display, or "" if managed/inherited.
func depVersion(v string) string {
	if v == "" || strings.HasPrefix(v, "${") {
		return ""
	}
	return v
}

// formatDepByRepo renders dependency matches sorted by repo name.
func formatDepByRepo(matches []ops.DepMatch) string {
	if len(matches) == 0 {
		return styles.ResultDim.Render("No matches found.")
	}

	sorted := make([]ops.DepMatch, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Repo < sorted[j].Repo
	})

	var b strings.Builder
	b.WriteString(depHeader(len(matches), "by repo"))

	for _, m := range sorted {
		b.WriteString("  ")
		b.WriteString(styles.NameStyle.Render(fmt.Sprintf("%-35s", m.Repo)))
		b.WriteString(" ")
		b.WriteString(styles.AccentStyle.Render(m.GroupID + ":" + m.Artifact))
		if v := depVersion(m.Version); v != "" {
			b.WriteString(" ")
			b.WriteString(styles.ResultDim.Render(v))
		}
		if m.InParent {
			b.WriteString(" ")
			b.WriteString(styles.ResultAccent.Render("[parent]"))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// formatDepByDep renders dependency matches grouped by dependency.
func formatDepByDep(matches []ops.DepMatch) string {
	if len(matches) == 0 {
		return styles.ResultDim.Render("No matches found.")
	}

	groups, order := groupMatchesByDep(matches)

	var b strings.Builder
	b.WriteString(depHeader(len(matches), "by dep"))

	for i, key := range order {
		g := groups[key]
		writeDepGroup(&b, g)
		if i < len(order)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

type depGroup struct {
	key     string
	matches []ops.DepMatch
}

func groupMatchesByDep(matches []ops.DepMatch) (map[string]*depGroup, []string) {
	groups := make(map[string]*depGroup)
	var order []string
	for _, m := range matches {
		key := m.GroupID + ":" + m.Artifact
		if g, ok := groups[key]; ok {
			g.matches = append(g.matches, m)
		} else {
			groups[key] = &depGroup{key: key, matches: []ops.DepMatch{m}}
			order = append(order, key)
		}
	}
	sort.Strings(order)
	return groups, order
}

func writeDepGroup(b *strings.Builder, g *depGroup) {
	b.WriteString(styles.AccentStyle.Render(g.key))
	b.WriteString(styles.ResultDim.Render(fmt.Sprintf("  (%d)", len(g.matches))))
	b.WriteString("\n")

	sort.Slice(g.matches, func(a, c int) bool {
		return g.matches[a].Repo < g.matches[c].Repo
	})
	for _, m := range g.matches {
		b.WriteString("    ")
		b.WriteString(styles.NameStyle.Render(fmt.Sprintf("%-35s", m.Repo)))
		if v := depVersion(m.Version); v != "" {
			b.WriteString(" ")
			b.WriteString(styles.ResultDim.Render(v))
		}
		if m.InParent {
			b.WriteString(" ")
			b.WriteString(styles.ResultAccent.Render("[parent]"))
		}
		b.WriteString("\n")
	}
}

func depHeader(count int, mode string) string {
	return styles.CountStyle.Render(fmt.Sprintf("%d", count)) +
		styles.AccentStyle.Render(" match(es) ") +
		styles.ResultDim.Render(mode) +
		"  " + styles.ResultDim.Render("tab to toggle") +
		"\n\n"
}

// formatStaleResults renders stale spec results with lipgloss styling.
func formatStaleResults(stale []ops.StaleSpec) string {
	if len(stale) == 0 {
		return styles.ResultOk.Render("✔ All integration specs are up to date.")
	}

	var b strings.Builder

	// Header.
	b.WriteString(styles.CountStyle.Render(fmt.Sprintf("%d", len(stale))))
	b.WriteString(" ")
	b.WriteString(styles.ResultFail.Render("stale"))
	b.WriteString(styles.AccentStyle.Render(" spec(s)"))
	b.WriteString("\n\n")

	for _, s := range stale {
		b.WriteString("  ")
		b.WriteString(styles.NameStyle.Render(fmt.Sprintf("%-35s", s.Repo)))
		b.WriteString(" ")
		b.WriteString(styles.AccentStyle.Render(s.SpecFile))
		b.WriteString("  ")
		b.WriteString(styles.ResultDim.Render(s.SpecVersion))
		b.WriteString(styles.ResultOk.Render(" → " + s.TargetVersion))
		b.WriteString("\n")
	}

	return b.String()
}

// formatTransformResults renders transform results with lipgloss styling.
func formatTransformResults(results []ops.Result) string {
	var b strings.Builder

	changed := 0
	for _, r := range results {
		if r.Changed {
			changed++
		}
	}

	// Header.
	b.WriteString(styles.CountStyle.Render(fmt.Sprintf("%d", changed)))
	b.WriteString(styles.AccentStyle.Render(" repo(s) changed"))
	b.WriteString("\n\n")

	for _, r := range results {
		if r.Err != nil {
			b.WriteString("  ")
			b.WriteString(styles.ResultFail.Render("✗"))
			b.WriteString(" ")
			b.WriteString(styles.NameStyle.Render(fmt.Sprintf("%-35s", r.Repo)))
			b.WriteString(" ")
			b.WriteString(styles.ResultFail.Render(r.Err.Error()))
			b.WriteString("\n")
		} else if r.Changed {
			b.WriteString("  ")
			b.WriteString(styles.ResultOk.Render("✔"))
			b.WriteString(" ")
			b.WriteString(styles.NameStyle.Render(fmt.Sprintf("%-35s", r.Repo)))
			b.WriteString(" ")
			b.WriteString(styles.ResultDim.Render(r.Message))
			b.WriteString("\n")
		} else if r.Repo != "" {
			b.WriteString("  ")
			b.WriteString(styles.ResultDim.Render("·"))
			b.WriteString(" ")
			b.WriteString(styles.ResultDim.Render(fmt.Sprintf("%-35s", r.Repo)))
			b.WriteString(" ")
			b.WriteString(styles.ResultDim.Render(r.Message))
			b.WriteString("\n")
		} else {
			// No repo (e.g. "All integration specs are already up to date.")
			b.WriteString("  ")
			b.WriteString(styles.ResultDim.Render(r.Message))
			b.WriteString("\n")
		}
	}

	return b.String()
}
