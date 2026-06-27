package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"TidyFS/carrier"
	"TidyFS/classifier_runner"
	"TidyFS/exporter"
	projectpath "TidyFS/project_path"
	"TidyFS/scanner"
)

type Mode string

type ClassifierMode string

const (
	ClassifierTFIDF ClassifierMode = "tf_idf"
	ClassifierLLM   ClassifierMode = "llm"
)

const (
	ModeFuse Mode = "fuse"
	ModeMove Mode = "move"
	ModeCopy Mode = "copy"
)

type Screen int

const (
	ScreenForm Screen = iota
	ScreenPipeline
	ScreenDone
)

type StageStatus int

const (
	StagePending StageStatus = iota
	StageRunning
	StageDone
	StageFailed
)

type Result struct {
	Source         string
	Target         string
	Mode           Mode
	CleanTarget    bool
	ClassifierMode ClassifierMode
}

type Stage struct {
	Title       string
	Description string
	Status      StageStatus
}

type Model struct {
	inputs         []textinput.Model
	focus          int
	mode           Mode
	cleanTarget    bool
	classifierMode ClassifierMode

	width  int
	height int

	screen Screen
	result Result

	stages       []Stage
	currentStage int
	err          error

	pulse bool

	ctx    context.Context
	cancel context.CancelFunc
}

const cardWidth = 76
const inputWidth = 54
const maxFocus = 4

var (
	colorText    = lipgloss.Color("#e7e9ee")
	colorMuted   = lipgloss.Color("#8b90a0")
	colorDim     = lipgloss.Color("#4b5162")
	colorAccent  = lipgloss.Color("#8b5cf6")
	colorAccent2 = lipgloss.Color("#22d3ee")
	colorGreen   = lipgloss.Color("#34d399")
	colorRed     = lipgloss.Color("#fb7185")
	colorOrange  = lipgloss.Color("#fb923c")
	colorYellow  = lipgloss.Color("#facc15")

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	inputBoxStyle = lipgloss.NewStyle().
			Width(inputWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(0, 1)

	inputBoxFocusedStyle = inputBoxStyle.Copy().
				BorderForeground(colorAccent2)

	modeStyle = lipgloss.NewStyle().
			Padding(0, 3).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Foreground(colorMuted)

	modeFocusedStyle = modeStyle.Copy().
				BorderForeground(colorAccent2).
				Foreground(colorText)

	modeActiveStyle = modeStyle.Copy().
			Background(colorAccent).
			Foreground(lipgloss.Color("#ffffff")).
			BorderForeground(colorAccent).
			Bold(true)

	modeActiveFocusedStyle = modeActiveStyle.Copy().
				BorderForeground(colorAccent2)

	contentStyle = lipgloss.NewStyle().
			Width(cardWidth-8).
			Padding(1, 4, 1, 4)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)
)

type stageFinishedMsg struct {
	index int
	err   error
}

type pulseMsg struct{}

func New() Model {
	ctx, cancel := context.WithCancel(context.Background())

	source := textinput.New()
	source.Placeholder = "~/Downloads"
	source.Width = inputWidth - 6
	source.Prompt = "❯ "
	source.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent2)
	source.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	source.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorDim)
	source.Focus()

	target := textinput.New()
	target.Placeholder = "~/TidyFS"
	target.Width = inputWidth - 6
	target.Prompt = "❯ "
	target.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent2)
	target.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	target.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorDim)

	return Model{
		inputs:         []textinput.Model{source, target},
		mode:           ModeFuse,
		cleanTarget:    false,
		classifierMode: ClassifierTFIDF,
		screen:         ScreenForm,
		stages:         defaultStages(ModeFuse, ClassifierTFIDF),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func Run() (Result, error) {
	finalModel, err := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	).Run()

	if err != nil {
		return Result{}, err
	}

	m, ok := finalModel.(Model)
	if !ok {
		return Result{}, fmt.Errorf("unexpected TUI model")
	}

	if m.err != nil {
		return m.result, m.err
	}

	return m.result, nil
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func pulseTick() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg {
		return pulseMsg{}
	})
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}

	return m, tea.Quit
}

func (m *Model) resetToForm() {
	m.screen = ScreenForm
	m.result = Result{}
	m.err = nil
	m.currentStage = 0
	m.pulse = false
	m.stages = defaultStages(m.mode, m.classifierMode)
	m.focus = 0
	m.updateFocus()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		}

		switch m.screen {
		case ScreenForm:
			switch msg.String() {
			case "esc":
				return m.quit()
			}

			var cmd tea.Cmd
			m, cmd = m.updateFormKey(msg)

			var inputCmds []tea.Cmd

			for i := range m.inputs {
				var inputCmd tea.Cmd
				m.inputs[i], inputCmd = m.inputs[i].Update(msg)
				inputCmds = append(inputCmds, inputCmd)
			}

			return m, tea.Batch(append(inputCmds, cmd)...)

		case ScreenPipeline:
			switch msg.String() {
			case "esc":
				return m.quit()
			}

			return m, nil

		case ScreenDone:
			switch msg.String() {
			case "esc", "enter":
				m.resetToForm()
				return m, nil

			case "q":
				return m.quit()
			}
		}

	case stageFinishedMsg:
		return m.updateStage(msg)

	case pulseMsg:
		if m.screen != ScreenPipeline {
			return m, nil
		}

		m.pulse = !m.pulse
		return m, pulseTick()
	}

	if m.screen != ScreenForm {
		return m, nil
	}

	var cmds []tea.Cmd

	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateFormKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		m.focus++
		if m.focus > maxFocus {
			m.focus = 0
		}
		m.updateFocus()
		return m, nil

	case "shift+tab", "up":
		m.focus--
		if m.focus < 0 {
			m.focus = maxFocus
		}
		m.updateFocus()
		return m, nil

	case "left", "right", " ":
		if m.focus == 2 {
			m.toggleMode()
			m.stages = defaultStages(m.mode, m.classifierMode)
			return m, nil
		}

		if m.focus == 3 {
			m.cleanTarget = !m.cleanTarget
			return m, nil
		}

		if m.focus == 4 {
			m.toggleClassifierMode()
			m.stages = defaultStages(m.mode, m.classifierMode)
			return m, nil
		}

	case "enter":
		if m.focus < maxFocus {
			m.focus++
			m.updateFocus()
			return m, nil
		}

		m.result = Result{
			Source:         strings.TrimSpace(m.inputs[0].Value()),
			Target:         strings.TrimSpace(m.inputs[1].Value()),
			Mode:           m.mode,
			CleanTarget:    m.cleanTarget,
			ClassifierMode: m.classifierMode,
		}

		m.screen = ScreenPipeline
		m.currentStage = 0
		m.stages = defaultStages(m.mode, m.classifierMode)
		m.stages[0].Status = StageRunning
		m.pulse = false

		return m, tea.Batch(
			runStage(m.ctx, 0, m.result),
			pulseTick(),
		)
	}

	return m, nil
}

func (m Model) updateStage(msg stageFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.index < 0 || msg.index >= len(m.stages) {
		return m, nil
	}

	if msg.err != nil {
		m.stages[msg.index].Status = StageFailed
		m.err = msg.err
		m.screen = ScreenDone
		return m, nil
	}

	m.stages[msg.index].Status = StageDone

	next := msg.index + 1
	if next >= len(m.stages) {
		m.screen = ScreenDone
		return m, nil
	}

	m.currentStage = next
	m.stages[next].Status = StageRunning

	return m, runStage(m.ctx, next, m.result)
}

func runStage(ctx context.Context, index int, result Result) tea.Cmd {
	return func() tea.Msg {
		var err error

		switch index {
		case 0:
			err = scanFoldersAndFiles(result.Source)

		case 1:
			err = classifyAndDistribute(result.ClassifierMode)

		case 2:
			if result.Mode == ModeFuse {
				err = runFuse(ctx, result.Target)
			} else {
				if result.CleanTarget {
					err = cleanTargetDirectory(result.Source, result.Target)
				}

				if err == nil && result.Mode == ModeMove {
					err = moveFiles(result.Source, result.Target)
				}

				if err == nil && result.Mode == ModeCopy {
					err = copyFiles(result.Source, result.Target)
				}
			}

		case 3:
			err = finishPipeline()
		}

		return stageFinishedMsg{
			index: index,
			err:   err,
		}
	}
}

func scanFoldersAndFiles(source string) error {
	scn := scanner.NewScanner()

	files, err := scn.ScanDirs(source)
	if err != nil {
		return err
	}

	return exporter.SaveJSON(projectpath.FilesJSON(), files)
}

func classifyAndDistribute(mode ClassifierMode) error {
	return classifier_runner.RunClassifier(projectpath.Root(), string(mode))
}

func runFuse(ctx context.Context, target string) error {
	target, err := scanner.ExpandPath(target)
	if err != nil {
		return err
	}

	return carrier.FuseRun(ctx, target)
}

func moveFiles(source string, target string) error {
	return carrier.FileSystemCarrierRun(source, target, "move")
}

func copyFiles(source string, target string) error {
	return carrier.FileSystemCarrierRun(source, target, "copy")
}

func cleanTargetDirectory(source string, target string) error {
	sourcePath, err := scanner.ExpandPath(source)
	if err != nil {
		return err
	}

	targetPath, err := expandPathLoose(target)
	if err != nil {
		return err
	}

	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	sourcePath = filepath.Clean(sourcePath)
	targetPath = filepath.Clean(targetPath)

	if targetPath == string(filepath.Separator) {
		return fmt.Errorf("refusing to clean root directory")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeDir, _ = filepath.Abs(homeDir)
		homeDir = filepath.Clean(homeDir)

		if targetPath == homeDir {
			return fmt.Errorf("refusing to clean home directory")
		}
	}

	if sourcePath == targetPath {
		return fmt.Errorf("refusing to clean target: source and target are the same directory")
	}

	if pathInside(sourcePath, targetPath) {
		return fmt.Errorf("refusing to clean target: source directory is inside target directory")
	}

	if err := os.RemoveAll(targetPath); err != nil {
		return err
	}

	return os.MkdirAll(targetPath, 0755)
}

func expandPathLoose(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		return filepath.Join(homeDir, strings.TrimPrefix(path, "~/")), nil
	}

	return path, nil
}

func pathInside(child string, parent string) bool {
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)

	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func finishPipeline() error {
	time.Sleep(500 * time.Millisecond)
	return nil
}

func defaultStages(mode Mode, classifierMode ClassifierMode) []Stage {
	actionTitle := "Start FUSE"
	actionDesc := "Mounting virtual organized filesystem view."

	if mode == ModeMove {
		actionTitle = "Move Files"
		actionDesc = "Moving files into sorted target folders."
	}

	if mode == ModeCopy {
		actionTitle = "Copy Files"
		actionDesc = "Copying files into sorted target folders."
	}

	classifierDesc := "Classifying documents using fast TF_IDF classifier."

	if classifierMode == ClassifierLLM {
		classifierDesc = "Classifying documents using slower LLM classifier."
	}

	return []Stage{
		{
			Title:       "Scan Folders/Files",
			Description: "Scanning source folder and collecting document metadata.",
			Status:      StagePending,
		},
		{
			Title:       "Classify / Distribute Files",
			Description: classifierDesc,
			Status:      StagePending,
		},
		{
			Title:       actionTitle,
			Description: actionDesc,
			Status:      StagePending,
		},
		{
			Title:       "Ready",
			Description: "TidyFS pipeline finished successfully.",
			Status:      StagePending,
		},
	}
}

func (m *Model) updateFocus() {
	for i := range m.inputs {
		if i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m *Model) toggleMode() {
	switch m.mode {
	case ModeFuse:
		m.mode = ModeMove
	case ModeMove:
		m.mode = ModeCopy
	default:
		m.mode = ModeFuse
	}
}

func (m *Model) toggleClassifierMode() {
	switch m.classifierMode {
	case ClassifierTFIDF:
		m.classifierMode = ClassifierLLM
	default:
		m.classifierMode = ClassifierTFIDF
	}
}

func (m Model) View() string {
	var body string

	switch m.screen {
	case ScreenForm:
		body = m.viewForm()

	case ScreenPipeline:
		body = m.viewPipeline()

	case ScreenDone:
		body = m.viewDone()
	}

	card := renderCard("TidyFS", contentStyle.Render(body))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		card,
	)
}

func (m Model) viewForm() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		renderHeader(),
		"",
		m.renderInput(0, "Source folder"),
		"",
		m.renderInput(1, "Target folder"),
		"",
		m.renderModePicker(),
		"",
		m.renderCleanTargetPicker(),
		"",
		m.renderClassifierModePicker(),
		"",
		helpStyle.Render("tab/up/down navigate  •  left/right/space toggle  •"),
		helpStyle.Render("enter start pipeline  •  esc quit"),
	)
}

func (m Model) viewPipeline() string {
	summary := lipgloss.JoinVertical(
		lipgloss.Left,
		subtitleStyle.Render("Pipeline is running"),
		descStyle.Render("TidyFS is processing your files. UI will stay open."),
		"",
		descStyle.Render("Source:     "+emptyFallback(m.result.Source, "not selected")),
		descStyle.Render("Target:     "+emptyFallback(m.result.Target, "not selected")),
		descStyle.Render("Mode:       "+string(m.result.Mode)),
		descStyle.Render("Clean:      "+boolLabel(m.result.CleanTarget)),
		descStyle.Render("Classifier: "+classifierLabel(m.result.ClassifierMode)),
	)

	var stageViews []string

	for i, stage := range m.stages {
		stageViews = append(stageViews, renderStage(i, stage, m.pulse))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		summary,
		"",
		strings.Join(stageViews, "\n"),
		"",
		helpStyle.Render("esc back  •  ctrl+c quit"),
	)
}

func (m Model) viewDone() string {
	title := successStyle.Render("Pipeline finished")

	if m.err != nil {
		title = errorStyle.Render("Pipeline failed")
	}

	var message string

	if m.err != nil {
		message = errorStyle.Render(m.err.Error())
	} else if m.result.Mode == ModeFuse {
		message = descStyle.Render("FUSE is mounted. Press q/ctrl+c to unmount and exit.")
	} else {
		message = descStyle.Render("All stages completed successfully.")
	}

	var stageViews []string

	for i, stage := range m.stages {
		stageViews = append(stageViews, renderStage(i, stage, false))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		message,
		"",
		descStyle.Render("Source:     "+emptyFallback(m.result.Source, "not selected")),
		descStyle.Render("Target:     "+emptyFallback(m.result.Target, "not selected")),
		descStyle.Render("Mode:       "+string(m.result.Mode)),
		descStyle.Render("Clean:      "+boolLabel(m.result.CleanTarget)),
		descStyle.Render("Classifier: "+classifierLabel(m.result.ClassifierMode)),
		"",
		strings.Join(stageViews, "\n"),
		"",
		helpStyle.Render("enter/esc back to start  •  q quit  •  ctrl+c graceful shutdown"),
	)
}

func renderHeader() string {
	logo := lipgloss.NewStyle().
		Foreground(colorAccent2).
		Bold(true).
		Render("✦ TidyFS")

	title := subtitleStyle.Render("Smart file organizer")
	desc := descStyle.Render("Scan, classify and prepare files for safe organization.")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		logo,
		title,
		desc,
	)
}

func (m Model) renderInput(index int, label string) string {
	box := inputBoxStyle

	if m.focus == index {
		box = inputBoxFocusedStyle
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render(label),
		box.Render(m.inputs[index].View()),
	)
}

func (m Model) renderModePicker() string {
	label := labelStyle.Render("Action mode")

	fuseStyle := modeStyle
	moveStyle := modeStyle
	copyStyle := modeStyle

	if m.focus == 2 {
		fuseStyle = modeFocusedStyle
		moveStyle = modeFocusedStyle
		copyStyle = modeFocusedStyle
	}

	if m.mode == ModeFuse {
		if m.focus == 2 {
			fuseStyle = modeActiveFocusedStyle
		} else {
			fuseStyle = modeActiveStyle
		}
	}

	if m.mode == ModeMove {
		if m.focus == 2 {
			moveStyle = modeActiveFocusedStyle
		} else {
			moveStyle = modeActiveStyle
		}
	}

	if m.mode == ModeCopy {
		if m.focus == 2 {
			copyStyle = modeActiveFocusedStyle
		} else {
			copyStyle = modeActiveStyle
		}
	}

	fuse := fuseStyle.Render("fuse")
	move := moveStyle.Render("move")
	copy := copyStyle.Render("copy")

	hint := descStyle.Render("fuse: virtual view  •  move: move files  •  copy: copy files")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		label,
		lipgloss.JoinHorizontal(lipgloss.Top, fuse, "  ", move, "  ", copy),
		hint,
	)
}

func (m Model) renderCleanTargetPicker() string {
	label := labelStyle.Render("Clean target before work")

	offStyle := modeStyle
	onStyle := modeStyle

	if m.focus == 3 {
		offStyle = modeFocusedStyle
		onStyle = modeFocusedStyle
	}

	if !m.cleanTarget {
		if m.focus == 3 {
			offStyle = modeActiveFocusedStyle
		} else {
			offStyle = modeActiveStyle
		}
	}

	if m.cleanTarget {
		if m.focus == 3 {
			onStyle = modeActiveFocusedStyle
		} else {
			onStyle = modeActiveStyle
		}
	}

	off := offStyle.Render("no")
	on := onStyle.Render("yes")

	hint := descStyle.Render("yes: delete everything inside target before move/copy")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		label,
		lipgloss.JoinHorizontal(lipgloss.Top, off, "  ", on),
		hint,
	)
}

func (m Model) renderClassifierModePicker() string {
	label := labelStyle.Render("Classifier")

	fastStyle := modeStyle
	slowStyle := modeStyle

	if m.focus == 4 {
		fastStyle = modeFocusedStyle
		slowStyle = modeFocusedStyle
	}

	if m.classifierMode == ClassifierTFIDF {
		if m.focus == 4 {
			fastStyle = modeActiveFocusedStyle
		} else {
			fastStyle = modeActiveStyle
		}
	}

	if m.classifierMode == ClassifierLLM {
		if m.focus == 4 {
			slowStyle = modeActiveFocusedStyle
		} else {
			slowStyle = modeActiveStyle
		}
	}

	fast := fastStyle.Render("TF_IDF")
	slow := slowStyle.Render("LLM")

	hint := descStyle.Render("TF_IDF: fast local classifier  •  LLM: slower but smarter classification")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		label,
		lipgloss.JoinHorizontal(lipgloss.Top, fast, "  ", slow),
		hint,
	)
}

func renderStage(index int, stage Stage, pulse bool) string {
	icon := "○"
	iconStyle := lipgloss.NewStyle().Foreground(colorDim)
	title := lipgloss.NewStyle().Foreground(colorMuted).Render(stage.Title)

	switch stage.Status {
	case StagePending:
		icon = "○"
		iconStyle = lipgloss.NewStyle().Foreground(colorDim)
		title = lipgloss.NewStyle().Foreground(colorMuted).Render(stage.Title)

	case StageRunning:
		icon = "●"

		runningColor := colorOrange
		if pulse {
			runningColor = colorYellow
		}

		iconStyle = lipgloss.NewStyle().Foreground(runningColor)
		title = lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(stage.Title + "…")

	case StageDone:
		icon = "✓"
		iconStyle = lipgloss.NewStyle().Foreground(colorGreen)
		title = lipgloss.NewStyle().Foreground(colorText).Render(stage.Title)

	case StageFailed:
		icon = "✕"
		iconStyle = lipgloss.NewStyle().Foreground(colorRed)
		title = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render(stage.Title)
	}

	number := lipgloss.NewStyle().
		Foreground(colorDim).
		Render(fmt.Sprintf("%d.", index+1))

	description := descStyle.Render("   " + stage.Description)

	line := lipgloss.JoinHorizontal(
		lipgloss.Top,
		number,
		" ",
		iconStyle.Render(icon),
		" ",
		title,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		line,
		description,
	)
}

func renderCard(title string, body string) string {
	borderColor := colorAccent

	titleRendered := titleStyle.Render(" " + title + " ")
	titleWidth := lipgloss.Width(titleRendered)

	innerWidth := cardWidth - 2

	leftLine := strings.Repeat("─", 4)
	rightLine := strings.Repeat("─", max(0, innerWidth-titleWidth-4))

	top := lipgloss.NewStyle().
		Foreground(borderColor).
		Render("╭"+leftLine) +
		titleRendered +
		lipgloss.NewStyle().
			Foreground(borderColor).
			Render(rightLine+"╮")

	bodyLines := strings.Split(body, "\n")
	renderedLines := make([]string, 0, len(bodyLines))

	for _, line := range bodyLines {
		lineWidth := lipgloss.Width(line)
		padding := max(0, innerWidth-lineWidth)

		renderedLines = append(
			renderedLines,
			lipgloss.NewStyle().Foreground(borderColor).Render("│")+
				line+
				strings.Repeat(" ", padding)+
				lipgloss.NewStyle().Foreground(borderColor).Render("│"),
		)
	}

	bottom := lipgloss.NewStyle().
		Foreground(borderColor).
		Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		top,
		strings.Join(renderedLines, "\n"),
		bottom,
	)
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func boolLabel(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func classifierLabel(value ClassifierMode) string {
	switch value {
	case ClassifierLLM:
		return "LLM"
	default:
		return "TF_IDF"
	}
}
