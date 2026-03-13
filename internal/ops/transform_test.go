package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── TestReplaceDependencyVersion ────────────────────────────────────

func TestReplaceDependencyVersion(t *testing.T) {
	t.Run("matching dep", func(t *testing.T) {
		dir := t.TempDir()
		pom := `<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework</groupId>
      <artifactId>spring-core</artifactId>
      <version>5.0.0</version>
    </dependency>
  </dependencies>
</project>`
		os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

		results := ReplaceDependencyVersion([]string{dir}, "org.springframework", "spring-core", "6.0.0")
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if !results[0].Changed {
			t.Error("expected Changed=true")
		}

		data, _ := os.ReadFile(filepath.Join(dir, "pom.xml"))
		if !strings.Contains(string(data), "<version>6.0.0</version>") {
			t.Error("pom.xml not updated")
		}
	})

	t.Run("non-matching dep", func(t *testing.T) {
		dir := t.TempDir()
		pom := `<project>
  <dependencies>
    <dependency>
      <groupId>com.other</groupId>
      <artifactId>other-lib</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
</project>`
		os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

		results := ReplaceDependencyVersion([]string{dir}, "org.springframework", "spring-core", "6.0.0")
		if len(results) != 1 {
			t.Fatalf("got %d results, want 1", len(results))
		}
		if results[0].Changed {
			t.Error("expected Changed=false for non-matching dep")
		}
	})
}

// ── TestBumpParentVersion ───────────────────────────────────────────

func TestBumpParentVersion(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>4.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	results := BumpParentVersion([]string{dir}, "5.0.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Changed {
		t.Error("expected Changed=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "pom.xml"))
	if !strings.Contains(string(data), "<version>5.0.0</version>") {
		t.Error("parent version not updated")
	}
}

// ── TestSwapDependency ──────────────────────────────────────────────

func TestSwapDependency(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>com.old</groupId>
      <artifactId>old-lib</artifactId>
      <version>1.0.0</version>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	results := SwapDependency([]string{dir}, "com.old", "old-lib", "com.new", "new-lib", "2.0.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Changed {
		t.Error("expected Changed=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "pom.xml"))
	content := string(data)

	if !strings.Contains(content, "<groupId>com.new</groupId>") {
		t.Error("new groupId not found")
	}
	if !strings.Contains(content, "<artifactId>new-lib</artifactId>") {
		t.Error("new artifactId not found")
	}
	if !strings.Contains(content, "<version>2.0.0</version>") {
		t.Error("new version not found")
	}
	if !strings.Contains(content, "<scope>test</scope>") {
		t.Error("scope not preserved")
	}
}

// ── TestFormatResults ───────────────────────────────────────────────

func TestFormatResults(t *testing.T) {
	results := []Result{
		{Repo: "repo-a", Changed: true, Message: "updated to 2.0.0"},
		{Repo: "repo-b", Err: fmt.Errorf("read error")},
		{Repo: "repo-c", Changed: false, Message: "no match"},
	}

	lines := FormatResults(results)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}

	// Header should say 1 changed
	if !strings.Contains(lines[0], "1 repo(s)") {
		t.Errorf("header = %q, expected to mention 1 repo(s)", lines[0])
	}

	// Check we have success, error, and unchanged markers
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "repo-a") {
		t.Error("missing repo-a in output")
	}
	if !strings.Contains(joined, "repo-b") {
		t.Error("missing repo-b in output")
	}
	if !strings.Contains(joined, "repo-c") {
		t.Error("missing repo-c in output")
	}
}

// ── TestReplaceInPom_DollarInVersion ────────────────────────────────

func TestReplaceInPom_DollarInVersion(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>4.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	// Use a version containing $ to verify escaping
	results := BumpParentVersion([]string{dir}, "$pecial-1.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Changed {
		t.Error("expected Changed=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "pom.xml"))
	if !strings.Contains(string(data), "$pecial-1.0") {
		t.Errorf("dollar sign not preserved in version: %s", string(data))
	}
}

// ── TestSwapDependency edge cases ───────────────────────────────────

func TestSwapDependency_MissingPom(t *testing.T) {
	dir := t.TempDir()
	results := SwapDependency([]string{dir}, "com.old", "old-lib", "com.new", "new-lib", "2.0.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected error for missing pom.xml")
	}
}

func TestSwapDependency_NotFound(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>com.other</groupId>
      <artifactId>other-lib</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	results := SwapDependency([]string{dir}, "com.old", "old-lib", "com.new", "new-lib", "2.0.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Changed {
		t.Error("expected Changed=false for non-matching dep")
	}
	if results[0].Message != "dependency not found" {
		t.Errorf("message = %q, want 'dependency not found'", results[0].Message)
	}
}

func TestSwapDependency_WithExclusions(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>com.old</groupId>
      <artifactId>old-lib</artifactId>
      <version>1.0.0</version>
      <scope>compile</scope>
      <exclusions>
        <exclusion>
          <groupId>com.unwanted</groupId>
          <artifactId>unwanted-lib</artifactId>
        </exclusion>
      </exclusions>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	results := SwapDependency([]string{dir}, "com.old", "old-lib", "com.new", "new-lib", "2.0.0")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Changed {
		t.Error("expected Changed=true")
	}

	data, _ := os.ReadFile(filepath.Join(dir, "pom.xml"))
	content := string(data)
	if !strings.Contains(content, "<scope>compile</scope>") {
		t.Error("scope not preserved")
	}
	if !strings.Contains(content, "<exclusions>") {
		t.Error("exclusions not preserved")
	}
}

func TestSwapDependency_WriteError(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>com.old</groupId>
      <artifactId>old-lib</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
</project>`
	pomPath := filepath.Join(dir, "pom.xml")
	os.WriteFile(pomPath, []byte(pom), 0644)
	// Make pom.xml read-only to trigger write error
	os.Chmod(pomPath, 0444)
	t.Cleanup(func() { os.Chmod(pomPath, 0644) })

	results := SwapDependency([]string{dir}, "com.old", "old-lib", "com.new", "new-lib", "2.0.0")
	if results[0].Err == nil {
		t.Error("expected error when pom.xml is read-only")
	}
}

// ── TestReplaceInPom edge cases ─────────────────────────────────────

func TestReplaceInPom_MissingPom(t *testing.T) {
	dir := t.TempDir()
	results := ReplaceDependencyVersion([]string{dir}, "g", "a", "1.0")
	if results[0].Err == nil {
		t.Error("expected error for missing pom.xml")
	}
}

func TestReplaceInPom_AlreadyAtVersion(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>org.example</groupId>
      <artifactId>my-lib</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	results := ReplaceDependencyVersion([]string{dir}, "org.example", "my-lib", "1.0.0")
	if results[0].Changed {
		t.Error("expected Changed=false when already at target version")
	}
	if results[0].Message != "already at target version" {
		t.Errorf("message = %q, want 'already at target version'", results[0].Message)
	}
}

func TestReplaceInPom_WriteError(t *testing.T) {
	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>org.example</groupId>
      <artifactId>my-lib</artifactId>
      <version>1.0.0</version>
    </dependency>
  </dependencies>
</project>`
	pomPath := filepath.Join(dir, "pom.xml")
	os.WriteFile(pomPath, []byte(pom), 0644)
	os.Chmod(pomPath, 0444)
	t.Cleanup(func() { os.Chmod(pomPath, 0644) })

	results := ReplaceDependencyVersion([]string{dir}, "org.example", "my-lib", "2.0.0")
	if results[0].Err == nil {
		t.Error("expected error when pom.xml is read-only")
	}
}

// ── TestUpdateIntegrationSpecs ──────────────────────────────────────

func TestUpdateIntegrationSpecs_NoStaleSpecs(t *testing.T) {
	// Two repos with no integration code — should get "all up to date" message
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create valid dept44 pom.xml files with version
	for _, dir := range []string{dir1, dir2} {
		pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>1.0.0</version>
</project>`
		os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)
	}

	results := UpdateIntegrationSpecs([]string{dir1, dir2})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !strings.Contains(results[0].Message, "up to date") {
		t.Errorf("message = %q, want 'up to date'", results[0].Message)
	}
}

func TestBuildSvcIndex(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// dir1 has valid pom with version and own spec
	pom1 := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>2.0.0</version>
</project>`
	os.WriteFile(filepath.Join(dir1, "pom.xml"), []byte(pom1), 0644)

	// Create own spec in dir1
	specDir := filepath.Join(dir1, "src", "integration-test", "resources")
	os.MkdirAll(specDir, 0755)
	specContent := "openapi: 3.0.1\ninfo:\n  title: Test\n  version: 2.0.0\n"
	os.WriteFile(filepath.Join(specDir, "openapi.yaml"), []byte(specContent), 0644)

	// dir2 has no pom.xml — should be skipped
	idx := buildSvcIndex([]string{dir1, dir2})

	name1 := serviceNameFromDir(dir1)
	if _, ok := idx[name1]; !ok {
		t.Errorf("expected svcIndex to contain %q", name1)
	}
	if idx[name1].pomVersion != "2.0.0" {
		t.Errorf("pomVersion = %q, want 2.0.0", idx[name1].pomVersion)
	}
	if idx[name1].specPath == "" {
		t.Error("expected specPath to be set")
	}
}

func TestBuildSvcIndex_NoVersion(t *testing.T) {
	dir := t.TempDir()
	// pom with no version — should be skipped
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	idx := buildSvcIndex([]string{dir})
	if len(idx) != 0 {
		t.Errorf("expected empty index for repo with no version, got %d entries", len(idx))
	}
}

func TestBuildSvcIndex_NoSpec(t *testing.T) {
	dir := t.TempDir()
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>3.0.0</version>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	idx := buildSvcIndex([]string{dir})
	name := serviceNameFromDir(dir)
	if idx[name].specPath != "" {
		t.Error("expected empty specPath when no spec exists")
	}
	if idx[name].pomVersion != "3.0.0" {
		t.Errorf("pomVersion = %q, want 3.0.0", idx[name].pomVersion)
	}
}

// ── TestUpdateOneSpec ───────────────────────────────────────────────

func TestUpdateOneSpec_NoSpecPath(t *testing.T) {
	s := StaleIntegration{
		SpecFile:   "test.yaml",
		TargetName: "svc",
	}
	msg, status := updateOneSpec(s, map[string]svcEntry{})
	if status != specSkipped {
		t.Errorf("status = %d, want specSkipped", status)
	}
	if !strings.Contains(msg, "no spec file path") {
		t.Errorf("msg = %q, want 'no spec file path'", msg)
	}
}

func TestUpdateOneSpec_TargetNotInIndex(t *testing.T) {
	s := StaleIntegration{
		SpecFile:   "test.yaml",
		SpecPath:   "/some/path",
		TargetName: "unknown-svc",
	}
	msg, status := updateOneSpec(s, map[string]svcEntry{})
	if status != specIgnored {
		t.Errorf("status = %d, want specIgnored", status)
	}
	if msg != "" {
		t.Errorf("msg = %q, want empty", msg)
	}
}

func TestUpdateOneSpec_NoSourceSpec(t *testing.T) {
	s := StaleIntegration{
		SpecFile:   "test.yaml",
		SpecPath:   "/some/path",
		TargetName: "my-svc",
	}
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: ""},
	}
	msg, status := updateOneSpec(s, idx)
	if status != specSkipped {
		t.Errorf("status = %d, want specSkipped", status)
	}
	if !strings.Contains(msg, "no source spec found") {
		t.Errorf("msg = %q, want 'no source spec found'", msg)
	}
}

func TestUpdateOneSpec_SourceSpecStale(t *testing.T) {
	// Create a source spec with version 1.0.0 but pom says 2.0.0
	srcDir := t.TempDir()
	srcSpec := filepath.Join(srcDir, "openapi.yaml")
	os.WriteFile(srcSpec, []byte("openapi: 3.0.1\ninfo:\n  title: API\n  version: 1.0.0\n"), 0644)

	s := StaleIntegration{
		SpecFile:    "test.yaml",
		SpecPath:    filepath.Join(t.TempDir(), "target.yaml"),
		TargetName:  "my-svc",
		SpecVersion: "0.9.0",
	}
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: srcSpec},
	}
	msg, status := updateOneSpec(s, idx)
	if status != specSkipped {
		t.Errorf("status = %d, want specSkipped", status)
	}
	if !strings.Contains(msg, "source") && !strings.Contains(msg, "stale") {
		t.Errorf("msg = %q, want stale source message", msg)
	}
}

func TestUpdateOneSpec_Success(t *testing.T) {
	// Create a source spec with matching version
	srcDir := t.TempDir()
	srcSpec := filepath.Join(srcDir, "openapi.yaml")
	specContent := "openapi: 3.0.1\ninfo:\n  title: API\n  version: 2.0.0\n"
	os.WriteFile(srcSpec, []byte(specContent), 0644)

	// Create target spec with old version
	dstDir := t.TempDir()
	dstSpec := filepath.Join(dstDir, "target.yaml")
	os.WriteFile(dstSpec, []byte("old content"), 0644)

	s := StaleIntegration{
		SpecFile:    "test.yaml",
		SpecPath:    dstSpec,
		TargetName:  "my-svc",
		SpecVersion: "1.0.0",
	}
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: srcSpec},
	}
	msg, status := updateOneSpec(s, idx)
	if status != specUpdated {
		t.Errorf("status = %d, want specUpdated", status)
	}
	if !strings.Contains(msg, "1.0.0") || !strings.Contains(msg, "2.0.0") {
		t.Errorf("msg = %q, want old and new version", msg)
	}

	// Verify the file was actually copied
	data, _ := os.ReadFile(dstSpec)
	if string(data) != specContent {
		t.Error("target spec was not updated with source content")
	}
}

func TestUpdateOneSpec_ReadSourceError(t *testing.T) {
	s := StaleIntegration{
		SpecFile:    "test.yaml",
		SpecPath:    filepath.Join(t.TempDir(), "target.yaml"),
		TargetName:  "my-svc",
		SpecVersion: "1.0.0",
	}
	// specPath points to a file that will pass version check but fail read
	// We need a valid spec for extractSpecVersion to return matching version
	// but then remove it before os.ReadFile — not easy to test without mocking.
	// Instead test with a nonexistent source spec path.
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: "/nonexistent/openapi.yaml"},
	}
	msg, status := updateOneSpec(s, idx)
	// extractSpecVersion will return "" which != "2.0.0", so this becomes specSkipped
	if status != specSkipped {
		t.Errorf("status = %d, want specSkipped (source version mismatch)", status)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestUpdateOneSpec_WriteError(t *testing.T) {
	// Create a source spec
	srcDir := t.TempDir()
	srcSpec := filepath.Join(srcDir, "openapi.yaml")
	os.WriteFile(srcSpec, []byte("openapi: 3.0.1\ninfo:\n  title: API\n  version: 2.0.0\n"), 0644)

	// Create target path in a read-only directory
	dstDir := t.TempDir()
	dstSpec := filepath.Join(dstDir, "subdir", "target.yaml")
	// Don't create the subdir — write will fail

	s := StaleIntegration{
		SpecFile:    "test.yaml",
		SpecPath:    dstSpec,
		TargetName:  "my-svc",
		SpecVersion: "1.0.0",
	}
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: srcSpec},
	}
	msg, status := updateOneSpec(s, idx)
	if status != specIgnored {
		t.Errorf("status = %d, want specIgnored (write error)", status)
	}
	if !strings.Contains(msg, "failed to write") {
		t.Errorf("msg = %q, want 'failed to write'", msg)
	}
}

// ── TestUpdateRepoSpecs ─────────────────────────────────────────────

func TestUpdateRepoSpecs_NoItems(t *testing.T) {
	_, ok := updateRepoSpecs("repo", nil, map[string]svcEntry{})
	if ok {
		t.Error("expected ok=false with no items")
	}
}

func TestUpdateRepoSpecs_WithUpdates(t *testing.T) {
	// Create source spec
	srcDir := t.TempDir()
	srcSpec := filepath.Join(srcDir, "openapi.yaml")
	os.WriteFile(srcSpec, []byte("openapi: 3.0.1\ninfo:\n  title: API\n  version: 2.0.0\n"), 0644)

	// Create target spec
	dstDir := t.TempDir()
	dstSpec := filepath.Join(dstDir, "target.yaml")
	os.WriteFile(dstSpec, []byte("old"), 0644)

	items := []StaleIntegration{
		{
			SpecFile:    "svc.yaml",
			SpecPath:    dstSpec,
			TargetName:  "my-svc",
			SpecVersion: "1.0.0",
		},
	}
	idx := map[string]svcEntry{
		"my-svc": {pomVersion: "2.0.0", specPath: srcSpec},
	}
	result, ok := updateRepoSpecs("repo", items, idx)
	if !ok {
		t.Error("expected ok=true")
	}
	if !result.Changed {
		t.Error("expected Changed=true")
	}
	if !strings.Contains(result.Message, "Updated 1") {
		t.Errorf("message = %q, want 'Updated 1'", result.Message)
	}
}

func TestUpdateRepoSpecs_AllSkipped(t *testing.T) {
	items := []StaleIntegration{
		{
			SpecFile:   "svc.yaml",
			TargetName: "my-svc",
			// No SpecPath — will be skipped
		},
	}
	result, ok := updateRepoSpecs("repo", items, map[string]svcEntry{})
	if !ok {
		t.Error("expected ok=true when items are skipped")
	}
	if result.Changed {
		t.Error("expected Changed=false when all skipped")
	}
}

// ── TestFormatResults with no repo ──────────────────────────────────

func TestFormatResults_NoRepo(t *testing.T) {
	results := []Result{
		{Message: "All integration specs are already up to date."},
	}
	lines := FormatResults(results)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "up to date") {
		t.Errorf("expected 'up to date' in output, got %q", joined)
	}
}

// ── TestFindOwnSpec ─────────────────────────────────────────────────

func TestFindOwnSpec_IntegrationTestResources(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "integration-test", "resources")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "openapi.yaml"), []byte("openapi: 3.0.1\ninfo:\n  title: T\n  version: 1.0\n"), 0644)

	got := findOwnSpec(dir)
	if got == "" {
		t.Error("expected to find spec in integration-test/resources")
	}
}

func TestFindOwnSpec_MainResources(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "main", "resources")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "openapi.yml"), []byte("openapi: 3.0.1\ninfo:\n  title: T\n  version: 1.0\n"), 0644)

	got := findOwnSpec(dir)
	if got == "" {
		t.Error("expected to find spec in main/resources")
	}
}

func TestFindOwnSpec_TestResources(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "src", "test", "resources")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "api-docs.yaml"), []byte("openapi: 3.0.1\ninfo:\n  title: T\n  version: 1.0\n"), 0644)

	got := findOwnSpec(dir)
	if got == "" {
		t.Error("expected to find spec in test/resources")
	}
}

func TestFindOwnSpec_NotFound(t *testing.T) {
	dir := t.TempDir()
	got := findOwnSpec(dir)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFindSpecIn_NoVersionInFile(t *testing.T) {
	dir := t.TempDir()
	// File matches name pattern but has no version
	os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("some: yaml\n"), 0644)

	got := findSpecIn(dir)
	if got != "" {
		t.Errorf("expected empty string for spec without version, got %q", got)
	}
}
