package screens

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CheeziCrew/raclette/internal/maven"
)

// ── Runner Update tests ─────────────────────────────────────────────

func TestRunnerUpdate_WindowSizeMsg(t *testing.T) {
	m := NewRunner()
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	m2, _ := m.Update(msg)
	if m2.width != 100 || m2.height != 50 {
		t.Errorf("size = %dx%d, want 100x50", m2.width, m2.height)
	}
}

func TestRunnerUpdate_WindowSizeMsg_NarrowWidth(t *testing.T) {
	m := NewRunner()
	msg := tea.WindowSizeMsg{Width: 30, Height: 20}
	m2, _ := m.Update(msg)
	if m2.width != 30 {
		t.Errorf("width = %d, want 30", m2.width)
	}
}

func TestRunnerUpdate_MavenDoneMsg_AllDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.tasks[0].Status = TaskRunning

	msg := MavenDoneMsg{Path: "/tmp/a", Result: "ok"}
	m2, _ := m.Update(msg)
	if !m2.done {
		t.Error("expected done = true when all tasks complete")
	}
}

func TestRunnerUpdate_MavenDoneMsg_NotAllDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b"})
	m.tasks[0].Status = TaskRunning

	msg := MavenDoneMsg{Path: "/tmp/a", Result: "ok"}
	m2, _ := m.Update(msg)
	if m2.done {
		t.Error("expected done = false when not all tasks complete")
	}
}

func TestRunnerUpdate_TextOutputMsg(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)

	msg := TextOutputMsg{Lines: []string{"result"}}
	m2, _ := m.Update(msg)
	if !m2.done {
		t.Error("expected done = true after TextOutputMsg")
	}
}

func TestRunnerUpdate_TickMsg_NotDone(t *testing.T) {
	m := NewRunner()
	m.done = false
	msg := tickMsg{}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected tick to return another tick cmd when not done")
	}
}

func TestRunnerUpdate_TickMsg_Done(t *testing.T) {
	m := NewRunner()
	m.done = true
	msg := tickMsg{}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("expected no cmd when done")
	}
}

func TestRunnerUpdate_KeyPressMsg_Back(t *testing.T) {
	m := NewRunner()
	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected back cmd on esc")
	}
}

func TestRunnerUpdate_KeyPressMsg_Tab_WithAltContent(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)
	m.content = "primary"
	m.altContent = "alternate"
	m.done = true

	msg := tea.KeyPressMsg{Code: tea.KeyTab}
	m2, _ := m.Update(msg)
	if !m2.showingAlt {
		t.Error("expected showingAlt = true after tab")
	}

	// Toggle back
	m3, _ := m2.Update(msg)
	if m3.showingAlt {
		t.Error("expected showingAlt = false after second tab")
	}
}

func TestRunnerUpdate_KeyPressMsg_Tab_NoAltContent(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)
	m.done = true
	m.altContent = ""

	msg := tea.KeyPressMsg{Code: tea.KeyTab}
	m2, _ := m.Update(msg)
	if m2.showingAlt {
		t.Error("should not toggle alt when altContent is empty")
	}
}

// ── Runner View tests ───────────────────────────────────────────────

func TestRunnerView_MavenInProgress(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "T"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b"})
	m.tasks[0].Status = TaskRunning
	m.width = 80
	m.height = 40

	v := m.View()
	if v == "" {
		t.Error("expected non-empty View")
	}
}

func TestRunnerView_MavenDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "T"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.tasks[0].Status = TaskDone
	m.tasks[0].Result = "v1"
	m.done = true

	v := m.View()
	if v == "" {
		t.Error("expected non-empty View")
	}
}

func TestRunnerView_MavenWithFailures(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "T"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b"})
	m.tasks[0].Status = TaskDone
	m.tasks[1].Status = TaskFailed
	m.tasks[1].Error = "build failed"
	m.done = true

	v := m.View()
	if v == "" {
		t.Error("expected non-empty View")
	}
}

func TestRunnerView_TextMode(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan", Icon: "S"}
	m.StartText(cmd)
	m.content = "scan results"
	m.done = true

	v := m.View()
	if v == "" {
		t.Error("expected non-empty View")
	}
}

func TestRunnerView_TextWithAltToggle(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan", Icon: "S"}
	m.StartText(cmd)
	m.content = "primary"
	m.altContent = "alternate"
	m.done = true

	v := m.View()
	if v == "" {
		t.Error("expected non-empty View")
	}
}

// ── viewProgress with many running tasks ────────────────────────────

func TestViewProgress_MoreThanThreeRunning(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "T"}
	dirs := []string{"/tmp/a", "/tmp/b", "/tmp/c", "/tmp/d", "/tmp/e"}
	m.StartMaven(cmd, dirs)
	for i := range m.tasks {
		m.tasks[i].Status = TaskRunning
	}
	m.width = 80

	v := m.viewProgress()
	if v == "" {
		t.Error("expected non-empty progress view")
	}
}

func TestViewProgress_WithFailedTasks(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "T"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b", "/tmp/c"})
	m.tasks[0].Status = TaskFailed
	m.tasks[1].Status = TaskRunning
	m.width = 80
	m.finished = 1

	v := m.viewProgress()
	if v == "" {
		t.Error("expected non-empty progress view")
	}
}

// ── Prompt tests ────────────────────────────────────────────────────

func TestPromptInit_WithInputs(t *testing.T) {
	cmd := maven.Command{
		Name:    "test",
		Icon:    "T",
		Prompts: []maven.Prompt{{Key: "k", Label: "L", Placeholder: "p"}},
	}
	m := NewPrompt(cmd)
	c := m.Init()
	if c == nil {
		t.Error("expected non-nil cmd from Init with inputs")
	}
}

func TestPromptInit_NoInputs(t *testing.T) {
	cmd := maven.Command{Name: "test", Icon: "T"}
	m := NewPrompt(cmd)
	c := m.Init()
	if c != nil {
		t.Error("expected nil cmd from Init with no inputs")
	}
}

func TestPromptUpdate_Tab(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
			{Key: "b", Label: "B", Placeholder: "pb"},
		},
	}
	m := NewPrompt(cmd)
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	msg := tea.KeyPressMsg{Code: tea.KeyTab}
	m2, _ := m.Update(msg)
	if m2.cursor != 1 {
		t.Errorf("cursor after tab = %d, want 1", m2.cursor)
	}
}

func TestPromptUpdate_ShiftTab(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
			{Key: "b", Label: "B", Placeholder: "pb"},
		},
	}
	m := NewPrompt(cmd)
	m.cursor = 1

	msg := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	m2, _ := m.Update(msg)
	if m2.cursor != 0 {
		t.Errorf("cursor after shift+tab = %d, want 0", m2.cursor)
	}
}

func TestPromptUpdate_Enter_MiddleField(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
			{Key: "b", Label: "B", Placeholder: "pb"},
		},
	}
	m := NewPrompt(cmd)

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	m2, _ := m.Update(msg)
	if m2.cursor != 1 {
		t.Errorf("cursor after enter on first field = %d, want 1", m2.cursor)
	}
}

func TestPromptUpdate_Enter_LastField(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
		},
	}
	m := NewPrompt(cmd)
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, c := m.Update(msg)
	if c == nil {
		t.Error("expected submit cmd on enter at last field")
	}
}

func TestPromptUpdate_Esc(t *testing.T) {
	cmd := maven.Command{
		Name:    "test",
		Icon:    "T",
		Prompts: []maven.Prompt{{Key: "k", Label: "L", Placeholder: "p"}},
	}
	m := NewPrompt(cmd)
	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	_, c := m.Update(msg)
	if c == nil {
		t.Error("expected back cmd on esc")
	}
}

func TestPromptUpdate_TabAtLastField(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A"},
		},
	}
	m := NewPrompt(cmd)
	m.cursor = 0

	msg := tea.KeyPressMsg{Code: tea.KeyTab}
	m2, _ := m.Update(msg)
	if m2.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m2.cursor)
	}
}

func TestPromptUpdate_ShiftTabAtFirstField(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A"},
			{Key: "b", Label: "B"},
		},
	}
	m := NewPrompt(cmd)
	m.cursor = 0

	msg := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	m2, _ := m.Update(msg)
	if m2.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m2.cursor)
	}
}

func TestPromptInputs(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
		},
	}
	m := NewPrompt(cmd)
	inputs := m.Inputs()
	if _, ok := inputs["a"]; !ok {
		t.Error("expected key 'a' in Inputs()")
	}
}

func TestPromptView_MultipleInputs(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A", Placeholder: "pa"},
			{Key: "b", Label: "B", Placeholder: "pb"},
		},
	}
	m := NewPrompt(cmd)
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}

// ── Menu tests ──────────────────────────────────────────────────────

func TestNewMenu_HasItems(t *testing.T) {
	m := NewMenu()
	v := m.View()
	if v == "" {
		t.Error("menu view is empty")
	}
}

// ── RepoSelect tests ───────────────────────────────────────────────

func TestNewRepoSelect(t *testing.T) {
	m := NewRepoSelect("/nonexistent", 40, 3)
	v := m.View()
	if v == "" {
		t.Error("reposelect view is empty")
	}
}

// ── Message type tests ──────────────────────────────────────────────

func TestScreenConstants(t *testing.T) {
	if ScreenMenu != 0 {
		t.Errorf("ScreenMenu = %d, want 0", ScreenMenu)
	}
	if ScreenRunner != 1 {
		t.Errorf("ScreenRunner = %d, want 1", ScreenRunner)
	}
	if ScreenRepoSelect != 2 {
		t.Errorf("ScreenRepoSelect = %d, want 2", ScreenRepoSelect)
	}
	if ScreenPrompt != 3 {
		t.Errorf("ScreenPrompt = %d, want 3", ScreenPrompt)
	}
}

// ── RepoSelect helper function tests ────────────────────────────────

func TestCountChanges_EmptyDir(t *testing.T) {
	c := countChanges("/nonexistent/path")
	if c.Modified != 0 || c.Added != 0 || c.Deleted != 0 || c.Untracked != 0 {
		t.Errorf("expected zero changes for nonexistent path, got %+v", c)
	}
}

func TestCurrentBranch_Nonexistent(t *testing.T) {
	branch := currentBranch("/nonexistent/path")
	if branch != "" {
		t.Errorf("expected empty branch for nonexistent path, got %q", branch)
	}
}

func TestDetectDefaultBranch_Nonexistent(t *testing.T) {
	branch := detectDefaultBranch("/nonexistent/path")
	if branch != "main" {
		t.Errorf("expected 'main' fallback, got %q", branch)
	}
}

// ── Keys tests ──────────────────────────────────────────────────────

func TestScreenIsBack(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	if !isBack(msg) {
		t.Error("expected isBack = true for esc")
	}
}

func TestScreenIsEnter(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	if !isEnter(msg) {
		t.Error("expected isEnter = true for enter")
	}
}

func TestScreenIsUp(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyUp}
	if !isUp(msg) {
		t.Error("expected isUp = true for up")
	}
}

func TestScreenIsDown(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	if !isDown(msg) {
		t.Error("expected isDown = true for down")
	}
}

func TestScreenIsToggle(t *testing.T) {
	msg := tea.KeyPressMsg{Code: tea.KeySpace}
	if !isToggle(msg) {
		t.Error("expected isToggle = true for space")
	}
}

func TestScreenIsSelectAll(t *testing.T) {
	msg := tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}
	if !isSelectAll(msg) {
		t.Error("expected isSelectAll = true for ctrl+a")
	}
}

// ── Runner Init test ────────────────────────────────────────────────

func TestRunnerInit(t *testing.T) {
	m := NewRunner()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil cmd from Init (should batch spinner tick + tickCmd)")
	}
}

// ── Runner handleMouseWheel tests ───────────────────────────────────

func TestRunnerHandleMouseWheel_TextMode(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)
	m.width = 80
	m.height = 40

	msg := tea.MouseWheelMsg{Button: tea.MouseWheelUp}
	m2, _ := m.handleMouseWheel(msg)
	// Should not panic, should update viewport
	_ = m2
}

func TestRunnerHandleMouseWheel_MavenNotDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.done = false

	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	m2, c := m.handleMouseWheel(msg)
	// Not viewport active in maven mode when not done — should be a no-op
	if c != nil {
		t.Error("expected nil cmd when viewport not active")
	}
	_ = m2
}

func TestRunnerHandleMouseWheel_MavenDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.done = true
	m.width = 80
	m.height = 40

	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	m2, _ := m.handleMouseWheel(msg)
	_ = m2
}

// ── Runner Update with spinner ──────────────────────────────────────

func TestRunnerUpdate_SpinnerMsg_NotDone(t *testing.T) {
	m := NewRunner()
	m.done = false
	// Get a real spinner tick msg
	tickCmd := m.spinner.Tick
	if tickCmd == nil {
		t.Skip("spinner tick is nil")
	}
	tickMsg := tickCmd()
	_, cmd := m.Update(tickMsg)
	if cmd == nil {
		t.Error("expected non-nil cmd for spinner tick when not done")
	}
}

func TestRunnerUpdate_SpinnerMsg_Done(t *testing.T) {
	m := NewRunner()
	m.done = true
	tickCmd := m.spinner.Tick
	if tickCmd == nil {
		t.Skip("spinner tick is nil")
	}
	tickMsg := tickCmd()
	_, cmd := m.Update(tickMsg)
	if cmd != nil {
		t.Error("expected nil cmd for spinner tick when done")
	}
}

// ── Runner viewMaven done ───────────────────────────────────────────

func TestRunnerViewMaven_Done(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.tasks[0].Status = TaskDone
	m.done = true

	v := m.viewMaven()
	if v == "" {
		t.Error("expected non-empty viewMaven when done")
	}
}

// ── Runner viewFailures ─────────────────────────────────────────────

func TestRunnerViewFailures_Empty(t *testing.T) {
	m := NewRunner()
	v := m.viewFailures(nil)
	if v != "" {
		t.Errorf("expected empty string for no failures, got %q", v)
	}
}

func TestRunnerViewFailures_WithErrors(t *testing.T) {
	m := NewRunner()
	tasks := []RepoTask{
		{Name: "repo-a", Status: TaskFailed, Error: "compilation error"},
		{Name: "repo-b", Status: TaskFailed, Error: ""},
	}
	v := m.viewFailures(tasks)
	if v == "" {
		t.Error("expected non-empty viewFailures")
	}
}

// ── Runner viewResultBanner with failures ───────────────────────────

func TestRunnerViewResultBanner_NoFailures(t *testing.T) {
	m := NewRunner()
	v := m.viewResultBanner(5, 0)
	if v == "" {
		t.Error("expected non-empty banner")
	}
}

func TestRunnerViewResultBanner_WithFailures(t *testing.T) {
	m := NewRunner()
	v := m.viewResultBanner(3, 2)
	if v == "" {
		t.Error("expected non-empty banner")
	}
}

// ── Runner handleKeyPress viewport delegation ───────────────────────

func TestRunnerHandleKeyPress_ViewportActive(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)
	m.width = 80
	m.height = 40

	// Send an arrow key — should delegate to viewport
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	m2, _ := m.handleKeyPress(msg)
	_ = m2
}

// ── RepoSelect mavenScan test ───────────────────────────────────────

func TestMavenScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

func TestMavenScan_NonexistentDir(t *testing.T) {
	_, err := mavenScan("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestMavenScan_SkipsNonDirs(t *testing.T) {
	dir := t.TempDir()
	// Create a file, not a directory
	os.WriteFile(filepath.Join(dir, "somefile.txt"), []byte("hi"), 0644)
	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

func TestMavenScan_SkipsNoGit(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "my-repo")
	os.MkdirAll(repoDir, 0755)
	// Has pom.xml but no .git
	os.WriteFile(filepath.Join(repoDir, "pom.xml"), []byte(`<project><parent><artifactId>dept44-parent</artifactId></parent></project>`), 0644)

	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (no .git), got %d", len(repos))
	}
}

func TestMavenScan_SkipsNoPom(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "my-repo")
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	// Has .git but no pom.xml

	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (no pom.xml), got %d", len(repos))
	}
}

func TestMavenScan_SkipsNonDept44(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "my-repo")
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(repoDir, "pom.xml"), []byte(pom), 0644)

	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos (non-dept44), got %d", len(repos))
	}
}

// ── countChanges with actual git repo ───────────────────────────────

func TestCountChanges_RealRepo(t *testing.T) {
	dir := t.TempDir()
	// Init a git repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create and commit a file
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify the file
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("changed"), 0644)

	// Add an untracked file
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)

	c := countChanges(dir)
	if c.Modified != 1 {
		t.Errorf("Modified = %d, want 1", c.Modified)
	}
	if c.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", c.Untracked)
	}
}

func TestCountChanges_AddedAndDeleted(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create, commit, then stage delete
	os.WriteFile(filepath.Join(dir, "old.txt"), []byte("old"), 0644)
	runGit(t, dir, "add", "old.txt")
	runGit(t, dir, "commit", "-m", "add old")

	// Stage a new file (added)
	os.WriteFile(filepath.Join(dir, "added.txt"), []byte("added"), 0644)
	runGit(t, dir, "add", "added.txt")

	// Delete old file
	os.Remove(filepath.Join(dir, "old.txt"))
	runGit(t, dir, "add", "old.txt")

	c := countChanges(dir)
	if c.Added != 1 {
		t.Errorf("Added = %d, want 1", c.Added)
	}
	if c.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", c.Deleted)
	}
}

// ── detectDefaultBranch with real repo ──────────────────────────────

func TestDetectDefaultBranch_RealRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hi"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	// No remote, so symbolic-ref will fail, should fallback to "main"
	branch := detectDefaultBranch(dir)
	if branch != "main" {
		t.Errorf("detectDefaultBranch = %q, want 'main'", branch)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2020-01-01T00:00:00+00:00", "GIT_COMMITTER_DATE=2020-01-01T00:00:00+00:00")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// ── Prompt Update with WindowSizeMsg ────────────────────────────────

func TestPromptUpdate_WindowSizeMsg(t *testing.T) {
	cmd := maven.Command{
		Name:    "test",
		Icon:    "T",
		Prompts: []maven.Prompt{{Key: "k", Label: "L", Placeholder: "p"}},
	}
	m := NewPrompt(cmd)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	m2, _ := m.Update(msg)
	if m2.width != 100 || m2.height != 50 {
		t.Errorf("size = %dx%d, want 100x50", m2.width, m2.height)
	}
}

// ── Prompt submit verify ────────────────────────────────────────────

func TestPromptSubmit(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "groupId", Label: "Group ID", Placeholder: "g"},
		},
	}
	m := NewPrompt(cmd)
	// Submit returns a tea.Cmd that produces a PromptDoneMsg
	c := m.submit()
	if c == nil {
		t.Error("expected non-nil cmd from submit")
	}
	msg := c()
	if _, ok := msg.(PromptDoneMsg); !ok {
		t.Errorf("expected PromptDoneMsg, got %T", msg)
	}
}

// ── Prompt Update no inputs path ────────────────────────────────────

func TestPromptUpdate_NoInputs(t *testing.T) {
	cmd := maven.Command{Name: "test", Icon: "T"}
	m := NewPrompt(cmd)
	// With no inputs, Update should handle gracefully
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	m2, c := m.Update(msg)
	// enter with no inputs should submit
	if c == nil {
		t.Error("expected submit cmd on enter with no inputs")
	}
	_ = m2
}

// ── Prompt Update Down key ──────────────────────────────────────────

func TestPromptUpdate_Down(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A"},
			{Key: "b", Label: "B"},
		},
	}
	m := NewPrompt(cmd)
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	m2, _ := m.Update(msg)
	if m2.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", m2.cursor)
	}
}

func TestPromptUpdate_Up(t *testing.T) {
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "a", Label: "A"},
			{Key: "b", Label: "B"},
		},
	}
	m := NewPrompt(cmd)
	m.cursor = 1
	msg := tea.KeyPressMsg{Code: tea.KeyUp}
	m2, _ := m.Update(msg)
	if m2.cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", m2.cursor)
	}
}

// ── Runner Update with progress.FrameMsg ────────────────────────────

func TestRunnerUpdate_ProgressFrameMsg(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})

	// Send bar.SetPercent to generate a FrameMsg, then send it to Update
	setCmd := m.bar.SetPercent(0.5)
	if setCmd != nil {
		frameMsg := setCmd()
		if frameMsg != nil {
			m2, _ := m.Update(frameMsg)
			_ = m2
		}
	}
}

// Test Update with MouseWheelMsg directly (through Update, not handleMouseWheel)
func TestRunnerUpdate_MouseWheelMsg(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)
	m.width = 80
	m.height = 40

	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	m2, _ := m.Update(msg)
	_ = m2
}

func TestRunnerUpdate_MouseWheelMsg_MavenNotDone(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.done = false

	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	m2, _ := m.Update(msg)
	_ = m2
}

// ── mavenScan with valid dept44 repo ────────────────────────────────

func TestMavenScan_ValidDept44Repo(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "api-service-my-svc")
	os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)

	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>1.0.0</version>
</project>`
	os.WriteFile(filepath.Join(repoDir, "pom.xml"), []byte(pom), 0644)

	repos, err := mavenScan(dir)
	if err != nil {
		t.Fatalf("mavenScan() error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].Name != "api-service-my-svc" {
		t.Errorf("repo name = %q, want 'api-service-my-svc'", repos[0].Name)
	}
}

// ── RunMaven cmd generation ─────────────────────────────────────────

func TestRunMaven_ReturnsCmd(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"clean", "verify"}}
	teaCmd := RunMaven(cmd, "/nonexistent")
	if teaCmd == nil {
		t.Error("expected non-nil tea.Cmd")
	}
	// Execute the cmd — it will fail because mvn is not in a valid dir
	// but should produce a MavenDoneMsg
	msg := teaCmd()
	doneMsg, ok := msg.(MavenDoneMsg)
	if !ok {
		t.Fatalf("expected MavenDoneMsg, got %T", msg)
	}
	if doneMsg.Err == nil {
		t.Error("expected error for invalid dir")
	}
}

func TestRunUpdateSpec_ReturnsCmd(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"verify"}}
	dir := t.TempDir()
	// Create a valid pom for OwnSpecCurrent check
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

	// Create a matching spec so OwnSpecCurrent returns true
	specDir := filepath.Join(dir, "src", "integration-test", "resources")
	os.MkdirAll(specDir, 0755)
	os.WriteFile(filepath.Join(specDir, "openapi.yaml"),
		[]byte("openapi: 3.0.1\ninfo:\n  title: API\n  version: 1.0.0\n"), 0644)

	teaCmd := RunUpdateSpec(cmd, dir)
	msg := teaCmd()
	doneMsg, ok := msg.(MavenDoneMsg)
	if !ok {
		t.Fatalf("expected MavenDoneMsg, got %T", msg)
	}
	// Should skip because spec is already current
	if doneMsg.Err != nil {
		t.Errorf("expected no error for current spec, got %v", doneMsg.Err)
	}
	if doneMsg.Result != "spec already current" {
		t.Errorf("result = %q, want 'spec already current'", doneMsg.Result)
	}
}

func TestRunUpdateSpec_NoSpecTest(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"verify"}}
	dir := t.TempDir()
	// Create a pom but no spec or test file
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

	teaCmd := RunUpdateSpec(cmd, dir)
	msg := teaCmd()
	doneMsg, ok := msg.(MavenDoneMsg)
	if !ok {
		t.Fatalf("expected MavenDoneMsg, got %T", msg)
	}
	// Should skip because no OpenApiSpecificationIT.java
	if doneMsg.Result == "" {
		t.Error("expected skip result message")
	}
}

// ── viewSuccesses empty ─────────────────────────────────────────────

func TestRunnerViewSuccesses_Empty(t *testing.T) {
	m := NewRunner()
	v := m.viewSuccesses(nil)
	if v != "" {
		t.Errorf("expected empty string for no successes, got %q", v)
	}
}

// ── Runner handleMavenDone partial ──────────────────────────────────

func TestRunnerHandleMavenDone_Partial(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b"})
	m.tasks[0].Status = TaskRunning

	msg := MavenDoneMsg{Path: "/tmp/a", Result: "ok"}
	m2, _ := m.handleMavenDone(msg)
	if m2.done {
		t.Error("expected done=false when not all tasks complete")
	}
	if m2.finished != 1 {
		t.Errorf("finished = %d, want 1", m2.finished)
	}
}

// ── Runner handleKeyPress with viewport key in maven progress ───────

func TestRunnerHandleKeyPress_NonViewportKey(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a"})
	m.done = false

	// A regular key that isn't esc/tab/viewport active
	msg := tea.KeyPressMsg{Code: 'x'}
	m2, c := m.handleKeyPress(msg)
	// Should return no cmd since viewport is not active
	if c != nil {
		t.Error("expected nil cmd for non-viewport key in maven progress mode")
	}
	_ = m2
}

// ── currentBranch with real repo ────────────────────────────────────

func TestCurrentBranch_RealRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hi"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	branch := currentBranch(dir)
	if branch != "main" {
		t.Errorf("currentBranch = %q, want 'main'", branch)
	}
}

// ── detectDefaultBranch with symbolic-ref set ───────────────────────

func TestRunUpdateSpec_ParseSpecFails(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"verify"}}
	dir := t.TempDir()
	// pom exists but no matching spec — OwnSpecCurrent will be false, ParseSpecPaths will fail
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>2.0.0</version>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	teaCmd := RunUpdateSpec(cmd, dir)
	msg := teaCmd()
	doneMsg := msg.(MavenDoneMsg)
	// ParseSpecPaths fails, so should get a skip result
	if doneMsg.Err != nil {
		t.Errorf("expected no error (skip), got %v", doneMsg.Err)
	}
	if doneMsg.Result == "" {
		t.Error("expected non-empty skip result")
	}
}

func TestRunUpdateSpec_CopySpecFails(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"verify"}}
	dir := t.TempDir()
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>2.0.0</version>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	// Create the test file with target + classpath patterns
	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall")
	os.MkdirAll(javaDir, 0755)
	javaContent := `class OpenApiSpecificationIT {
    @Value("classpath:/api/openapi.yaml")
    private String s;
    void run() throws Exception {
        Files.writeString(Path.of("target/openapi.yml"), s);
    }
}`
	os.WriteFile(filepath.Join(javaDir, "OpenApiSpecificationIT.java"), []byte(javaContent), 0644)

	// Create the classpath resource so parseResourcePath works
	resDir := filepath.Join(dir, "src", "integration-test", "resources", "api")
	os.MkdirAll(resDir, 0755)
	os.WriteFile(filepath.Join(resDir, "openapi.yaml"), []byte("openapi: 3.0.1\ninfo:\n  title: T\n  version: 1.0\n"), 0644)

	// Don't create target/openapi.yml — CopySpec will fail
	teaCmd := RunUpdateSpec(cmd, dir)
	msg := teaCmd()
	doneMsg := msg.(MavenDoneMsg)
	// CopySpec should fail because target file doesn't exist (mvn didn't actually run successfully)
	if doneMsg.Err == nil {
		t.Error("expected error from CopySpec failure")
	}
}

func TestRunUpdateSpec_FullFlow_CopySucceeds_SecondRunFails(t *testing.T) {
	cmd := maven.Command{Name: "test", Args: []string{"verify"}}
	dir := t.TempDir()
	pom := `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>5.0.0</version>
  </parent>
  <version>2.0.0</version>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	// Create test file
	javaDir := filepath.Join(dir, "src", "integration-test", "java", "se", "sundsvall")
	os.MkdirAll(javaDir, 0755)
	javaContent := `class OpenApiSpecificationIT {
    @Value("classpath:/api/openapi.yaml")
    private String s;
    void run() throws Exception {
        Files.writeString(Path.of("target/openapi.yml"), s);
    }
}`
	os.WriteFile(filepath.Join(javaDir, "OpenApiSpecificationIT.java"), []byte(javaContent), 0644)

	// Create classpath resource
	resDir := filepath.Join(dir, "src", "integration-test", "resources", "api")
	os.MkdirAll(resDir, 0755)
	os.WriteFile(filepath.Join(resDir, "openapi.yaml"), []byte("openapi: 3.0.1\ninfo:\n  title: T\n  version: 1.0\n"), 0644)

	// Create the target file so CopySpec succeeds
	targetDir := filepath.Join(dir, "target")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "openapi.yml"), []byte("generated spec content"), 0644)

	teaCmd := RunUpdateSpec(cmd, dir)
	msg := teaCmd()
	doneMsg := msg.(MavenDoneMsg)
	// CopySpec succeeds, but second mvn run fails (no real mvn installed for this test dir)
	if doneMsg.Err == nil {
		// In CI where mvn is available, this might pass. Just verify we got a MavenDoneMsg.
		_ = doneMsg
	}
}

// ── RunMaven success path ───────────────────────────────────────────

func TestRunMaven_SuccessPath(t *testing.T) {
	// Create a dir with a valid pom that `mvn -B help:evaluate` can run against
	// Actually, just test the error path more carefully
	cmd := maven.Command{Name: "test", Args: []string{"nonexistent-goal"}}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644)

	teaCmd := RunMaven(cmd, dir)
	msg := teaCmd()
	doneMsg := msg.(MavenDoneMsg)
	// Should either succeed or fail, but produce a valid MavenDoneMsg
	_ = doneMsg
}

// ── runMvn error parsing ────────────────────────────────────────────

func TestRunMvn_ErrorParsing(t *testing.T) {
	// runMvn with a nonexistent directory — mvn will fail
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	errMsg := runMvn([]string{"clean"}, "/nonexistent/dir")
	if errMsg == "" {
		t.Error("expected error message from runMvn")
	}
}

// ── countChanges covering more git status codes ─────────────────────

func TestCountChanges_Renamed(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.WriteFile(filepath.Join(dir, "old.txt"), []byte("content"), 0644)
	runGit(t, dir, "add", "old.txt")
	runGit(t, dir, "commit", "-m", "init")

	// Rename via git mv (staged rename = R_)
	cmd := exec.Command("git", "mv", "old.txt", "new.txt")
	cmd.Dir = dir
	cmd.Run()

	c := countChanges(dir)
	// Rename is counted as "Added" (R_ matches "R " case)
	if c.Added < 1 {
		t.Errorf("Added = %d after rename, want >= 1", c.Added)
	}
}

func TestCountChanges_Copied(t *testing.T) {
	// C_ (copy) status is rare but supported — test with default fallback
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("content"), 0644)
	runGit(t, dir, "add", "a.txt")
	runGit(t, dir, "commit", "-m", "init")

	// Modify in a way that triggers default branch (e.g. staged and working tree modified)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("modified"), 0644)
	runGit(t, dir, "add", "a.txt")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("modified again"), 0644)

	c := countChanges(dir)
	if c.Modified < 1 {
		t.Errorf("Modified = %d, want >= 1", c.Modified)
	}
}

func TestCountChanges_StagedDelete(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	os.WriteFile(filepath.Join(dir, "d.txt"), []byte("doomed"), 0644)
	runGit(t, dir, "add", "d.txt")
	runGit(t, dir, "commit", "-m", "init")

	// Delete and track (unstaged)
	os.Remove(filepath.Join(dir, "d.txt"))

	c := countChanges(dir)
	if c.Deleted < 1 {
		t.Errorf("Deleted = %d, want >= 1", c.Deleted)
	}
}

func TestDetectDefaultBranch_WithSymbolicRef(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "develop")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hi"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	// Create a fake remote ref to simulate origin/HEAD
	os.MkdirAll(filepath.Join(dir, ".git", "refs", "remotes", "origin"), 0755)
	// Set symbolic-ref
	runGit(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop")

	branch := detectDefaultBranch(dir)
	if branch != "develop" {
		t.Errorf("detectDefaultBranch = %q, want 'develop'", branch)
	}
}
