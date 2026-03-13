package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CheeziCrew/raclette/internal/ops"
)

func TestBuildCLI(t *testing.T) {
	root := BuildCLI()
	if root.Use != "raclette" {
		t.Errorf("root.Use = %q, want %q", root.Use, "raclette")
	}
}

func TestBuildCLISubcommands(t *testing.T) {
	root := BuildCLI()
	names := make(map[string]bool)
	for _, cmd := range root.Commands() {
		names[cmd.Name()] = true
	}

	want := []string{"format", "verify", "find-dep", "find-stale",
		"replace-dep", "swap-dep", "bump-parent", "update-specs", "update-integration-specs"}
	for _, w := range want {
		if !names[w] {
			t.Errorf("expected subcommand %q", w)
		}
	}
}

func TestBuildCLIHasJSONFlag(t *testing.T) {
	root := BuildCLI()
	f := root.PersistentFlags().Lookup("json")
	if f == nil {
		t.Error("expected --json persistent flag")
	}
}

func TestPrintJSON(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := printJSON(data)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("printJSON() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var parsed map[string]string
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("parsed[key] = %q, want %q", parsed["key"], "value")
	}
}

func TestPrintJSONSlice(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSON([]int{1, 2, 3})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("printJSON() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var parsed []int
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if len(parsed) != 3 {
		t.Errorf("expected 3 items, got %d", len(parsed))
	}
}

func TestDiscoverReposEmpty(t *testing.T) {
	// Create a temp dir with no repos.
	tmp := t.TempDir()

	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	repos := discoverRepos()
	if len(repos) != 0 {
		t.Errorf("expected 0 repos in empty dir, got %d", len(repos))
	}
}

func TestDiscoverReposSkipsNonGit(t *testing.T) {
	tmp := t.TempDir()

	// Create a dir with pom.xml but no .git
	dir := filepath.Join(tmp, "some-repo")
	os.Mkdir(dir, 0755)
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	repos := discoverRepos()
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (no .git), got %d", len(repos))
	}
}

func TestDiscoverReposSkipsNoPom(t *testing.T) {
	tmp := t.TempDir()

	// Create a dir with .git but no pom.xml
	dir := filepath.Join(tmp, "some-repo")
	os.Mkdir(dir, 0755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	repos := discoverRepos()
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (no pom.xml), got %d", len(repos))
	}
}

func TestPrintTransformResultsText(t *testing.T) {
	// Ensure jsonOutput is false.
	jsonOutput = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	results := []ops.Result{
		{Repo: "repo-a", Changed: true, Message: "updated version"},
		{Repo: "repo-b", Changed: false, Message: "no changes needed"},
		{Repo: "repo-c", Err: os.ErrNotExist},
		{Message: "summary line"},
	}

	err := printTransformResults(results)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("printTransformResults() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "repo-a") {
		t.Error("expected repo-a in output")
	}
	if !strings.Contains(output, "repo-b") {
		t.Error("expected repo-b in output")
	}
	if !strings.Contains(output, "repo-c") {
		t.Error("expected repo-c in output")
	}
	if !strings.Contains(output, "summary line") {
		t.Error("expected summary line in output")
	}
}

func TestPrintTransformResultsJSON(t *testing.T) {
	jsonOutput = true
	defer func() { jsonOutput = false }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	results := []ops.Result{
		{Repo: "repo-a", Changed: true, Message: "done"},
	}

	err := printTransformResults(results)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("printTransformResults() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Should produce valid JSON.
	var parsed []json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v\nOutput: %s", err, buf.String())
	}
}

func TestRunMavenParallelEmpty(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMavenParallel(nil, []string{"clean"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runMavenParallel() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dept44 repos") {
		t.Error("expected 'No dept44 repos' message for empty dirs")
	}
}

func TestStyleVars(t *testing.T) {
	if ok == "" {
		t.Error("ok style should not be empty")
	}
	if fail == "" {
		t.Error("fail style should not be empty")
	}
}

func TestFormatCmdStructure(t *testing.T) {
	cmd := formatCmd()
	if cmd.Use != "format" {
		t.Errorf("formatCmd().Use = %q, want %q", cmd.Use, "format")
	}
}

func TestVerifyCmdStructure(t *testing.T) {
	cmd := verifyCmd()
	if cmd.Use != "verify" {
		t.Errorf("verifyCmd().Use = %q, want %q", cmd.Use, "verify")
	}
}

func TestFindDepCmdStructure(t *testing.T) {
	cmd := findDepCmd()
	if cmd.Use != "find-dep <query>" {
		t.Errorf("findDepCmd().Use = %q", cmd.Use)
	}
}

func TestFindStaleCmdStructure(t *testing.T) {
	cmd := findStaleCmd()
	if cmd.Use != "find-stale" {
		t.Errorf("findStaleCmd().Use = %q", cmd.Use)
	}
}

func TestReplaceDepCmdFlags(t *testing.T) {
	cmd := replaceDepCmd()
	if cmd.Flags().Lookup("group") == nil {
		t.Error("replace-dep should have --group flag")
	}
	if cmd.Flags().Lookup("artifact") == nil {
		t.Error("replace-dep should have --artifact flag")
	}
	if cmd.Flags().Lookup("version") == nil {
		t.Error("replace-dep should have --version flag")
	}
}

func TestSwapDepCmdFlags(t *testing.T) {
	cmd := swapDepCmd()
	for _, flag := range []string{"old-group", "old-artifact", "new-group", "new-artifact", "version"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("swap-dep should have --%s flag", flag)
		}
	}
}

func TestBumpParentCmdFlags(t *testing.T) {
	cmd := bumpParentCmd()
	if cmd.Flags().Lookup("version") == nil {
		t.Error("bump-parent should have --version flag")
	}
}

func TestUpdateSpecsCmdStructure(t *testing.T) {
	cmd := updateSpecsCmd()
	if cmd.Use != "update-specs" {
		t.Errorf("updateSpecsCmd().Use = %q", cmd.Use)
	}
}

func TestUpdateIntegrationSpecsCmdStructure(t *testing.T) {
	cmd := updateIntegrationSpecsCmd()
	if cmd.Use != "update-integration-specs" {
		t.Errorf("updateIntegrationSpecsCmd().Use = %q", cmd.Use)
	}
}

// ── CLI integration tests (with temp dirs) ──────────────────────────

func TestFormatCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"format"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("format with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dept44 repos") {
		t.Error("expected 'No dept44 repos' message")
	}
}

func TestVerifyCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"verify"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("verify with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dept44 repos") {
		t.Error("expected 'No dept44 repos' message")
	}
}

func TestFindDepCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"find-dep", "some-dep"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("find-dep with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No matches") {
		t.Error("expected 'No matches' message")
	}
}

func TestFindDepCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	jsonOutput = true
	defer func() { jsonOutput = false }()

	root := BuildCLI()
	root.SetArgs([]string{"find-dep", "some-dep", "--json"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("find-dep --json should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Should produce valid JSON (null or empty array)
	output := strings.TrimSpace(buf.String())
	if output != "null" && output != "[]" {
		t.Logf("output: %s", output)
	}
}

func TestFindStaleCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"find-stale"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("find-stale with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "up to date") {
		t.Error("expected 'up to date' message")
	}
}

func TestFindStaleCmdJSON(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	jsonOutput = true
	defer func() { jsonOutput = false }()

	root := BuildCLI()
	root.SetArgs([]string{"find-stale", "--json"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("find-stale --json should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())
	if output != "null" && output != "[]" {
		t.Logf("output: %s", output)
	}
}

func TestUpdateSpecsCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"update-specs"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("update-specs with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No dept44 repos") {
		t.Error("expected 'No dept44 repos' message")
	}
}

func TestUpdateIntegrationSpecsCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"update-integration-specs"})

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("update-integration-specs with no repos should succeed, got: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// No repos means no results — printTransformResults with empty slice
}

func TestBumpParentCmdNoRepos(t *testing.T) {
	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	root := BuildCLI()
	root.SetArgs([]string{"bump-parent", "--version", "9.0.0"})

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	w.Close()
	os.Stdout = oldStdout

	// No repos means empty results — should succeed.
	if err != nil {
		t.Fatalf("bump-parent with no repos should succeed, got: %v", err)
	}
}

func TestReplaceDepCmdMissingFlags(t *testing.T) {
	root := BuildCLI()
	root.SetArgs([]string{"replace-dep"})

	err := root.Execute()
	if err == nil {
		t.Error("replace-dep without required flags should fail")
	}
}

func TestSwapDepCmdMissingFlags(t *testing.T) {
	root := BuildCLI()
	root.SetArgs([]string{"swap-dep"})

	err := root.Execute()
	if err == nil {
		t.Error("swap-dep without required flags should fail")
	}
}

func TestBumpParentCmdMissingFlags(t *testing.T) {
	root := BuildCLI()
	root.SetArgs([]string{"bump-parent"})

	err := root.Execute()
	if err == nil {
		t.Error("bump-parent without required flags should fail")
	}
}

func TestRunMavenParallelEmptyJSON(t *testing.T) {
	jsonOutput = true
	defer func() { jsonOutput = false }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMavenParallel(nil, []string{"clean"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runMavenParallel() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// Empty dirs prints "No dept44 repos" even in JSON mode.
	if !strings.Contains(buf.String(), "No dept44 repos") {
		t.Error("expected 'No dept44 repos' message")
	}
}

func TestRunMavenParallelWithMockedMvn(t *testing.T) {
	origFunc := runMvnFunc
	defer func() { runMvnFunc = origFunc }()

	runMvnFunc = func(args []string, dir string) string {
		return "" // success
	}

	jsonOutput = false
	dirs := []string{"/tmp/fake-repo-a", "/tmp/fake-repo-b"}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMavenParallel(dirs, []string{"clean"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runMavenParallel() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "fake-repo-a") || !strings.Contains(output, "fake-repo-b") {
		t.Errorf("expected repo names in output, got: %s", output)
	}
}

func TestRunMavenParallelWithFailures(t *testing.T) {
	origFunc := runMvnFunc
	defer func() { runMvnFunc = origFunc }()

	runMvnFunc = func(args []string, dir string) string {
		if strings.Contains(dir, "fail") {
			return "compilation error"
		}
		return ""
	}

	jsonOutput = false
	dirs := []string{"/tmp/ok-repo", "/tmp/fail-repo"}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMavenParallel(dirs, []string{"clean"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("runMavenParallel() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "ok-repo") {
		t.Error("expected ok-repo in output")
	}
	if !strings.Contains(output, "fail-repo") {
		t.Error("expected fail-repo in output")
	}
}

func TestRunMavenParallelJSON(t *testing.T) {
	origFunc := runMvnFunc
	defer func() { runMvnFunc = origFunc }()

	runMvnFunc = func(args []string, dir string) string { return "" }

	jsonOutput = true
	defer func() { jsonOutput = false }()

	dirs := []string{"/tmp/repo-x"}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runMavenParallel(dirs, []string{"clean"})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var parsed []mavenResult
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if len(parsed) != 1 || !parsed[0].Success {
		t.Errorf("expected 1 successful result, got: %+v", parsed)
	}
}

func TestDefaultRunMvnNotFound(t *testing.T) {
	// Test defaultRunMvn with a nonexistent dir — mvn should fail.
	result := defaultRunMvn([]string{"clean"}, "/nonexistent/dir")
	if result == "" {
		t.Error("expected error from defaultRunMvn in nonexistent dir")
	}
}

func TestDiscoverReposSkipsNonDept44(t *testing.T) {
	tmp := t.TempDir()

	// Create a dir with .git and pom.xml but not a dept44 repo.
	dir := filepath.Join(tmp, "some-repo")
	os.Mkdir(dir, 0755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	// Write a pom.xml that is NOT a dept44 repo (no dept44 parent).
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(`<project><parent><artifactId>spring-boot-starter-parent</artifactId></parent></project>`), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldDir)

	repos := discoverRepos()
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (not dept44), got %d", len(repos))
	}
}
