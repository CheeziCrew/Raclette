package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Result tracks what happened per repo.
type Result struct {
	Repo    string
	Changed bool
	Message string
	Err     error
}

// ── Replace Dependency Version ──────────────────────────────────────

// ReplaceDependencyVersion bumps the version of a specific dependency in pom.xml.
func ReplaceDependencyVersion(dirs []string, groupID, artifactID, newVersion string) []Result {
	// Match a full <dependency> block with this groupId + artifactId and replace its version.
	pattern := fmt.Sprintf(
		`(?s)(<dependency>\s*<groupId>%s</groupId>\s*<artifactId>%s</artifactId>\s*<version>)([^<]*)(</version>)`,
		regexp.QuoteMeta(groupID), regexp.QuoteMeta(artifactID),
	)
	re := regexp.MustCompile(pattern)

	var results []Result
	for _, dir := range dirs {
		results = append(results, replaceInPom(dir, re, newVersion))
	}
	return results
}

// ── Swap Dependency ─────────────────────────────────────────────────

// SwapDependency replaces one dependency with another (different groupId/artifactId/version).
// Preserves <scope> and <exclusions> blocks from the original dependency.
func SwapDependency(dirs []string, oldGroupID, oldArtifactID, newGroupID, newArtifactID, newVersion string) []Result {
	pattern := fmt.Sprintf(
		`(?s)<dependency>\s*<groupId>%s</groupId>\s*<artifactId>%s</artifactId>(.*?)</dependency>`,
		regexp.QuoteMeta(oldGroupID), regexp.QuoteMeta(oldArtifactID),
	)
	re := regexp.MustCompile(pattern)

	scopeRe := regexp.MustCompile(`(?s)<scope>[^<]*</scope>`)
	exclusionsRe := regexp.MustCompile(`(?s)<exclusions>.*?</exclusions>`)

	var results []Result
	for _, dir := range dirs {
		results = append(results, swapDepInRepo(dir, re, scopeRe, exclusionsRe, newGroupID, newArtifactID, newVersion))
	}
	return results
}

func swapDepInRepo(dir string, re, scopeRe, exclusionsRe *regexp.Regexp, newGroupID, newArtifactID, newVersion string) Result {
	repo := filepath.Base(dir)
	pomPath := filepath.Join(dir, "pom.xml")

	data, err := os.ReadFile(pomPath)
	if err != nil {
		return Result{Repo: repo, Err: fmt.Errorf("read pom.xml: %w", err)}
	}

	content := string(data)
	if !re.MatchString(content) {
		return Result{Repo: repo, Message: "dependency not found"}
	}

	updated := re.ReplaceAllStringFunc(content, func(match string) string {
		return buildSwappedDep(re, scopeRe, exclusionsRe, match, newGroupID, newArtifactID, newVersion)
	})

	if updated == content {
		return Result{Repo: repo, Message: "no change needed"}
	}

	if err := os.WriteFile(pomPath, []byte(updated), 0644); err != nil {
		return Result{Repo: repo, Err: fmt.Errorf("write pom.xml: %w", err)}
	}

	return Result{Repo: repo, Changed: true, Message: "dependency swapped"}
}

func buildSwappedDep(re, scopeRe, exclusionsRe *regexp.Regexp, match, newGroupID, newArtifactID, newVersion string) string {
	inner := re.FindStringSubmatch(match)
	tail := ""
	if len(inner) > 1 {
		if s := scopeRe.FindString(inner[1]); s != "" {
			tail += "\n\t\t\t" + s
		}
		if e := exclusionsRe.FindString(inner[1]); e != "" {
			tail += "\n\t\t\t" + e
		}
	}
	return fmt.Sprintf("<dependency>\n\t\t\t<groupId>%s</groupId>\n\t\t\t<artifactId>%s</artifactId>\n\t\t\t<version>%s</version>%s\n\t\t</dependency>",
		newGroupID, newArtifactID, newVersion, tail)
}

// ── Bump Parent Version ─────────────────────────────────────────────

// BumpParentVersion updates the parent version in pom.xml.
func BumpParentVersion(dirs []string, newVersion string) []Result {
	pattern := `(?s)(<parent>\s*<groupId>[^<]*</groupId>\s*<artifactId>[^<]*</artifactId>\s*<version>)([^<]*)(</version>)`
	re := regexp.MustCompile(pattern)

	var results []Result
	for _, dir := range dirs {
		results = append(results, replaceInPom(dir, re, newVersion))
	}
	return results
}

// ── Update Integration Specs ────────────────────────────────────────

// UpdateIntegrationSpecs copies the target service's own OpenAPI spec into
// repos that consume it as an integration spec.
// Uses ScanStaleIntegrations (shared with FindStaleSpecs) to detect staleness,
// then copies the source spec if its version matches PomVersion (i.e. it's
// been regenerated). If the source spec is also stale, reports that the
// target service needs "Update Own Specs" first.
// svcEntry holds a service's own spec path and POM version.
type svcEntry struct {
	specPath   string
	pomVersion string
}

func UpdateIntegrationSpecs(dirs []string) []Result {
	svcIndex := buildSvcIndex(dirs)
	staleList := ScanStaleIntegrations(dirs)
	byRepo, repoOrder := groupStaleByRepo(staleList)

	var results []Result
	for _, repo := range repoOrder {
		if r, ok := updateRepoSpecs(repo, byRepo[repo], svcIndex); ok {
			results = append(results, r)
		}
	}

	if len(results) == 0 {
		results = append(results, Result{
			Message: "All integration specs are already up to date.",
		})
	}
	return results
}

func buildSvcIndex(dirs []string) map[string]svcEntry {
	svcIndex := make(map[string]svcEntry)
	for _, dir := range dirs {
		name := serviceNameFromDir(dir)
		ver := PomVersion(dir)
		if ver == "" {
			continue
		}
		entry := svcEntry{pomVersion: ver}
		if ownSpec := findOwnSpec(dir); ownSpec != "" {
			entry.specPath = ownSpec
		}
		svcIndex[name] = entry
	}
	return svcIndex
}

func groupStaleByRepo(staleList []StaleIntegration) (map[string][]StaleIntegration, []string) {
	byRepo := make(map[string][]StaleIntegration)
	var repoOrder []string
	for _, s := range staleList {
		if _, ok := byRepo[s.Repo]; !ok {
			repoOrder = append(repoOrder, s.Repo)
		}
		byRepo[s.Repo] = append(byRepo[s.Repo], s)
	}
	return byRepo, repoOrder
}

func updateRepoSpecs(repo string, items []StaleIntegration, svcIndex map[string]svcEntry) (Result, bool) {
	updated := 0
	skipped := 0
	var msgs []string

	for _, s := range items {
		msg, status := updateOneSpec(s, svcIndex)
		if msg != "" {
			msgs = append(msgs, msg)
		}
		switch status {
		case specUpdated:
			updated++
		case specSkipped:
			skipped++
		}
	}

	if updated == 0 && skipped == 0 {
		return Result{}, false
	}
	return Result{
		Repo:    repo,
		Changed: updated > 0,
		Message: fmt.Sprintf("Updated %d, skipped %d:\n%s", updated, skipped, strings.Join(msgs, "\n")),
	}, true
}

type specUpdateStatus int

const (
	specIgnored specUpdateStatus = iota
	specUpdated
	specSkipped
)

func updateOneSpec(s StaleIntegration, svcIndex map[string]svcEntry) (string, specUpdateStatus) {
	if s.SpecPath == "" {
		return fmt.Sprintf("  ⚠ %s: no spec file path for %s", s.SpecFile, s.TargetName), specSkipped
	}

	svc, ok := svcIndex[s.TargetName]
	if !ok || svc.pomVersion == "" {
		return "", specIgnored
	}

	if svc.specPath == "" {
		return fmt.Sprintf("  ⚠ %s: no source spec found for %s", s.SpecFile, s.TargetName), specSkipped
	}

	sourceVersion := extractSpecVersion(svc.specPath)
	if sourceVersion != svc.pomVersion {
		return fmt.Sprintf("  ⚠ %s: source %s spec is stale (%s, need %s) — run 'Update Own Specs' on %s first",
			s.SpecFile, s.TargetName, sourceVersion, svc.pomVersion, s.TargetName), specSkipped
	}

	sourceData, err := os.ReadFile(svc.specPath)
	if err != nil {
		return fmt.Sprintf("  ✗ %s: failed to read source: %v", s.SpecFile, err), specIgnored
	}

	if err := os.WriteFile(s.SpecPath, sourceData, 0644); err != nil {
		return fmt.Sprintf("  ✗ %s: failed to write: %v", s.SpecFile, err), specIgnored
	}

	return fmt.Sprintf("  ✓ %s: %s → %s", s.SpecFile, s.SpecVersion, svc.pomVersion), specUpdated
}

// findOwnSpec locates a repo's own OpenAPI spec file.
// Searches three locations (in priority order):
//   - src/integration-test/resources (36 repos)
//   - src/main/resources            (12 repos: case-data, templating, document, …)
//   - src/test/resources             (12 repos)
func findOwnSpec(repoDir string) string {
	roots := []string{
		filepath.Join(repoDir, "src", "integration-test", "resources"),
		filepath.Join(repoDir, "src", "main", "resources"),
		filepath.Join(repoDir, "src", "test", "resources"),
	}
	for _, root := range roots {
		if found := findSpecIn(root); found != "" {
			return found
		}
	}
	return ""
}

// findSpecIn walks a directory tree looking for an openapi/api-docs YAML file
// that has a parseable info.version header.
func findSpecIn(root string) string {
	var found string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		if strings.HasPrefix(name, "openapi") || strings.HasPrefix(name, "api-docs") {
			if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
				if extractSpecVersion(path) != "" {
					found = path
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	return found
}

// ── Helpers ─────────────────────────────────────────────────────────

func replaceInPom(dir string, re *regexp.Regexp, newVersion string) Result {
	repo := filepath.Base(dir)
	pomPath := filepath.Join(dir, "pom.xml")

	data, err := os.ReadFile(pomPath)
	if err != nil {
		return Result{Repo: repo, Err: fmt.Errorf("read pom.xml: %w", err)}
	}

	content := string(data)
	if !re.MatchString(content) {
		return Result{Repo: repo, Message: "no match"}
	}

	// Escape $ in newVersion so it's not treated as a backreference.
	safeVersion := strings.ReplaceAll(newVersion, "$", "$$")
	updated := re.ReplaceAllString(content, "${1}"+safeVersion+"${3}")
	if updated == content {
		return Result{Repo: repo, Message: "already at target version"}
	}

	if err := os.WriteFile(pomPath, []byte(updated), 0644); err != nil {
		return Result{Repo: repo, Err: fmt.Errorf("write pom.xml: %w", err)}
	}

	return Result{Repo: repo, Changed: true, Message: fmt.Sprintf("updated to %s", newVersion)}
}

// FormatResults formats transform results as lines for the TUI.
func FormatResults(results []Result) []string {
	var lines []string
	changed := 0
	for _, r := range results {
		if r.Err != nil {
			lines = append(lines, fmt.Sprintf("  ✗ %-35s %v", r.Repo, r.Err))
		} else if r.Changed {
			changed++
			lines = append(lines, fmt.Sprintf("  ✓ %-35s %s", r.Repo, r.Message))
		} else if r.Repo != "" {
			lines = append(lines, fmt.Sprintf("  · %-35s %s", r.Repo, r.Message))
		} else {
			lines = append(lines, fmt.Sprintf("  %s", r.Message))
		}
	}

	header := fmt.Sprintf("Changed %d repo(s):", changed)
	return append([]string{header, ""}, lines...)
}
