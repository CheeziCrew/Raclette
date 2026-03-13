package tui

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/CheeziCrew/raclette/internal/maven"
	"github.com/CheeziCrew/raclette/internal/ops"
	"github.com/CheeziCrew/raclette/internal/tui/screens"
	"github.com/CheeziCrew/curd"
)

const maxParallel = 3

// Model is the root app model — routes between screens.
type Model struct {
	current    screens.Screen
	menu       screens.MenuModel
	runner     screens.RunnerModel
	repoSelect screens.RepoSelectModel
	prompt     screens.PromptModel

	// Stash the selected command and collected inputs.
	pendingCmd    maven.Command
	pendingInputs map[string]string

	// Multi-repo queue.
	queue     []string
	running   int
	completed int
	total     int

	history *History
	width   int
	height  int
}

// New creates the root model.
func New() Model {
	return Model{
		current: screens.ScreenMenu,
		menu:    screens.NewMenu(),
		runner:  screens.NewRunner(),
		history: LoadHistory(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.menu.Init(),
		m.runner.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu, _ = m.menu.Update(msg)
		m.runner, _ = m.runner.Update(msg)
		m.repoSelect, _ = m.repoSelect.Update(msg)
		m.prompt, _ = m.prompt.Update(msg)
		return m, nil

	case tea.KeyPressMsg:
		if cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
		}

	case curd.MenuSelectionMsg:
		return m.handleMenuSelection(msg)

	case screens.RepoSelectDoneMsg:
		return m.handleRepoSelectDone()

	case screens.NavigateMsg:
		m.current = msg.Screen
		return m, nil

	case screens.PromptDoneMsg:
		m.pendingInputs = msg.Inputs
		return m, m.execute()

	case screens.BackToMenuMsg:
		m.current = screens.ScreenMenu
		return m, nil

	case screens.SaveHistoryMsg:
		m.history.Add(msg.Category, msg.Value)
		return m, nil

	case screens.MavenDoneMsg:
		return m.handleMavenDone(msg)
	}

	return m.delegateToScreen(msg)
}

func (m *Model) handleGlobalKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if msg.String() == "ctrl+c" {
		return tea.Quit, true
	}
	if msg.String() == "q" && m.current != screens.ScreenPrompt {
		if m.current == screens.ScreenMenu {
			return tea.Quit, true
		}
		m.current = screens.ScreenMenu
		return nil, true
	}
	return nil, false
}

func (m Model) handleMenuSelection(msg curd.MenuSelectionMsg) (tea.Model, tea.Cmd) {
	for _, cmd := range maven.Commands() {
		if cmd.Name == msg.Command {
			m.pendingCmd = cmd
			m.pendingInputs = nil
			base, _ := os.Getwd()
			m.repoSelect = screens.NewRepoSelect(base, m.height, 3)
			m.current = screens.ScreenRepoSelect
			return m, m.repoSelect.Init()
		}
	}
	return m, nil
}

func (m Model) handleRepoSelectDone() (tea.Model, tea.Cmd) {
	if len(m.pendingCmd.Prompts) > 0 && m.pendingInputs == nil {
		m.prompt = screens.NewPromptWithHistory(m.pendingCmd, m.history.Messages)
		m.current = screens.ScreenPrompt
		return m, m.prompt.Init()
	}
	return m, m.execute()
}

func (m Model) handleMavenDone(msg screens.MavenDoneMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.runner, cmd = m.runner.Update(msg)
	cmds = append(cmds, cmd)
	m.running--
	m.completed++
	cmds = append(cmds, m.drainQueue()...)
	return m, tea.Batch(cmds...)
}

func (m Model) delegateToScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.current {
	case screens.ScreenMenu:
		m.menu, cmd = m.menu.Update(msg)
	case screens.ScreenRunner:
		m.runner, cmd = m.runner.Update(msg)
	case screens.ScreenRepoSelect:
		m.repoSelect, cmd = m.repoSelect.Update(msg)
	case screens.ScreenPrompt:
		m.prompt, cmd = m.prompt.Update(msg)
	}
	return m, cmd
}

// execute runs the pending command based on its kind.
func (m *Model) execute() tea.Cmd {
	dirs := m.repoSelect.SelectedPaths()

	switch m.pendingCmd.Kind {
	case maven.KindMaven, maven.KindUpdateSpec:
		return m.startMultiRepo(dirs)
	case maven.KindScan:
		return m.executeScan(dirs)
	case maven.KindTransform:
		return m.executeTransform(dirs)
	}

	return nil
}

// executeScan runs a read-only scan and shows results in the runner.
func (m *Model) executeScan(dirs []string) tea.Cmd {
	m.current = screens.ScreenRunner
	m.runner.StartText(m.pendingCmd)

	var content, altContent string

	switch m.pendingCmd.Name {
	case "Find Dependency":
		query := m.pendingInputs["query"]
		matches := ops.FindDependency(dirs, query)
		content = formatDepByRepo(matches)
		altContent = formatDepByDep(matches)
	case "Find Stale Specs":
		stale := ops.FindStaleSpecs(dirs)
		content = formatStaleResults(stale)
	}

	return tea.Batch(
		m.runner.Init(),
		func() tea.Msg {
			return screens.TextOutputMsg{Lines: []string{content}, AltContent: altContent}
		},
	)
}

// executeTransform runs a file-mutating transform and shows results.
func (m *Model) executeTransform(dirs []string) tea.Cmd {
	m.current = screens.ScreenRunner
	m.runner.StartText(m.pendingCmd)

	var results []ops.Result

	switch m.pendingCmd.Name {
	case "Replace Dependency Version":
		results = ops.ReplaceDependencyVersion(
			dirs,
			m.pendingInputs["groupId"],
			m.pendingInputs["artifactId"],
			m.pendingInputs["newVersion"],
		)
	case "Swap Dependency":
		results = ops.SwapDependency(
			dirs,
			m.pendingInputs["oldGroupId"],
			m.pendingInputs["oldArtifactId"],
			m.pendingInputs["newGroupId"],
			m.pendingInputs["newArtifactId"],
			m.pendingInputs["newVersion"],
		)
	case "Bump Parent Version":
		results = ops.BumpParentVersion(dirs, m.pendingInputs["newVersion"])
	case "Update Integration Specs":
		results = ops.UpdateIntegrationSpecs(dirs)
	}

	content := formatTransformResults(results)
	altContent := ops.FormatDiffs(results)

	return tea.Batch(
		m.runner.Init(),
		func() tea.Msg {
			return screens.TextOutputMsg{Lines: []string{content}, AltContent: altContent}
		},
	)
}

// startMultiRepo initializes the queue and fires up to maxParallel maven builds.
func (m *Model) startMultiRepo(dirs []string) tea.Cmd {
	if len(dirs) == 0 {
		// Nothing to run — show empty result immediately.
		m.current = screens.ScreenRunner
		m.runner.StartText(m.pendingCmd)
		return tea.Batch(
			m.runner.Init(),
			func() tea.Msg {
				return screens.TextOutputMsg{Lines: []string{"No repos selected."}}
			},
		)
	}

	m.current = screens.ScreenRunner
	m.queue = make([]string, len(dirs))
	copy(m.queue, dirs)
	m.running = 0
	m.completed = 0
	m.total = len(dirs)
	m.runner.StartMaven(m.pendingCmd, dirs)

	var cmds []tea.Cmd
	cmds = append(cmds, m.runner.Init()) // Init already starts spinner + tick loop.
	cmds = append(cmds, m.drainQueue()...)

	return tea.Batch(cmds...)
}

// drainQueue pops dirs off the queue until we hit maxParallel running.
func (m *Model) drainQueue() []tea.Cmd {
	var cmds []tea.Cmd
	for m.running < maxParallel && len(m.queue) > 0 {
		dir := m.queue[0]
		m.queue = m.queue[1:]
		m.running++
		m.runner.MarkRunning(dir)
		switch m.pendingCmd.Kind {
		case maven.KindUpdateSpec:
			cmds = append(cmds, screens.RunUpdateSpec(m.pendingCmd, dir))
		default:
			cmds = append(cmds, screens.RunMaven(m.pendingCmd, dir))
		}
	}
	return cmds
}

func (m Model) View() tea.View {
	var content string
	switch m.current {
	case screens.ScreenMenu:
		content = m.menu.View()
	case screens.ScreenRunner:
		content = m.runner.View()
	case screens.ScreenRepoSelect:
		content = lipgloss.NewStyle().Bold(true).Foreground(Styles.Title.GetForeground()).Render(m.pendingCmd.Icon+" "+m.pendingCmd.Name) + "\n\n" + m.repoSelect.View()
	case screens.ScreenPrompt:
		content = m.prompt.View()
	}
	v := tea.NewView(lipgloss.NewStyle().Padding(1, 2, 0, 2).Render(content))
	v.AltScreen = true
	v.WindowTitle = "raclette"
	return v
}
