package screens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CheeziCrew/raclette/internal/maven"
)

func TestNewRunnerView(t *testing.T) {
	m := NewRunner()
	v := m.View()
	if v == "" {
		t.Error("NewRunner().View() returned empty string")
	}
}

func TestStartMaven(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "🧪"}
	dirs := []string{"/tmp/repo-a", "/tmp/repo-b"}
	m.StartMaven(cmd, dirs)

	if m.mode != modeMaven {
		t.Errorf("mode = %d, want modeMaven (%d)", m.mode, modeMaven)
	}
	if len(m.tasks) != 2 {
		t.Fatalf("tasks count = %d, want 2", len(m.tasks))
	}
	if m.tasks[0].Name != "repo-a" {
		t.Errorf("tasks[0].Name = %q, want %q", m.tasks[0].Name, "repo-a")
	}
	if m.tasks[1].Name != "repo-b" {
		t.Errorf("tasks[1].Name = %q, want %q", m.tasks[1].Name, "repo-b")
	}
	if m.done {
		t.Error("expected done to be false after StartMaven")
	}
}

func TestStartText(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan", Icon: "🔍"}
	m.StartText(cmd)

	if m.mode != modeText {
		t.Errorf("mode = %d, want modeText (%d)", m.mode, modeText)
	}
	if m.done {
		t.Error("expected done to be false after StartText")
	}
}

func TestMarkRunning(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/repo-a", "/tmp/repo-b"})

	m.MarkRunning("/tmp/repo-a")
	if m.tasks[0].Status != TaskRunning {
		t.Errorf("tasks[0].Status = %d, want TaskRunning (%d)", m.tasks[0].Status, TaskRunning)
	}
	if m.tasks[1].Status != TaskPending {
		t.Errorf("tasks[1].Status = %d, want TaskPending (%d)", m.tasks[1].Status, TaskPending)
	}
}

func TestHasFailures(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/repo-a"})

	if m.hasFailures() {
		t.Error("hasFailures should be false initially")
	}

	m.tasks[0].Status = TaskFailed
	if !m.hasFailures() {
		t.Error("hasFailures should be true after setting a task to TaskFailed")
	}
}

func TestPartitionTasks(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/a", "/tmp/b", "/tmp/c"})

	m.tasks[0].Status = TaskDone
	m.tasks[1].Status = TaskFailed
	m.tasks[2].Status = TaskDone

	ok, fail := m.partitionTasks()
	if len(ok) != 2 {
		t.Errorf("ok count = %d, want 2", len(ok))
	}
	if len(fail) != 1 {
		t.Errorf("fail count = %d, want 1", len(fail))
	}
}

func TestHandleTextOutput(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan"}
	m.StartText(cmd)

	msg := TextOutputMsg{Lines: []string{"line1", "line2"}, AltContent: "alt view"}
	m = m.handleTextOutput(msg)

	if !m.done {
		t.Error("expected done to be true after TextOutputMsg")
	}
	if m.content != "line1\nline2" {
		t.Errorf("content = %q, want %q", m.content, "line1\nline2")
	}
	if m.altContent != "alt view" {
		t.Errorf("altContent = %q, want %q", m.altContent, "alt view")
	}
}

func TestApplyTaskResult_Success(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/repo-a"})
	m.tasks[0].Status = TaskRunning

	msg := MavenDoneMsg{Path: "/tmp/repo-a", Result: "1.2.3"}
	m.applyTaskResult(msg)

	if m.tasks[0].Status != TaskDone {
		t.Errorf("tasks[0].Status = %d, want TaskDone (%d)", m.tasks[0].Status, TaskDone)
	}
	if m.tasks[0].Result != "1.2.3" {
		t.Errorf("tasks[0].Result = %q, want %q", m.tasks[0].Result, "1.2.3")
	}
}

func TestApplyTaskResult_Failure(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test"}
	m.StartMaven(cmd, []string{"/tmp/repo-a"})
	m.tasks[0].Status = TaskRunning
	m.width = 80

	msg := MavenDoneMsg{Path: "/tmp/repo-a", Err: errString("build failed")}
	m.applyTaskResult(msg)

	if m.tasks[0].Status != TaskFailed {
		t.Errorf("tasks[0].Status = %d, want TaskFailed (%d)", m.tasks[0].Status, TaskFailed)
	}
	if m.tasks[0].Error == "" {
		t.Error("expected non-empty error string")
	}
}

func TestIsViewportActive(t *testing.T) {
	m := NewRunner()

	m.mode = modeText
	if !m.isViewportActive() {
		t.Error("isViewportActive should be true in modeText")
	}

	m.mode = modeMaven
	m.done = false
	if m.isViewportActive() {
		t.Error("isViewportActive should be false in modeMaven when not done")
	}

	m.done = true
	if !m.isViewportActive() {
		t.Error("isViewportActive should be true in modeMaven when done")
	}
}

func TestWriteLog(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	writeLog("my-repo", "stdout content", "stderr content")

	logPath := filepath.Join(tmp, ".cache", "raclette", "logs", "my-repo.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("log file is empty")
	}
}

func TestWriteLog_NoStderr(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	writeLog("clean-repo", "output only", "")

	logPath := filepath.Join(tmp, ".cache", "raclette", "logs", "clean-repo.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("log file is empty")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
