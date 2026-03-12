package screens

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/CheeziCrew/raclette/internal/maven"
	"github.com/CheeziCrew/raclette/internal/ops"
	"github.com/CheeziCrew/curd"
)

// Re-export task types from curd.
type RepoTaskStatus = curd.RepoTaskStatus
type RepoTask = curd.RepoTask

const (
	TaskPending = curd.TaskPending
	TaskRunning = curd.TaskRunning
	TaskDone    = curd.TaskDone
	TaskFailed  = curd.TaskFailed
)

// runnerMode distinguishes maven builds (progress+result) from scan/transform (text output).
type runnerMode int

const (
	modeMaven runnerMode = iota
	modeText
)

// RunnerModel shows progress during execution and results when done.
type RunnerModel struct {
	command maven.Command
	mode    runnerMode

	// Maven mode: task-based progress.
	tasks    []RepoTask
	bar      progress.Model
	spinner  spinner.Model
	finished int
	done     bool

	// Text mode: scan/transform output in a viewport.
	viewport   viewport.Model
	textDone   bool
	content    string // primary text content
	altContent string // alternate view (toggle with tab)
	showingAlt bool   // which view is active

	start   time.Time
	elapsed time.Duration
	width   int
	height  int

	// Styles from curd palette
	styles curd.StyleSet
}

// NewRunner creates a fresh runner.
func NewRunner() RunnerModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(curd.ColorBrYellow)

	bar := progress.New(
		progress.WithoutPercentage(),
		progress.WithWidth(40),
		progress.WithColors(curd.ColorBrYellow, curd.ColorGray),
	)

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))

	styles := curd.RaclettePalette.Styles()

	return RunnerModel{
		spinner:  s,
		bar:      bar,
		viewport: vp,
		styles:   styles,
	}
}

// StartMaven initializes the runner for multi-repo maven builds.
func (m *RunnerModel) StartMaven(cmd maven.Command, dirs []string) {
	m.command = cmd
	m.mode = modeMaven
	m.done = false
	m.finished = 0
	m.start = time.Now()
	m.elapsed = 0

	m.tasks = make([]RepoTask, len(dirs))
	for i, d := range dirs {
		m.tasks[i] = RepoTask{
			Name:   filepath.Base(d),
			Path:   d,
			Status: TaskPending,
		}
	}
}

// StartText initializes the runner for scan/transform text output.
func (m *RunnerModel) StartText(cmd maven.Command) {
	m.command = cmd
	m.mode = modeText
	m.textDone = false
	m.done = false
	m.start = time.Now()
	m.elapsed = 0
	m.viewport.SetContent("")
}


func (m RunnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickCmd())
}

// tickMsg updates the elapsed timer.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m RunnerModel) Update(msg tea.Msg) (RunnerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		w := msg.Width - 20
		if w > 60 {
			w = 60
		}
		if w < 20 {
			w = 20
		}
		m.bar.SetWidth(w)
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 8)

	case tea.MouseWheelMsg:
		if m.mode == modeText || (m.mode == modeMaven && m.done) {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if isBack(msg) {
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
		// Tab toggles between primary and alternate content.
		if msg.String() == "tab" && m.altContent != "" && m.done {
			m.showingAlt = !m.showingAlt
			if m.showingAlt {
				m.viewport.SetContent(m.altContent)
			} else {
				m.viewport.SetContent(m.content)
			}
			m.viewport.GotoTop()
			return m, nil
		}
		if m.mode == modeText || (m.mode == modeMaven && m.done) {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case MavenDoneMsg:
		// A single repo finished its maven build.
		for i := range m.tasks {
			if m.tasks[i].Path == msg.Path {
				if msg.Err != nil {
					m.tasks[i].Status = TaskFailed
					m.tasks[i].Error = curd.TruncateError(msg.Err.Error(), m.width-10)
				} else {
					m.tasks[i].Status = TaskDone
					m.tasks[i].Result = msg.Result
				}
				break
			}
		}

		m.finished = 0
		allDone := true
		for _, t := range m.tasks {
			switch t.Status {
			case TaskDone, TaskFailed:
				m.finished++
			default:
				allDone = false
			}
		}

		if allDone {
			m.done = true
			m.elapsed = time.Since(m.start)
			m.viewport.SetContent(m.viewResults())
			m.viewport.GotoTop()
			return m, tea.Batch(
				m.bar.SetPercent(1.0),
				func() tea.Msg { return AllDoneMsg{} },
			)
		}

		pct := float64(m.finished) / float64(len(m.tasks))
		cmds = append(cmds, m.bar.SetPercent(pct))

	case TextOutputMsg:
		// Scan/transform results.
		m.content = strings.Join(msg.Lines, "\n")
		m.altContent = msg.AltContent
		m.showingAlt = false
		m.viewport.SetContent(m.content)
		m.viewport.GotoTop()
		m.textDone = true
		m.done = true
		m.elapsed = time.Since(m.start)

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.bar, cmd = m.bar.Update(msg)
		cmds = append(cmds, cmd)

	case tickMsg:
		if !m.done {
			m.elapsed = time.Since(m.start)
			cmds = append(cmds, tickCmd())
		}

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m RunnerModel) View() string {
	var b strings.Builder

	// Header: command name + status indicator.
	header := fmt.Sprintf("%s %s", m.command.Icon, m.command.Name)
	if m.done {
		if m.hasFailures() {
			header += " " + m.styles.ResultFail.Render("✗")
		} else {
			header += " " + m.styles.ResultOk.Render("✓")
		}
	} else {
		header += " " + m.spinner.View()
	}
	b.WriteString(m.styles.Title.Render(header))
	b.WriteString("\n")

	elapsed := m.styles.ResultDim.Render(fmt.Sprintf("⏱ %s", m.elapsed.Round(time.Second)))
	b.WriteString(elapsed)
	b.WriteString("\n\n")

	if m.mode == modeText || (m.mode == modeMaven && m.done) {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(m.viewMaven())
	}

	b.WriteString("\n")
	hints := []curd.Hint{
		{Key: "esc/q", Desc: "back"},
	}
	if m.done && m.altContent != "" {
		hints = append([]curd.Hint{{Key: "tab", Desc: "toggle view"}}, hints...)
	}
	b.WriteString(curd.RenderHintBar(m.styles, hints))

	return b.String()
}

// viewMaven renders the progress bar or result summary.
func (m RunnerModel) viewMaven() string {
	if m.done {
		return m.viewResults()
	}
	return m.viewProgress()
}

// viewProgress shows progress bar + running tasks.
func (m RunnerModel) viewProgress() string {
	var s string
	total := len(m.tasks)
	var pct float64
	if total > 0 {
		pct = float64(m.finished) / float64(total)
	}

	s += "  " + m.bar.ViewAs(pct) + "\n"
	s += "  " + m.styles.CountStyle.Render(fmt.Sprintf("%d/%d", m.finished, total))

	failed := 0
	for _, t := range m.tasks {
		if t.Status == TaskFailed {
			failed++
		}
	}
	if failed > 0 {
		s += "  " + m.styles.ResultFail.Render(fmt.Sprintf("(%d failed)", failed))
	}
	s += "\n\n"

	// Show currently running tasks (max 3).
	running := 0
	for _, t := range m.tasks {
		if t.Status == TaskRunning {
			if running < 3 {
				s += fmt.Sprintf("  %s %s\n", m.spinner.View(), m.styles.NameStyle.Render(t.Name))
			}
			running++
		}
	}
	if running > 3 {
		s += m.styles.ResultDim.Render(fmt.Sprintf("  … and %d more", running-3)) + "\n"
	}

	return s
}

// viewResults shows the success/failure summary.
func (m RunnerModel) viewResults() string {
	var succeeded, failed int
	var okTasks, failTasks []RepoTask
	for _, t := range m.tasks {
		switch t.Status {
		case TaskDone:
			succeeded++
			okTasks = append(okTasks, t)
		case TaskFailed:
			failed++
			failTasks = append(failTasks, t)
		}
	}

	var s string

	// Summary banner.
	banner := m.styles.ResultAccent.Render(m.command.Name) + m.styles.ResultDim.Render("  ")
	banner += m.styles.ResultOk.Render(fmt.Sprintf("✔ %d", succeeded))
	if failed > 0 {
		banner += m.styles.ResultDim.Render("  ") + m.styles.ResultFail.Render(fmt.Sprintf("✗ %d", failed))
	}
	s += m.styles.SummaryBox.Render(banner) + "\n\n"

	// Failures first.
	if len(failTasks) > 0 {
		var failContent string
		for _, t := range failTasks {
			failContent += fmt.Sprintf("  %s %s\n", m.styles.ResultFail.Render("✗"), m.styles.NameStyle.Render(t.Name))
			if t.Error != "" {
				failContent += fmt.Sprintf("    %s\n", m.styles.ResultDim.Render(t.Error))
			}
		}
		failContent = strings.TrimRight(failContent, "\n")
		s += m.styles.FailBox.Render(failContent) + "\n\n"
	}

	// Successes — compact list.
	if len(okTasks) > 0 {
		var okContent string
		for _, t := range okTasks {
			line := fmt.Sprintf("  %s %s", m.styles.ResultOk.Render("✔"), m.styles.NameStyle.Render(t.Name))
			if t.Result != "" {
				line += "  " + m.styles.ResultDim.Render(t.Result)
			}
			okContent += line + "\n"
		}
		okContent = strings.TrimRight(okContent, "\n")
		s += m.styles.SuccessBox.Render(okContent) + "\n"
	}

	return s
}

func (m RunnerModel) hasFailures() bool {
	for _, t := range m.tasks {
		if t.Status == TaskFailed {
			return true
		}
	}
	return false
}

// MarkRunning marks a task as running by its path.
func (m *RunnerModel) MarkRunning(dir string) {
	for i := range m.tasks {
		if m.tasks[i].Path == dir {
			m.tasks[i].Status = TaskRunning
			break
		}
	}
}

// ── Messages ────────────────────────────────────────────────────────────

// MavenDoneMsg signals one repo's maven build completed.
type MavenDoneMsg struct {
	Path   string
	Result string
	Err    error
}

// TextOutputMsg carries pre-formatted scan/transform output.
// AltContent provides an alternate view that can be toggled with tab.
type TextOutputMsg struct {
	Lines      []string
	AltContent string // optional alternate view (toggled with tab)
}

// RunMaven returns a tea.Cmd that executes mvn in the given directory.
func RunMaven(command maven.Command, dir string) tea.Cmd {
	return func() tea.Msg {
		if errMsg := runMvn(command.Args, dir); errMsg != "" {
			return MavenDoneMsg{Path: dir, Err: fmt.Errorf("%s", errMsg)}
		}
		return MavenDoneMsg{Path: dir}
	}
}

// RunUpdateSpec runs the OpenAPI spec test, copies the generated spec back, then runs again.
// Skips repos whose own spec is already up to date (version matches pom.xml).
func RunUpdateSpec(command maven.Command, dir string) tea.Cmd {
	return func() tea.Msg {
		// 0. Quick check: if own spec version already matches POM version, skip.
		if ops.OwnSpecCurrent(dir) {
			return MavenDoneMsg{Path: dir, Result: "spec already current"}
		}

		// 1. Parse spec paths from the test file.
		// Repos without writeString or classpath resource can't be auto-updated
		// (compare-only test pattern) — report as skip, not failure.
		paths, err := ops.ParseSpecPaths(dir)
		if err != nil {
			return MavenDoneMsg{Path: dir, Result: "skip: " + err.Error()}
		}

		// 2. First run — generate the spec (ignore test failure).
		runMvn(command.Args, dir)

		// 3. Copy generated spec back to resources.
		if err := ops.CopySpec(dir, paths); err != nil {
			return MavenDoneMsg{Path: dir, Err: err}
		}

		// 4. Second run — should pass now.
		errMsg := runMvn(command.Args, dir)
		if errMsg != "" {
			return MavenDoneMsg{Path: dir, Err: fmt.Errorf("%s", errMsg)}
		}

		return MavenDoneMsg{Path: dir}
	}
}

// mvnTimeout is the max duration a single mvn invocation may run before being killed.
const mvnTimeout = 10 * time.Minute

// runMvn executes mvn with args in dir and returns the first meaningful error (or "").
// Uses buffered output instead of pipes to avoid hangs from child processes holding FDs open.
func runMvn(args []string, dir string) string {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)

	// -B = batch mode: no interactive prompts, no ANSI download progress.
	argsCopy = append([]string{"-B"}, argsCopy...)

	ctx, cancel := context.WithTimeout(context.Background(), mvnTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "mvn", argsCopy...)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()

	// Always write log file for debugging.
	writeLog(filepath.Base(dir), stdoutBuf.String(), stderrBuf.String())

	if runErr != nil {
		// Parse first meaningful [ERROR] line from stdout.
		for _, line := range strings.Split(stdoutBuf.String(), "\n") {
			if strings.Contains(line, "[ERROR]") {
				cleaned := strings.TrimSpace(strings.TrimPrefix(line, "[ERROR]"))
				if cleaned == "" || strings.HasPrefix(cleaned, "[Help") || strings.HasPrefix(cleaned, "-> [Help") || strings.HasPrefix(cleaned, "http") {
					continue
				}
				return cleaned
			}
		}
		if s := strings.TrimSpace(stderrBuf.String()); s != "" {
			return s
		}
		return runErr.Error()
	}
	return ""
}

// writeLog dumps maven output to ~/.raclette/logs/{repo}.log for debugging.
func writeLog(repo, stdout, stderr string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(home, ".cache", "raclette", "logs")
	os.MkdirAll(logDir, 0755)

	var buf strings.Builder
	buf.WriteString("=== STDOUT ===\n")
	buf.WriteString(stdout)
	if stderr != "" {
		buf.WriteString("\n=== STDERR ===\n")
		buf.WriteString(stderr)
	}
	os.WriteFile(filepath.Join(logDir, repo+".log"), []byte(buf.String()), 0644)
}

