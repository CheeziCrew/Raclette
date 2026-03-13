package ops

import (
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ─────────────────────────────────────────────────────────

// mkFile creates a file with content in a temp directory tree.
func mkFile(t *testing.T, base, relPath, content string) {
	t.Helper()
	full := filepath.Join(base, relPath)
	os.MkdirAll(filepath.Dir(full), 0755)
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("mkFile: %v", err)
	}
}

// mkJavaRepo creates a minimal Maven repo with a Java integration source file.
func mkJavaRepo(t *testing.T, base, repoName, javaRelPath, javaContent string) string {
	t.Helper()
	repo := filepath.Join(base, repoName)
	os.MkdirAll(repo, 0755)
	mkFile(t, repo, javaRelPath, javaContent)
	return repo
}

// ── TestScanRepo ────────────────────────────────────────────────────

func TestScanRepo(t *testing.T) {
	t.Run("finds CLIENT_ID", func(t *testing.T) {
		base := t.TempDir()
		javaContent := `package com.example.integration.casedata;
public class CaseDataClient {
    private static final String CLIENT_ID = "case-data";
}`
		repo := mkJavaRepo(t, base, "api-service-foo",
			filepath.Join("src", "main", "java", "com", "example", "integration", "casedata", "CaseDataClient.java"),
			javaContent)

		integrations := scanRepo(repo)
		if len(integrations) == 0 {
			t.Fatal("expected at least one integration")
		}
		found := false
		for _, integ := range integrations {
			if integ.clientID == "case-data" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find clientID=case-data, got %v", integrations)
		}
	})

	t.Run("finds INTEGRATION_NAME", func(t *testing.T) {
		base := t.TempDir()
		javaContent := `package com.example.integration.messaging;
public class MessagingIntegration {
    private static final String INTEGRATION_NAME = "messaging";
}`
		repo := mkJavaRepo(t, base, "api-service-bar",
			filepath.Join("src", "main", "java", "com", "example", "integration", "messaging", "MessagingIntegration.java"),
			javaContent)

		integrations := scanRepo(repo)
		if len(integrations) == 0 {
			t.Fatal("expected at least one integration")
		}
		found := false
		for _, integ := range integrations {
			if integ.clientID == "messaging" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find clientID=messaging, got %v", integrations)
		}
	})

	t.Run("skips db packages", func(t *testing.T) {
		base := t.TempDir()
		javaContent := `package com.example.integration.db;
public class DbClient {
    private static final String CLIENT_ID = "should-not-find";
}`
		repo := mkJavaRepo(t, base, "api-service-db",
			filepath.Join("src", "main", "java", "com", "example", "integration", "db", "DbClient.java"),
			javaContent)

		integrations := scanRepo(repo)
		for _, integ := range integrations {
			if integ.clientID == "should-not-find" {
				t.Error("should have skipped db package")
			}
		}
	})

	t.Run("skips non-java files", func(t *testing.T) {
		base := t.TempDir()
		repo := mkJavaRepo(t, base, "api-service-txt",
			filepath.Join("src", "main", "java", "com", "example", "integration", "foo", "readme.txt"),
			`CLIENT_ID = "should-not-find"`)

		integrations := scanRepo(repo)
		for _, integ := range integrations {
			if integ.clientID == "should-not-find" {
				t.Error("should have skipped non-java file")
			}
		}
	})

	t.Run("adds unresolved packages", func(t *testing.T) {
		base := t.TempDir()
		// Java file in integration subpackage but without CLIENT_ID or INTEGRATION_NAME
		javaContent := `package com.example.integration.orphanpkg;
public class SomeHelper {
    // no CLIENT_ID or INTEGRATION_NAME
}`
		repo := mkJavaRepo(t, base, "api-service-orphan",
			filepath.Join("src", "main", "java", "com", "example", "integration", "orphanpkg", "SomeHelper.java"),
			javaContent)

		integrations := scanRepo(repo)
		found := false
		for _, integ := range integrations {
			if integ.clientID == "orphanpkg" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected unresolved package 'orphanpkg' to be added, got %v", integrations)
		}
	})

	t.Run("empty repo returns nil", func(t *testing.T) {
		base := t.TempDir()
		repo := filepath.Join(base, "empty-repo")
		os.MkdirAll(repo, 0755)

		integrations := scanRepo(repo)
		if len(integrations) != 0 {
			t.Errorf("expected empty integrations, got %d", len(integrations))
		}
	})

	t.Run("deduplicates CLIENT_ID across files", func(t *testing.T) {
		base := t.TempDir()
		javaA := `package com.example.integration.svc;
public class ClientA {
    private static final String CLIENT_ID = "my-svc";
}`
		javaB := `package com.example.integration.svc;
public class ClientB {
    private static final String CLIENT_ID = "my-svc";
}`
		repo := filepath.Join(base, "api-service-dedup")
		os.MkdirAll(repo, 0755)
		mkFile(t, repo, filepath.Join("src", "main", "java", "com", "example", "integration", "svc", "ClientA.java"), javaA)
		mkFile(t, repo, filepath.Join("src", "main", "java", "com", "example", "integration", "svc", "ClientB.java"), javaB)

		integrations := scanRepo(repo)
		count := 0
		for _, integ := range integrations {
			if integ.clientID == "my-svc" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 'my-svc' integration, got %d", count)
		}
	})

	t.Run("integrations are sorted by clientID", func(t *testing.T) {
		base := t.TempDir()
		repo := filepath.Join(base, "api-service-sorted")
		os.MkdirAll(repo, 0755)
		mkFile(t, repo, filepath.Join("src", "main", "java", "com", "example", "integration", "zebra", "Z.java"),
			`package x;
public class Z { private static final String CLIENT_ID = "zebra"; }`)
		mkFile(t, repo, filepath.Join("src", "main", "java", "com", "example", "integration", "alpha", "A.java"),
			`package x;
public class A { private static final String CLIENT_ID = "alpha"; }`)

		integrations := scanRepo(repo)
		if len(integrations) < 2 {
			t.Fatalf("expected at least 2 integrations, got %d", len(integrations))
		}
		if integrations[0].clientID >= integrations[1].clientID {
			t.Errorf("expected sorted order, got %q before %q", integrations[0].clientID, integrations[1].clientID)
		}
	})
}

// ── TestAddIntegration ──────────────────────────────────────────────

func TestAddIntegration(t *testing.T) {
	t.Run("adds new entry", func(t *testing.T) {
		var integrations []integration
		seen := map[string]bool{}
		addIntegration(&integrations, seen, "my-svc")
		if len(integrations) != 1 || integrations[0].clientID != "my-svc" {
			t.Errorf("got %v, want [{clientID:my-svc}]", integrations)
		}
	})

	t.Run("skips duplicate", func(t *testing.T) {
		var integrations []integration
		seen := map[string]bool{}
		addIntegration(&integrations, seen, "my-svc")
		addIntegration(&integrations, seen, "my-svc")
		if len(integrations) != 1 {
			t.Errorf("expected 1 integration after duplicate, got %d", len(integrations))
		}
	})
}

// ── TestMatchSpecVersions ───────────────────────────────────────────

func TestMatchSpecVersions(t *testing.T) {
	t.Run("populates spec info", func(t *testing.T) {
		base := t.TempDir()

		// Create pom.xml with inputSpec reference
		pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>2.0.0</version>
  <build>
    <plugins>
      <plugin>
        <executions>
          <execution>
            <configuration>
              <inputSpec>${project.basedir}/src/main/resources/integrations/case-data-api-2.0.yaml</inputSpec>
            </configuration>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</project>`
		mkFile(t, base, "pom.xml", pomContent)

		// Create the spec file
		specContent := `openapi: 3.0.1
info:
  title: Case Data API
  version: 2.0.0
paths: {}`
		mkFile(t, base, filepath.Join("src", "main", "resources", "integrations", "case-data-api-2.0.yaml"), specContent)

		integrations := []integration{
			{clientID: "case-data"},
		}

		matchSpecVersions(base, integrations)

		if integrations[0].specVersion != "2.0.0" {
			t.Errorf("specVersion = %q, want 2.0.0", integrations[0].specVersion)
		}
		if integrations[0].specFile != "case-data-api-2.0.yaml" {
			t.Errorf("specFile = %q, want case-data-api-2.0.yaml", integrations[0].specFile)
		}
	})

	t.Run("no specs found", func(t *testing.T) {
		base := t.TempDir()
		// No pom.xml at all
		integrations := []integration{
			{clientID: "foo"},
		}
		matchSpecVersions(base, integrations)
		if integrations[0].specVersion != "" {
			t.Errorf("expected empty specVersion, got %q", integrations[0].specVersion)
		}
	})
}

// ── TestFindIntegrationSpecFile ─────────────────────────────────────

func TestFindIntegrationSpecFile(t *testing.T) {
	t.Run("found in integrations dir", func(t *testing.T) {
		base := t.TempDir()
		mkFile(t, base, filepath.Join("src", "main", "resources", "integrations", "my-spec.yaml"), "spec")

		got := findIntegrationSpecFile(base, "my-spec.yaml")
		if got == "" {
			t.Fatal("expected to find spec file")
		}
	})

	t.Run("found in integrations/rest dir", func(t *testing.T) {
		base := t.TempDir()
		mkFile(t, base, filepath.Join("src", "main", "resources", "integrations", "rest", "my-spec.yaml"), "spec")

		got := findIntegrationSpecFile(base, "my-spec.yaml")
		if got == "" {
			t.Fatal("expected to find spec file in rest subdir")
		}
	})

	t.Run("found in contract dir", func(t *testing.T) {
		base := t.TempDir()
		mkFile(t, base, filepath.Join("src", "main", "resources", "contract", "my-spec.yaml"), "spec")

		got := findIntegrationSpecFile(base, "my-spec.yaml")
		if got == "" {
			t.Fatal("expected to find spec file in contract dir")
		}
	})

	t.Run("not found", func(t *testing.T) {
		base := t.TempDir()
		got := findIntegrationSpecFile(base, "nonexistent.yaml")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

// ── TestBuildServiceIndex ───────────────────────────────────────────

func TestBuildServiceIndex(t *testing.T) {
	base := t.TempDir()

	// Create two repos with pom.xml
	repo1 := filepath.Join(base, "api-service-foo")
	os.MkdirAll(repo1, 0755)
	mkFile(t, repo1, "pom.xml", `<?xml version="1.0"?>
<project>
  <parent><groupId>se.sundsvall.dept44</groupId><artifactId>dept44-service-parent</artifactId><version>5.0.0</version></parent>
  <version>1.0.0</version>
</project>`)

	repo2 := filepath.Join(base, "api-service-bar")
	os.MkdirAll(repo2, 0755)
	mkFile(t, repo2, "pom.xml", `<?xml version="1.0"?>
<project>
  <parent><groupId>se.sundsvall.dept44</groupId><artifactId>dept44-service-parent</artifactId><version>5.0.0</version></parent>
  <version>2.0.0</version>
</project>`)

	versions, idx := buildServiceIndex([]string{repo1, repo2})

	if versions["foo"] != "1.0.0" {
		t.Errorf("versions[foo] = %q, want 1.0.0", versions["foo"])
	}
	if versions["bar"] != "2.0.0" {
		t.Errorf("versions[bar] = %q, want 2.0.0", versions["bar"])
	}
	if idx.Resolve("foo") == "" {
		t.Error("expected idx to resolve 'foo'")
	}
}

// ── TestNormalizeClientIDs ──────────────────────────────────────────

func TestNormalizeClientIDs(t *testing.T) {
	idx := NewNameIndex([]string{"case-data", "messaging"})

	t.Run("strips client suffix", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-dataclient"},
		}
		normalizeClientIDs(integrations, idx)
		if integrations[0].clientID != "case-data" {
			t.Errorf("got %q, want case-data", integrations[0].clientID)
		}
	})

	t.Run("strips integration suffix", func(t *testing.T) {
		integrations := []integration{
			{clientID: "messagingintegration"},
		}
		normalizeClientIDs(integrations, idx)
		if integrations[0].clientID != "messaging" {
			t.Errorf("got %q, want messaging", integrations[0].clientID)
		}
	})

	t.Run("strips service suffix", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-dataservice"},
		}
		normalizeClientIDs(integrations, idx)
		if integrations[0].clientID != "case-data" {
			t.Errorf("got %q, want case-data", integrations[0].clientID)
		}
	})

	t.Run("already resolved leaves unchanged", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-data"},
		}
		normalizeClientIDs(integrations, idx)
		if integrations[0].clientID != "case-data" {
			t.Errorf("got %q, want case-data", integrations[0].clientID)
		}
	})

	t.Run("unresolvable stays unchanged", func(t *testing.T) {
		integrations := []integration{
			{clientID: "totally-unknown"},
		}
		normalizeClientIDs(integrations, idx)
		if integrations[0].clientID != "totally-unknown" {
			t.Errorf("got %q, want totally-unknown", integrations[0].clientID)
		}
	})
}

// ── TestFindStaleInRepo ─────────────────────────────────────────────

func TestFindStaleInRepo(t *testing.T) {
	idx := NewNameIndex([]string{"case-data", "messaging"})
	svcVersions := map[string]string{
		"case-data": "3.0.0",
		"messaging": "2.0.0",
	}

	t.Run("finds stale integration", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-data", specVersion: "1.0.0", specFile: "case-data-api-1.0.yaml", specPath: "/path/spec.yaml"},
		}
		stale := findStaleInRepo("/tmp/repo", "repo", integrations, svcVersions, idx)
		if len(stale) != 1 {
			t.Fatalf("expected 1 stale, got %d", len(stale))
		}
		if stale[0].SpecVersion != "1.0.0" || stale[0].TargetVersion != "3.0.0" {
			t.Errorf("got spec=%q target=%q", stale[0].SpecVersion, stale[0].TargetVersion)
		}
	})

	t.Run("skips current integration", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-data", specVersion: "3.0.0", specFile: "f.yaml", specPath: "/p"},
		}
		stale := findStaleInRepo("/tmp/repo", "repo", integrations, svcVersions, idx)
		if len(stale) != 0 {
			t.Errorf("expected 0 stale, got %d", len(stale))
		}
	})

	t.Run("skips no spec version", func(t *testing.T) {
		integrations := []integration{
			{clientID: "case-data"},
		}
		stale := findStaleInRepo("/tmp/repo", "repo", integrations, svcVersions, idx)
		if len(stale) != 0 {
			t.Errorf("expected 0 stale, got %d", len(stale))
		}
	})

	t.Run("skips unknown target", func(t *testing.T) {
		integrations := []integration{
			{clientID: "unknown-svc", specVersion: "1.0.0", specFile: "f.yaml", specPath: "/p"},
		}
		stale := findStaleInRepo("/tmp/repo", "repo", integrations, svcVersions, idx)
		if len(stale) != 0 {
			t.Errorf("expected 0 stale, got %d", len(stale))
		}
	})
}

// ── TestScanStaleIntegrations ───────────────────────────────────────

func TestScanStaleIntegrations(t *testing.T) {
	base := t.TempDir()

	// Target service repo: case-data at version 3.0.0
	target := filepath.Join(base, "api-service-case-data")
	os.MkdirAll(target, 0755)
	mkFile(t, target, "pom.xml", `<?xml version="1.0"?>
<project>
  <parent><groupId>se.sundsvall.dept44</groupId><artifactId>dept44-service-parent</artifactId><version>5.0.0</version></parent>
  <version>3.0.0</version>
</project>`)

	// Consumer repo: has an integration with old spec version
	consumer := filepath.Join(base, "api-service-consumer")
	os.MkdirAll(consumer, 0755)
	mkFile(t, consumer, "pom.xml", `<?xml version="1.0"?>
<project>
  <parent><groupId>se.sundsvall.dept44</groupId><artifactId>dept44-service-parent</artifactId><version>5.0.0</version></parent>
  <version>1.0.0</version>
  <build>
    <plugins>
      <plugin>
        <executions>
          <execution>
            <configuration>
              <inputSpec>${project.basedir}/src/main/resources/integrations/case-data-api-1.0.yaml</inputSpec>
            </configuration>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</project>`)

	// Java source with CLIENT_ID
	mkFile(t, consumer,
		filepath.Join("src", "main", "java", "com", "example", "integration", "casedata", "CaseDataClient.java"),
		`package com.example.integration.casedata;
public class CaseDataClient {
    private static final String CLIENT_ID = "case-data";
}`)

	// Spec file with old version
	mkFile(t, consumer,
		filepath.Join("src", "main", "resources", "integrations", "case-data-api-1.0.yaml"),
		`openapi: 3.0.1
info:
  title: Case Data API
  version: 1.0.0
paths: {}`)

	stale := ScanStaleIntegrations([]string{target, consumer})
	if len(stale) == 0 {
		t.Fatal("expected at least one stale integration")
	}

	found := false
	for _, s := range stale {
		if s.TargetName == "case-data" && s.SpecVersion == "1.0.0" && s.TargetVersion == "3.0.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected stale integration for case-data 1.0.0 -> 3.0.0, got %v", stale)
	}
}

// ── TestFindStaleSpecs ──────────────────────────────────────────────

func TestFindStaleSpecs(t *testing.T) {
	t.Run("empty dirs", func(t *testing.T) {
		result := FindStaleSpecs(nil)
		if len(result) != 0 {
			t.Errorf("expected 0, got %d", len(result))
		}
	})

	t.Run("no stale", func(t *testing.T) {
		base := t.TempDir()
		repo := filepath.Join(base, "api-service-clean")
		os.MkdirAll(repo, 0755)
		mkFile(t, repo, "pom.xml", `<?xml version="1.0"?>
<project>
  <parent><groupId>se.sundsvall.dept44</groupId><artifactId>dept44-service-parent</artifactId><version>5.0.0</version></parent>
  <version>1.0.0</version>
</project>`)

		result := FindStaleSpecs([]string{repo})
		if len(result) != 0 {
			t.Errorf("expected no stale specs, got %d", len(result))
		}
	})
}

// ── TestScanRepoForStale ────────────────────────────────────────────

func TestScanRepoForStale(t *testing.T) {
	t.Run("no integrations returns nil", func(t *testing.T) {
		base := t.TempDir()
		repo := filepath.Join(base, "empty-repo")
		os.MkdirAll(repo, 0755)

		idx := NewNameIndex([]string{"foo"})
		svcVersions := map[string]string{"foo": "1.0.0"}
		stale := scanRepoForStale(repo, svcVersions, idx)
		if stale != nil {
			t.Errorf("expected nil, got %v", stale)
		}
	})
}
