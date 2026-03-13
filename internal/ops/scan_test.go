package ops

import (
	"os"
	"path/filepath"
	"testing"
)

// ── TestParsePomProperties ──────────────────────────────────────────

func TestParsePomProperties(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name: "extracts key-value pairs",
			content: `<project>
  <properties>
    <spring.version>6.0.0</spring.version>
    <java.version>17</java.version>
  </properties>
</project>`,
			want: map[string]string{
				"spring.version": "6.0.0",
				"java.version":  "17",
			},
		},
		{
			name:    "no properties block",
			content: `<project><version>1.0</version></project>`,
			want:    map[string]string{},
		},
		{
			name:    "malformed properties",
			content: `<project><properties>not xml at all</project>`,
			want:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePomProperties(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d props, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("props[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// ── TestResolveProperty ─────────────────────────────────────────────

func TestResolveProperty(t *testing.T) {
	props := map[string]string{"my.version": "2.0.0"}

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"literal version", "1.0.0", "1.0.0"},
		{"resolved property", "${my.version}", "2.0.0"},
		{"unknown property", "${unknown}", ""},
		{"project.version", "${project.version}", ""},
		{"non-property string", "3.1.4", "3.1.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProperty(tt.version, props)
			if got != tt.want {
				t.Errorf("resolveProperty(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

// ── TestMatchesQuery ────────────────────────────────────────────────

func TestMatchesQuery(t *testing.T) {
	tests := []struct {
		name       string
		groupID    string
		artifactID string
		query      string
		want       bool
	}{
		{"exact artifactId", "com.foo", "bar", "bar", true},
		{"exact full match", "com.foo", "bar", "com.foo:bar", true},
		{"partial match", "com.foo", "spring-core", "spring", true},
		{"no match", "com.foo", "bar", "baz", false},
		{"case insensitive", "com.foo", "Bar", "bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesQuery(tt.groupID, tt.artifactID, tt.query)
			if got != tt.want {
				t.Errorf("matchesQuery(%q, %q, %q) = %v, want %v",
					tt.groupID, tt.artifactID, tt.query, got, tt.want)
			}
		})
	}
}

// ── TestExtractPackageName ──────────────────────────────────────────

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		want    string
	}{
		{"integration subpackage", filepath.Join("integration", "foo", "Bar.java"), "foo"},
		{"no integration segment", filepath.Join("other", "Bar.java"), ""},
		{"integration is last segment", filepath.Join("integration"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPackageName(tt.relPath)
			if got != tt.want {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.relPath, got, tt.want)
			}
		})
	}
}

// ── TestNormalizeSpecFilename ────────────────────────────────────────

func TestNormalizeSpecFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"with api and version", "messaging-api-7.9.yaml", "messaging"},
		{"with version only", "case-data-1.0.yml", "casedata"},
		{"plain yaml", "foo.yaml", "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSpecFilename(tt.filename)
			if got != tt.want {
				t.Errorf("normalizeSpecFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

// ── TestServiceNameFromDir ──────────────────────────────────────────

func TestServiceNameFromDir(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"api-service prefix", "/path/api-service-foo", "foo"},
		{"api prefix", "/path/api-bar", "bar"},
		{"pw prefix", "/path/pw-baz", "baz"},
		{"plain name", "/path/plain", "plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceNameFromDir(tt.dir)
			if got != tt.want {
				t.Errorf("serviceNameFromDir(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

// ── TestContainsDBSegment ───────────────────────────────────────────

func TestContainsDBSegment(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		want bool
	}{
		{"db segment", filepath.Join("foo", "db", "Bar.java"), true},
		{"database is not db", filepath.Join("foo", "database", "Bar.java"), false},
		{"no db", filepath.Join("foo", "bar", "Baz.java"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDBSegment(tt.rel)
			if got != tt.want {
				t.Errorf("containsDBSegment(%q) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}

// ── TestFormatDepResults ────────────────────────────────────────────

func TestFormatDepResults(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		lines := FormatDepResults(nil)
		if len(lines) != 1 || lines[0] != "No matches found." {
			t.Errorf("got %v, want [No matches found.]", lines)
		}
	})

	t.Run("with matches", func(t *testing.T) {
		matches := []DepMatch{
			{Repo: "my-repo", GroupID: "com.foo", Artifact: "bar", Version: "1.0"},
		}
		lines := FormatDepResults(matches)
		if len(lines) < 3 {
			t.Fatalf("expected at least 3 lines, got %d", len(lines))
		}
		if lines[0] != "Found 1 match(es):" {
			t.Errorf("header = %q", lines[0])
		}
	})
}

// ── TestFormatStaleResults ──────────────────────────────────────────

func TestFormatStaleResults(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		lines := FormatStaleResults(nil)
		if len(lines) != 1 || lines[0] != "All integration specs are up to date." {
			t.Errorf("got %v", lines)
		}
	})

	t.Run("with stale", func(t *testing.T) {
		stale := []StaleSpec{
			{Repo: "r", SpecFile: "s.yaml", SpecVersion: "1.0", TargetRepo: "t", TargetVersion: "2.0"},
		}
		lines := FormatStaleResults(stale)
		if len(lines) < 3 {
			t.Fatalf("expected at least 3 lines, got %d", len(lines))
		}
		if lines[0] != "Found 1 stale spec(s):" {
			t.Errorf("header = %q", lines[0])
		}
	})
}

// ── TestFindDependency ──────────────────────────────────────────────

func TestFindDependency(t *testing.T) {
	dir := t.TempDir()
	pom := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <dependencies>
    <dependency>
      <groupId>org.springframework</groupId>
      <artifactId>spring-core</artifactId>
      <version>6.0.0</version>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	t.Run("match found", func(t *testing.T) {
		matches := FindDependency([]string{dir}, "spring-core")
		if len(matches) == 0 {
			t.Fatal("expected at least one match")
		}
		if matches[0].Artifact != "spring-core" {
			t.Errorf("artifact = %q, want spring-core", matches[0].Artifact)
		}
		if matches[0].Version != "6.0.0" {
			t.Errorf("version = %q, want 6.0.0", matches[0].Version)
		}
	})

	t.Run("no match", func(t *testing.T) {
		matches := FindDependency([]string{dir}, "nonexistent")
		if len(matches) != 0 {
			t.Errorf("expected no matches, got %d", len(matches))
		}
	})
}

// ── TestNewNameIndex_Resolve ────────────────────────────────────────

func TestNewNameIndex_Resolve(t *testing.T) {
	idx := NewNameIndex([]string{"case-data", "messaging"})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"exact", "case-data", "case-data"},
		{"stripped hyphens", "casedata", "case-data"},
		{"dotted", "case.data", "case-data"},
		{"plural match", "messagings", "messaging"},
		{"unknown", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idx.Resolve(tt.input)
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── TestExtractSpecVersion ──────────────────────────────────────────

func TestExtractSpecVersion(t *testing.T) {
	t.Run("plain version", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "openapi.yaml")
		content := `openapi: 3.0.1
info:
  title: My API
  version: 1.2.3
paths: {}`
		os.WriteFile(specFile, []byte(content), 0644)

		got := extractSpecVersion(specFile)
		if got != "1.2.3" {
			t.Errorf("got %q, want 1.2.3", got)
		}
	})

	t.Run("quoted version", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "openapi.yaml")
		content := `openapi: 3.0.1
info:
  title: My API
  version: "4.5.6"
paths: {}`
		os.WriteFile(specFile, []byte(content), 0644)

		got := extractSpecVersion(specFile)
		if got != "4.5.6" {
			t.Errorf("got %q, want 4.5.6", got)
		}
	})

	t.Run("no version", func(t *testing.T) {
		dir := t.TempDir()
		specFile := filepath.Join(dir, "openapi.yaml")
		os.WriteFile(specFile, []byte("openapi: 3.0.1\ninfo:\n  title: X\n"), 0644)

		got := extractSpecVersion(specFile)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("bad file", func(t *testing.T) {
		got := extractSpecVersion("/nonexistent/file.yaml")
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

// ── TestFormatDepResults with parent ────────────────────────────────

func TestFormatDepResults_WithParent(t *testing.T) {
	matches := []DepMatch{
		{Repo: "r", GroupID: "g", Artifact: "a", Version: "", InParent: true},
	}
	lines := FormatDepResults(matches)
	joined := ""
	for _, l := range lines {
		joined += l
	}
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines, got %d", len(lines))
	}
}

// ── TestNewNameIndex_SingularPlural ──────────────────────────────────

func TestNewNameIndex_SingularToPlural(t *testing.T) {
	idx := NewNameIndex([]string{"case-data"})
	// "case-data" does NOT end with 's', so "case-datas" should resolve
	got := idx.Resolve("case-datas")
	if got != "case-data" {
		t.Errorf("Resolve('case-datas') = %q, want 'case-data'", got)
	}
}

func TestNewNameIndex_PluralToSingular(t *testing.T) {
	idx := NewNameIndex([]string{"documents"})
	// "documents" ends with 's', so "document" should resolve
	got := idx.Resolve("document")
	if got != "documents" {
		t.Errorf("Resolve('document') = %q, want 'documents'", got)
	}
}

// ── TestResolve_StrippedFallback ────────────────────────────────────

func TestResolve_StrippedFallback(t *testing.T) {
	idx := NewNameIndex([]string{"case-data"})
	// Try resolving with a hyphen that gets stripped as fallback
	got := idx.Resolve("CASE-DATA")
	if got != "case-data" {
		t.Errorf("Resolve('CASE-DATA') = %q, want 'case-data'", got)
	}
}

// ── TestFindDependency parent match ─────────────────────────────────

func TestFindDependency_ParentMatch(t *testing.T) {
	dir := t.TempDir()
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	matches := FindDependency([]string{dir}, "dept44")
	if len(matches) == 0 {
		t.Fatal("expected at least one match on parent")
	}
	if !matches[0].InParent {
		t.Error("expected InParent=true")
	}
}

func TestFindDependency_MissingPom(t *testing.T) {
	dir := t.TempDir()
	matches := FindDependency([]string{dir}, "anything")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for missing pom.xml, got %d", len(matches))
	}
}

// ── TestFindStaleSpecs empty ────────────────────────────────────────

func TestFindStaleSpecs_NoRepos(t *testing.T) {
	stale := FindStaleSpecs(nil)
	if len(stale) != 0 {
		t.Errorf("expected 0 stale specs, got %d", len(stale))
	}
}

// ── TestScanRepo_NoIntegrationCode ──────────────────────────────────

func TestScanRepo_NoIntegrationCode(t *testing.T) {
	dir := t.TempDir()
	integrations := scanRepo(dir)
	if len(integrations) != 0 {
		t.Errorf("expected 0 integrations, got %d", len(integrations))
	}
}

func TestScanRepo_WithClientID(t *testing.T) {
	dir := t.TempDir()
	javaDir := filepath.Join(dir, "src", "main", "java", "se", "sundsvall", "myservice", "integration", "messaging")
	os.MkdirAll(javaDir, 0755)

	javaContent := `package se.sundsvall.myservice.integration.messaging;
public class MessagingClient {
    private static final String CLIENT_ID = "messaging";
}
`
	os.WriteFile(filepath.Join(javaDir, "MessagingClient.java"), []byte(javaContent), 0644)

	integrations := scanRepo(dir)
	if len(integrations) == 0 {
		t.Fatal("expected at least one integration")
	}
	found := false
	for _, i := range integrations {
		if i.clientID == "messaging" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'messaging' integration")
	}
}

func TestScanRepo_WithIntegrationName(t *testing.T) {
	dir := t.TempDir()
	javaDir := filepath.Join(dir, "src", "main", "java", "se", "sundsvall", "myservice", "integration", "casedata")
	os.MkdirAll(javaDir, 0755)

	javaContent := `package se.sundsvall.myservice.integration.casedata;
public class CaseDataConfig {
    private static final String INTEGRATION_NAME = "case-data";
}
`
	os.WriteFile(filepath.Join(javaDir, "CaseDataConfig.java"), []byte(javaContent), 0644)

	integrations := scanRepo(dir)
	found := false
	for _, i := range integrations {
		if i.clientID == "case-data" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'case-data' integration")
	}
}

func TestScanRepo_SkipsDBSegment(t *testing.T) {
	dir := t.TempDir()
	javaDir := filepath.Join(dir, "src", "main", "java", "se", "sundsvall", "myservice", "integration", "db")
	os.MkdirAll(javaDir, 0755)

	javaContent := `package se.sundsvall.myservice.integration.db;
public class DatabaseClient {
    private static final String CLIENT_ID = "dbclient";
}
`
	os.WriteFile(filepath.Join(javaDir, "DatabaseClient.java"), []byte(javaContent), 0644)

	integrations := scanRepo(dir)
	for _, i := range integrations {
		if i.clientID == "dbclient" {
			t.Error("should skip integrations in db segment")
		}
	}
}

// ── TestAddIntegration ──────────────────────────────────────────────

func TestAddIntegration_Dedup(t *testing.T) {
	var integrations []integration
	seen := make(map[string]bool)
	addIntegration(&integrations, seen, "foo")
	addIntegration(&integrations, seen, "foo")
	if len(integrations) != 1 {
		t.Errorf("expected 1 integration after dedup, got %d", len(integrations))
	}
}

// ── TestNormalizeClientIDs ──────────────────────────────────────────

func TestNormalizeClientIDs_SuffixStrip(t *testing.T) {
	idx := NewNameIndex([]string{"messaging"})
	integrations := []integration{
		{clientID: "messagingclient"},
	}
	normalizeClientIDs(integrations, idx)
	if integrations[0].clientID != "messaging" {
		t.Errorf("expected normalized clientID 'messaging', got %q", integrations[0].clientID)
	}
}

func TestNormalizeClientIDs_NoStripping(t *testing.T) {
	idx := NewNameIndex([]string{"messaging"})
	integrations := []integration{
		{clientID: "messaging"},
	}
	normalizeClientIDs(integrations, idx)
	if integrations[0].clientID != "messaging" {
		t.Errorf("expected unchanged clientID 'messaging', got %q", integrations[0].clientID)
	}
}

// ── TestFindIntegrationSpecFile ─────────────────────────────────────

func TestFindIntegrationSpecFile_Found(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "main", "resources", "integrations")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "messaging-api-7.9.yaml"), []byte("spec"), 0644)

	got := findIntegrationSpecFile(dir, "messaging-api-7.9.yaml")
	if got == "" {
		t.Error("expected to find spec file")
	}
}

func TestFindIntegrationSpecFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	got := findIntegrationSpecFile(dir, "nonexistent.yaml")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFindIntegrationSpecFile_InContract(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "main", "resources", "contract")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "api.yaml"), []byte("spec"), 0644)

	got := findIntegrationSpecFile(dir, "api.yaml")
	if got == "" {
		t.Error("expected to find spec file in contract dir")
	}
}

func TestFindIntegrationSpecFile_InRestDir(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "main", "resources", "integrations", "rest")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "api.yaml"), []byte("spec"), 0644)

	got := findIntegrationSpecFile(dir, "api.yaml")
	if got == "" {
		t.Error("expected to find spec file in rest dir")
	}
}
