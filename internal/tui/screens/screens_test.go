package screens

import (
	"testing"

	"github.com/CheeziCrew/raclette/internal/maven"
)

func TestNewMenu(t *testing.T) {
	m := NewMenu()
	v := m.View()
	if v == "" {
		t.Error("NewMenu().View() returned empty string")
	}
}

func TestNewRunner(t *testing.T) {
	m := NewRunner()
	v := m.View()
	if v == "" {
		t.Error("NewRunner().View() returned empty string")
	}
}

func TestRunnerStartMaven(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "test", Icon: "🧪"}
	m.StartMaven(cmd, []string{"/tmp/repo-a", "/tmp/repo-b"})

	if len(m.tasks) != 2 {
		t.Errorf("tasks count = %d, want 2", len(m.tasks))
	}
	v := m.View()
	if v == "" {
		t.Error("RunnerModel.View() returned empty string after StartMaven")
	}
}

func TestRunnerStartText(t *testing.T) {
	m := NewRunner()
	cmd := maven.Command{Name: "scan", Icon: "🔍"}
	m.StartText(cmd)

	v := m.View()
	if v == "" {
		t.Error("RunnerModel.View() returned empty string after StartText")
	}
}

func TestNewPrompt_NoPrompts(t *testing.T) {
	cmd := maven.Command{Name: "test", Icon: "🧪"}
	m := NewPrompt(cmd)
	v := m.View()
	if v == "" {
		t.Error("NewPrompt(no prompts).View() returned empty string")
	}
}

func TestNewPrompt_WithPrompts(t *testing.T) {
	cmd := maven.Command{
		Name: "replace",
		Icon: "🔄",
		Prompts: []maven.Prompt{
			{Key: "groupId", Label: "Group ID", Placeholder: "com.example"},
			{Key: "artifactId", Label: "Artifact ID", Placeholder: "my-lib"},
		},
	}
	m := NewPrompt(cmd)
	if len(m.inputs) != 2 {
		t.Errorf("inputs count = %d, want 2", len(m.inputs))
	}
	v := m.View()
	if v == "" {
		t.Error("NewPrompt(with prompts).View() returned empty string")
	}
}

func TestPromptUpdate(t *testing.T) {
	cmd := maven.Command{
		Name:    "test",
		Icon:    "🧪",
		Prompts: []maven.Prompt{{Key: "k", Label: "L", Placeholder: "p"}},
	}
	m := NewPrompt(cmd)
	m2, _ := m.Update(nil)
	v := m2.View()
	if v == "" {
		t.Error("PromptModel.View() returned empty string after Update")
	}
}
