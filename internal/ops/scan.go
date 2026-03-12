package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ── Find Dependency ─────────────────────────────────────────────────

// DepMatch represents a dependency match in a repo.
type DepMatch struct {
	Repo      string
	GroupID   string
	Artifact  string
	Version   string
	InParent  bool
}

// FindDependency scans pom.xml files across repos for a dependency matching the query.
// Query can be a partial artifactId, groupId, or groupId:artifactId.
func FindDependency(dirs []string, query string) []DepMatch {
	query = strings.ToLower(query)
	var matches []DepMatch

	// Regex to match <dependency> blocks in pom.xml
	depRegex := regexp.MustCompile(`(?s)<dependency>\s*<groupId>([^<]+)</groupId>\s*<artifactId>([^<]+)</artifactId>(?:\s*<version>([^<]*)</version>)?`)
	// Also match parent
	parentRegex := regexp.MustCompile(`(?s)<parent>\s*<groupId>([^<]+)</groupId>\s*<artifactId>([^<]+)</artifactId>\s*<version>([^<]*)</version>`)

	for _, dir := range dirs {
		data, err := os.ReadFile(filepath.Join(dir, "pom.xml"))
		if err != nil {
			continue
		}
		content := string(data)
		repo := filepath.Base(dir)
		props := parsePomProperties(content)

		// Check parent
		if pm := parentRegex.FindStringSubmatch(content); len(pm) > 3 {
			if matchesQuery(pm[1], pm[2], query) {
				matches = append(matches, DepMatch{
					Repo:     repo,
					GroupID:  pm[1],
					Artifact: pm[2],
					Version:  resolveProperty(pm[3], props),
					InParent: true,
				})
			}
		}

		// Check dependencies
		for _, dm := range depRegex.FindAllStringSubmatch(content, -1) {
			if len(dm) < 3 {
				continue
			}
			groupID := dm[1]
			artifactID := dm[2]
			version := ""
			if len(dm) > 3 {
				version = resolveProperty(dm[3], props)
			}
			if matchesQuery(groupID, artifactID, query) {
				matches = append(matches, DepMatch{
					Repo:     repo,
					GroupID:  groupID,
					Artifact: artifactID,
					Version:  version,
				})
			}
		}
	}

	return matches
}

// propRegex matches <key>value</key> inside a <properties> block.
var propRegex = regexp.MustCompile(`<([a-zA-Z0-9._-]+)>([^<]+)</`)

// parsePomProperties extracts the <properties> block values from pom.xml content.
func parsePomProperties(content string) map[string]string {
	props := make(map[string]string)
	// Find the <properties> block.
	start := strings.Index(content, "<properties>")
	end := strings.Index(content, "</properties>")
	if start == -1 || end == -1 || end <= start {
		return props
	}
	block := content[start:end]
	for _, m := range propRegex.FindAllStringSubmatch(block, -1) {
		if len(m) > 2 {
			props[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return props
}

// resolveProperty resolves ${property.name} references using the properties map.
// Returns the resolved value, or "" if the property can't be resolved.
func resolveProperty(version string, props map[string]string) string {
	if !strings.HasPrefix(version, "${") || !strings.HasSuffix(version, "}") {
		return version
	}
	key := version[2 : len(version)-1]
	if val, ok := props[key]; ok {
		return val
	}
	// Built-in Maven properties
	if key == "project.version" || key == "pom.version" {
		return "" // can't resolve without project version context
	}
	return ""
}

func matchesQuery(groupID, artifactID, query string) bool {
	g := strings.ToLower(groupID)
	a := strings.ToLower(artifactID)
	full := g + ":" + a

	// Exact match on artifactId or groupId:artifactId
	if a == query || full == query {
		return true
	}
	// Partial match
	return strings.Contains(a, query) || strings.Contains(g, query)
}

// FormatDepResults formats dependency matches as lines for the TUI.
func FormatDepResults(matches []DepMatch) []string {
	if len(matches) == 0 {
		return []string{"No matches found."}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d match(es):", len(matches)))
	lines = append(lines, "")

	for _, m := range matches {
		ver := m.Version
		if ver == "" {
			ver = "(managed)"
		}
		tag := ""
		if m.InParent {
			tag = " [parent]"
		}
		lines = append(lines, fmt.Sprintf("  %-35s %s:%s %s%s", m.Repo, m.GroupID, m.Artifact, ver, tag))
	}

	return lines
}

// ── Name Index (fondue-style fuzzy matching) ────────────────────────

// NameIndex maps multiple normalized variants of a service name back to the
// canonical name, enabling fuzzy matching of spec filenames to services.
type NameIndex struct {
	nameMap map[string]string // normalized variant → canonical service name
}

// NewNameIndex builds a NameIndex from a set of service names.
// For each name it registers: exact, stripped hyphens, dotted, and
// singular/plural variants — matching fondue's approach.
func NewNameIndex(names []string) *NameIndex {
	m := make(map[string]string)
	for _, name := range names {
		lower := strings.ToLower(name)
		m[lower] = name

		stripped := strings.ReplaceAll(lower, "-", "")
		m[stripped] = name

		dotted := strings.ReplaceAll(lower, "-", ".")
		m[dotted] = name

		if strings.HasSuffix(lower, "s") {
			m[strings.TrimSuffix(lower, "s")] = name
			m[strings.TrimSuffix(stripped, "s")] = name
		} else {
			m[lower+"s"] = name
			m[stripped+"s"] = name
		}
	}
	return &NameIndex{nameMap: m}
}

// Resolve looks up a normalized name and returns the canonical service name,
// or "" if no match is found. Tries exact lowercase first, then stripped hyphens.
func (idx *NameIndex) Resolve(name string) string {
	lower := strings.ToLower(name)
	if svc, ok := idx.nameMap[lower]; ok {
		return svc
	}
	stripped := strings.ReplaceAll(lower, "-", "")
	if svc, ok := idx.nameMap[stripped]; ok {
		return svc
	}
	return ""
}

// ── Integration scanning (ported from fondue) ───────────────────────

// integration tracks a single outbound dependency discovered from source code.
type integration struct {
	clientID    string // e.g. "case-data"
	specVersion string // version from integration spec's info.version (empty if no spec)
	specFile    string // spec filename (e.g. "messaging-api-7.9.yaml")
	specPath    string // absolute path to spec file in consuming repo
}

var (
	clientIDRe        = regexp.MustCompile(`CLIENT_ID\s*=\s*"([^"]+)"`)
	integrationNameRe = regexp.MustCompile(`INTEGRATION_NAME\s*=\s*"([^"]+)"`)
	specFileCleanRe   = regexp.MustCompile(`(?i)(-api)?(-(v?\d+(\.\d+)*))?\.ya?ml$`)
)

// scanRepo walks src/main/java looking for CLIENT_ID and INTEGRATION_NAME
// constants in integration packages — discovering real integration points
// from source code rather than relying on spec filenames.
func scanRepo(repoPath string) []integration {
	integrationBase := filepath.Join(repoPath, "src", "main", "java")

	allPackages := make(map[string]bool)
	resolvedPkgs := make(map[string]bool)
	seen := make(map[string]bool)
	var integrations []integration

	filepath.Walk(integrationBase, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		rel, _ := filepath.Rel(integrationBase, path)
		if !strings.Contains(rel, "integration") {
			return nil
		}

		parts := strings.Split(rel, string(filepath.Separator))
		for _, p := range parts {
			if p == "db" {
				return nil
			}
		}

		pkg := extractPackageName(rel)
		if pkg != "" && !strings.HasSuffix(pkg, ".java") {
			allPackages[pkg] = true
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		if matches := clientIDRe.FindStringSubmatch(content); len(matches) > 1 {
			addIntegration(&integrations, seen, matches[1])
			if pkg != "" {
				resolvedPkgs[pkg] = true
			}
		}

		if matches := integrationNameRe.FindStringSubmatch(content); len(matches) > 1 {
			addIntegration(&integrations, seen, matches[1])
			if pkg != "" {
				resolvedPkgs[pkg] = true
			}
		}

		return nil
	})

	for pkg := range allPackages {
		if !resolvedPkgs[pkg] && !seen[pkg] {
			seen[pkg] = true
			integrations = append(integrations, integration{clientID: pkg})
		}
	}

	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].clientID < integrations[j].clientID
	})

	return integrations
}

func extractPackageName(relPath string) string {
	parts := strings.Split(relPath, string(filepath.Separator))
	for i, p := range parts {
		if p == "integration" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func addIntegration(integrations *[]integration, seen map[string]bool, clientID string) {
	if !seen[clientID] {
		seen[clientID] = true
		*integrations = append(*integrations, integration{clientID: clientID})
	}
}

// matchSpecVersions populates specVersion/specFile/specPath on integrations
// by reading pom.xml <inputSpec> entries and finding the actual spec files.
func matchSpecVersions(repoPath string, integrations []integration) {
	specFiles := InputSpecs(repoPath)
	if len(specFiles) == 0 {
		return
	}

	type specInfo struct {
		version  string
		filename string
		path     string
	}

	specVersions := make(map[string]specInfo) // normalized name → info
	for _, specFilename := range specFiles {
		specPath := findIntegrationSpecFile(repoPath, specFilename)
		if specPath == "" {
			continue
		}

		version := extractSpecVersion(specPath)
		if version == "" {
			continue
		}

		normalized := normalizeSpecFilename(specFilename)
		specVersions[normalized] = specInfo{
			version:  version,
			filename: specFilename,
			path:     specPath,
		}
	}

	for i := range integrations {
		normalizedID := strings.ToLower(strings.ReplaceAll(integrations[i].clientID, "-", ""))
		for specName, info := range specVersions {
			if specName == normalizedID {
				integrations[i].specVersion = info.version
				integrations[i].specFile = info.filename
				integrations[i].specPath = info.path
				break
			}
		}
	}
}

// findIntegrationSpecFile checks 3 directories for a spec file.
func findIntegrationSpecFile(repoPath, filename string) string {
	dirs := []string{
		"src/main/resources/integrations",
		"src/main/resources/integrations/rest",
		"src/main/resources/contract",
	}
	for _, dir := range dirs {
		path := filepath.Join(repoPath, dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func normalizeSpecFilename(filename string) string {
	name := specFileCleanRe.ReplaceAllString(filename, "")
	name = strings.TrimPrefix(name, "api-")
	return strings.ToLower(strings.ReplaceAll(name, "-", ""))
}

// ── Shared stale integration scanner ────────────────────────────────

// StaleIntegration represents one stale integration spec in a repo.
type StaleIntegration struct {
	RepoDir       string // absolute path to consuming repo
	Repo          string // basename for display
	SpecFile      string // spec filename
	SpecPath      string // absolute path to spec file in consuming repo
	SpecVersion   string // version in the spec's info.version
	TargetName    string // canonical service name
	TargetVersion string // POM version of target service
}

// ScanStaleIntegrations discovers all stale integration specs across repos
// using fondue's approach: scan Java source for integration annotations,
// match spec versions via pom.xml inputSpec entries, and compare against
// target service POM versions.
func ScanStaleIntegrations(dirs []string) []StaleIntegration {
	// Phase 1: build version index from pom.xml <version> and a NameIndex.
	svcVersions := make(map[string]string) // canonical name → POM version
	var svcNames []string

	for _, dir := range dirs {
		name := serviceNameFromDir(dir)
		ver := PomVersion(dir)
		if ver == "" {
			continue
		}
		svcVersions[name] = ver
		svcNames = append(svcNames, name)
	}

	idx := NewNameIndex(svcNames)

	// Phase 2: for each repo, discover integrations from source and find stale ones.
	var stale []StaleIntegration

	for _, dir := range dirs {
		repo := filepath.Base(dir)

		// Discover integrations from Java source code.
		integrations := scanRepo(dir)
		if len(integrations) == 0 {
			continue
		}

		// Populate spec versions from pom.xml inputSpec entries.
		matchSpecVersions(dir, integrations)

		// Normalize CLIENT_IDs — try trimming common suffixes.
		for i := range integrations {
			if idx.Resolve(integrations[i].clientID) == "" {
				lower := strings.ToLower(integrations[i].clientID)
				for _, suffix := range []string{"client", "integration", "service"} {
					trimmed := strings.TrimSuffix(lower, suffix)
					if trimmed != lower {
						if resolved := idx.Resolve(trimmed); resolved != "" {
							integrations[i].clientID = resolved
							break
						}
					}
				}
			}
		}

		// Compare each integration's spec version against the target service's POM version.
		for _, integ := range integrations {
			if integ.specVersion == "" {
				continue
			}
			targetName := idx.Resolve(integ.clientID)
			if targetName == "" {
				continue
			}
			targetVersion, ok := svcVersions[targetName]
			if !ok || targetVersion == "" {
				continue
			}
			if integ.specVersion != targetVersion {
				stale = append(stale, StaleIntegration{
					RepoDir:       dir,
					Repo:          repo,
					SpecFile:      integ.specFile,
					SpecPath:      integ.specPath,
					SpecVersion:   integ.specVersion,
					TargetName:    targetName,
					TargetVersion: targetVersion,
				})
			}
		}
	}

	return stale
}

// ── Find Stale Specs (public API) ───────────────────────────────────

// StaleSpec represents a stale integration spec in a repo.
type StaleSpec struct {
	Repo          string
	SpecFile      string
	SpecVersion   string
	TargetRepo    string
	TargetVersion string
}

// FindStaleSpecs scans repos for integration specs whose info.version doesn't
// match the target service's pom.xml <version>.
func FindStaleSpecs(dirs []string) []StaleSpec {
	staleList := ScanStaleIntegrations(dirs)

	var result []StaleSpec
	for _, s := range staleList {
		result = append(result, StaleSpec{
			Repo:          s.Repo,
			SpecFile:      s.SpecFile,
			SpecVersion:   s.SpecVersion,
			TargetRepo:    s.TargetName,
			TargetVersion: s.TargetVersion,
		})
	}
	return result
}

// specVersionRe is precompiled for extractSpecVersion (called 100s of times).
var specVersionRe = regexp.MustCompile(`(?m)^\s+version:\s*["']?([^"'\n\r]+)["']?\s*$`)

// extractSpecVersion reads info.version from an OpenAPI spec YAML header.
func extractSpecVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	header := strings.Join(strings.SplitN(string(data), "\n", 16), "\n")
	if m := specVersionRe.FindStringSubmatch(header); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func serviceNameFromDir(dir string) string {
	base := filepath.Base(dir)
	for _, prefix := range []string{"api-service-", "api-", "pw-"} {
		if strings.HasPrefix(base, prefix) {
			return strings.TrimPrefix(base, prefix)
		}
	}
	return base
}

// FormatStaleResults formats stale spec results as lines for the TUI.
func FormatStaleResults(stale []StaleSpec) []string {
	if len(stale) == 0 {
		return []string{"All integration specs are up to date."}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d stale spec(s):", len(stale)))
	lines = append(lines, "")

	for _, s := range stale {
		lines = append(lines, fmt.Sprintf("  %-35s %s: %s → %s", s.Repo, s.SpecFile, s.SpecVersion, s.TargetVersion))
	}

	return lines
}
