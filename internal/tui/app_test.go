package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/raclette/internal/maven"
	"github.com/CheeziCrew/raclette/internal/tui/screens"
)

func TestNew(t *testing.T) {
	m := New()
	if m.current != screens.ScreenMenu {
		t.Errorf("initial screen = %d, want ScreenMenu (%d)", m.current, screens.ScreenMenu)
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil cmd")
	}
}

func TestView_Menu(t *testing.T) {
	m := New()
	v := m.View()
	if v.Content == "" {
		t.Error("View() on menu returned empty content")
	}
	if !v.AltScreen {
		t.Error("expected AltScreen = true")
	}
	if v.WindowTitle != "raclette" {
		t.Errorf("WindowTitle = %q, want raclette", v.WindowTitle)
	}
}

func TestView_Runner(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	v := m.View()
	if v.Content == "" {
		t.Error("View() on runner returned empty content")
	}
}

func TestView_Prompt(t *testing.T) {
	m := New()
	cmd := maven.Command{
		Name: "test",
		Icon: "T",
		Prompts: []maven.Prompt{
			{Key: "k", Label: "L", Placeholder: "p"},
		},
	}
	m.prompt = screens.NewPrompt(cmd)
	m.current = screens.ScreenPrompt
	v := m.View()
	if v.Content == "" {
		t.Error("View() on prompt returned empty content")
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := New()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.width != 120 || rm.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", rm.width, rm.height)
	}
}

func TestUpdate_CtrlC(t *testing.T) {
	m := New()
	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("ctrl+c should return a quit cmd")
	}
}

func TestUpdate_QuitFromMenu(t *testing.T) {
	m := New()
	msg := tea.KeyPressMsg{Code: 'q'}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("q from menu should return quit cmd")
	}
}

func TestUpdate_QuitFromRunner(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	msg := tea.KeyPressMsg{Code: 'q'}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.current != screens.ScreenMenu {
		t.Errorf("q from runner should go back to menu, got screen %d", rm.current)
	}
}

func TestUpdate_QNoQuitFromPrompt(t *testing.T) {
	m := New()
	cmd := maven.Command{
		Name:    "test",
		Icon:    "T",
		Prompts: []maven.Prompt{{Key: "k", Label: "L", Placeholder: "p"}},
	}
	m.prompt = screens.NewPrompt(cmd)
	m.current = screens.ScreenPrompt
	msg := tea.KeyPressMsg{Code: 'q'}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Should NOT quit or go back - q is valid text input in prompt
	if rm.current != screens.ScreenPrompt {
		t.Errorf("q from prompt should stay on prompt, got screen %d", rm.current)
	}
}

func TestUpdate_NavigateMsg(t *testing.T) {
	m := New()
	msg := screens.NavigateMsg{Screen: screens.ScreenRunner}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.current != screens.ScreenRunner {
		t.Errorf("expected ScreenRunner, got %d", rm.current)
	}
}

func TestUpdate_BackToMenuMsg(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	msg := curd.BackToMenuMsg{}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.current != screens.ScreenMenu {
		t.Errorf("expected ScreenMenu, got %d", rm.current)
	}
}

func TestUpdate_PromptDoneMsg(t *testing.T) {
	m := New()
	m.current = screens.ScreenPrompt
	m.pendingCmd = maven.Command{Name: "Find Dependency", Kind: maven.KindScan}
	msg := screens.PromptDoneMsg{Inputs: map[string]string{"query": "lombok"}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.pendingInputs["query"] != "lombok" {
		t.Errorf("expected query=lombok, got %v", rm.pendingInputs)
	}
}

func TestUpdate_MenuSelectionMsg_Unknown(t *testing.T) {
	m := New()
	msg := curd.MenuSelectionMsg{Command: "Nonexistent Command"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Should stay on menu if command not found
	if rm.current != screens.ScreenMenu {
		t.Errorf("expected ScreenMenu for unknown command, got %d", rm.current)
	}
}

func TestUpdate_MavenDoneMsg(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	m.pendingCmd = maven.Command{Name: "test", Kind: maven.KindMaven}
	m.runner.StartMaven(m.pendingCmd, []string{"/tmp/repo-a"})
	m.runner.MarkRunning("/tmp/repo-a")
	m.running = 1
	m.total = 1

	msg := screens.MavenDoneMsg{Path: "/tmp/repo-a", Result: "done"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	if rm.completed != 1 {
		t.Errorf("completed = %d, want 1", rm.completed)
	}
	if rm.running != 0 {
		t.Errorf("running = %d, want 0", rm.running)
	}
}

func TestHandleGlobalKey_CtrlC(t *testing.T) {
	m := New()
	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	cmd, handled := m.handleGlobalKey(msg)
	if !handled {
		t.Error("ctrl+c should be handled")
	}
	if cmd == nil {
		t.Error("ctrl+c should return a quit command")
	}
}

func TestHandleGlobalKey_QFromMenu(t *testing.T) {
	m := New()
	m.current = screens.ScreenMenu
	msg := tea.KeyPressMsg{Code: 'q'}
	cmd, handled := m.handleGlobalKey(msg)
	if !handled {
		t.Error("q from menu should be handled")
	}
	if cmd == nil {
		t.Error("q from menu should return quit")
	}
}

func TestHandleGlobalKey_QFromNonMenu(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	msg := tea.KeyPressMsg{Code: 'q'}
	_, handled := m.handleGlobalKey(msg)
	if !handled {
		t.Error("q from non-menu should be handled")
	}
	if m.current != screens.ScreenMenu {
		t.Errorf("expected screen to change to menu, got %d", m.current)
	}
}

func TestDelegateToScreen_Menu(t *testing.T) {
	m := New()
	m.current = screens.ScreenMenu
	// Sending nil should not panic
	_, _ = m.delegateToScreen(nil)
}

func TestDelegateToScreen_Runner(t *testing.T) {
	m := New()
	m.current = screens.ScreenRunner
	_, _ = m.delegateToScreen(nil)
}

func TestExecute_NilForUnknownKind(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Kind: maven.Kind(99)}
	cmd := m.execute()
	if cmd != nil {
		t.Error("expected nil cmd for unknown Kind")
	}
}

func TestStartMultiRepo_EmptyDirs(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "test", Kind: maven.KindMaven}
	cmd := m.startMultiRepo(nil)
	if cmd == nil {
		t.Error("expected non-nil cmd even for empty dirs")
	}
	if m.current != screens.ScreenRunner {
		t.Errorf("expected ScreenRunner, got %d", m.current)
	}
}

func TestStartMultiRepo_WithDirs(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "test", Kind: maven.KindMaven}
	cmd := m.startMultiRepo([]string{"/tmp/a", "/tmp/b"})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
	if m.current != screens.ScreenRunner {
		t.Errorf("expected ScreenRunner, got %d", m.current)
	}
	if m.total != 2 {
		t.Errorf("total = %d, want 2", m.total)
	}
}

func TestDrainQueue_MaxParallel(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "test", Kind: maven.KindMaven}
	m.queue = []string{"/a", "/b", "/c", "/d", "/e"}
	m.running = 0

	cmds := m.drainQueue()
	// Should drain up to maxParallel (3)
	if len(cmds) != maxParallel {
		t.Errorf("expected %d cmds, got %d", maxParallel, len(cmds))
	}
	if m.running != maxParallel {
		t.Errorf("running = %d, want %d", m.running, maxParallel)
	}
	if len(m.queue) != 2 {
		t.Errorf("queue len = %d, want 2", len(m.queue))
	}
}

func TestDrainQueue_UpdateSpec(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "test", Kind: maven.KindUpdateSpec}
	m.queue = []string{"/a"}
	m.running = 0

	cmds := m.drainQueue()
	if len(cmds) != 1 {
		t.Errorf("expected 1 cmd, got %d", len(cmds))
	}
}

func TestHandleRepoSelectDone_WithPrompts(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{
		Name: "test",
		Kind: maven.KindTransform,
		Prompts: []maven.Prompt{
			{Key: "k", Label: "L"},
		},
	}
	m.pendingInputs = nil

	result, _ := m.handleRepoSelectDone()
	rm := result.(Model)
	if rm.current != screens.ScreenPrompt {
		t.Errorf("expected ScreenPrompt, got %d", rm.current)
	}
}

func TestHandleRepoSelectDone_NoPrompts(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{
		Name: "Find Stale Specs",
		Kind: maven.KindScan,
	}

	_, cmd := m.handleRepoSelectDone()
	if cmd == nil {
		t.Error("expected non-nil cmd when no prompts")
	}
}

// ── handleMenuSelection tests ───────────────────────────────────────

func TestHandleMenuSelection_KnownCommand(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 40
	// Use a known command name from maven.Commands()
	msg := curd.MenuSelectionMsg{Command: "Find Stale Specs"}
	result, cmd := m.handleMenuSelection(msg)
	rm := result.(Model)
	if rm.current != screens.ScreenRepoSelect {
		t.Errorf("expected ScreenRepoSelect, got %d", rm.current)
	}
	if rm.pendingCmd.Name != "Find Stale Specs" {
		t.Errorf("pendingCmd.Name = %q, want 'Find Stale Specs'", rm.pendingCmd.Name)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for repo select init")
	}
}

func TestHandleMenuSelection_TransformCommand(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 40
	msg := curd.MenuSelectionMsg{Command: "Bump Parent Version"}
	result, _ := m.handleMenuSelection(msg)
	rm := result.(Model)
	if rm.current != screens.ScreenRepoSelect {
		t.Errorf("expected ScreenRepoSelect, got %d", rm.current)
	}
	if rm.pendingCmd.Kind != maven.KindTransform {
		t.Errorf("pendingCmd.Kind = %d, want KindTransform", rm.pendingCmd.Kind)
	}
}

// ── executeTransform tests ──────────────────────────────────────────

func TestExecuteTransform_ReplaceDependencyVersion(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 40
	m.pendingCmd = maven.Command{Name: "Replace Dependency Version", Kind: maven.KindTransform}
	m.pendingInputs = map[string]string{
		"groupId":    "org.example",
		"artifactId": "my-lib",
		"newVersion": "2.0.0",
	}

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

	// We need repoSelect to return dirs, but since it's a curd model we can't easily set it.
	// Instead, test executeTransform directly by setting up the model manually.
	cmd := m.executeTransform([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd from executeTransform")
	}
	if m.current != screens.ScreenRunner {
		t.Errorf("expected ScreenRunner, got %d", m.current)
	}
}

func TestExecuteTransform_SwapDependency(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Swap Dependency", Kind: maven.KindTransform}
	m.pendingInputs = map[string]string{
		"oldGroupId":    "com.old",
		"oldArtifactId": "old-lib",
		"newGroupId":    "com.new",
		"newArtifactId": "new-lib",
		"newVersion":    "2.0.0",
	}

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
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	cmd := m.executeTransform([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestExecuteTransform_BumpParentVersion(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Bump Parent Version", Kind: maven.KindTransform}
	m.pendingInputs = map[string]string{"newVersion": "5.0.0"}

	dir := t.TempDir()
	pom := `<project>
  <parent>
    <groupId>se.sundsvall.dept44</groupId>
    <artifactId>dept44-service-parent</artifactId>
    <version>4.0.0</version>
  </parent>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	cmd := m.executeTransform([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestExecuteTransform_UpdateIntegrationSpecs(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Update Integration Specs", Kind: maven.KindTransform}

	dir := t.TempDir()
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

	cmd := m.executeTransform([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

// ── executeScan tests ───────────────────────────────────────────────

func TestExecuteScan_FindDependency(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Find Dependency", Kind: maven.KindScan}
	m.pendingInputs = map[string]string{"query": "lombok"}

	dir := t.TempDir()
	pom := `<project>
  <dependencies>
    <dependency>
      <groupId>org.projectlombok</groupId>
      <artifactId>lombok</artifactId>
      <version>1.18.30</version>
    </dependency>
  </dependencies>
</project>`
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pom), 0644)

	cmd := m.executeScan([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
	if m.current != screens.ScreenRunner {
		t.Errorf("expected ScreenRunner, got %d", m.current)
	}
}

func TestExecuteScan_FindStaleSpecs(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Find Stale Specs", Kind: maven.KindScan}

	dir := t.TempDir()
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

	cmd := m.executeScan([]string{dir})
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

// ── execute routing tests ───────────────────────────────────────────

func TestExecute_Scan(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Find Stale Specs", Kind: maven.KindScan}
	cmd := m.execute()
	if cmd == nil {
		t.Error("expected non-nil cmd for KindScan")
	}
}

func TestExecute_Transform(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "Bump Parent Version", Kind: maven.KindTransform}
	m.pendingInputs = map[string]string{"newVersion": "5.0.0"}
	cmd := m.execute()
	if cmd == nil {
		t.Error("expected non-nil cmd for KindTransform")
	}
}

// ── View for ScreenRepoSelect ───────────────────────────────────────

func TestView_RepoSelect(t *testing.T) {
	m := New()
	m.pendingCmd = maven.Command{Name: "test", Icon: "T"}
	m.repoSelect = screens.NewRepoSelect("/nonexistent", 40, 3)
	m.current = screens.ScreenRepoSelect
	v := m.View()
	if v.Content == "" {
		t.Error("View() on reposelect returned empty content")
	}
}

// ── DelegateToScreen for all screens ────────────────────────────────

func TestDelegateToScreen_RepoSelect(t *testing.T) {
	m := New()
	m.current = screens.ScreenRepoSelect
	m.repoSelect = screens.NewRepoSelect("/nonexistent", 40, 3)
	_, _ = m.delegateToScreen(nil)
}

func TestDelegateToScreen_Prompt(t *testing.T) {
	m := New()
	m.current = screens.ScreenPrompt
	m.prompt = screens.NewPrompt(maven.Command{
		Name:    "test",
		Prompts: []maven.Prompt{{Key: "k", Label: "L"}},
	})
	_, _ = m.delegateToScreen(nil)
}
