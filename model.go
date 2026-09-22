package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	// "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type task struct {
	id          string
	title       string
	done        bool
	priority    int
	tags        []string
	description string
}

func (t *task) generateID() {
	// Nanosecond timestamp hex-encoded (~16 chars)
	ts := time.Now().UnixNano()

	// Crypto-random string prefix/suffix
	b := make([]byte, 4)
	rand.Read(b)
	t.id = fmt.Sprintf("%d-%x", ts, b)
}

func (t task) save(dirPath string) error {
	if t.id == "" {
		t.generateID()
	}
	taskDir := path.Join(dirPath, t.id)

	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}

	filePath := path.Join(taskDir, "TASK.md")

	fd, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fd.Close()

	status := "OPEN"
	if t.done {
		status = "CLOSE"
	}
	tags := strings.Join(t.tags, " ,")

	_, err = fmt.Fprintf(fd, "# %s\n\n- STATUS: %s\n- PRIORITY: %d\n- TAGS: %s\n\n~ %s", t.title, status, t.priority, tags, t.description)
	return err
}

type model struct {
	all      []*task // full unfiltered set
	filtered []*task // what's currently shown
	cursor   int

	mode      appState // stateList, stateFilterName, stateFilterTag, stateDetail
	filterVal string   // current text for whichever filter is active

	viewHeight int // how many rows to show at once (optional windowing)
}

type appState int

const (
	stateList appState = iota
	stateFilterName
	stateFilterTag
	stateDetail
)

func newModel(tasks []*task) model {
	return model{
		all:      tasks,
		filtered: tasks,
		mode:     stateList,
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editFinishedMsg:
		if msg.err == nil && msg.t != nil {
			for i, t := range m.all {
				if t.id == msg.t.id {
					*m.all[i] = *msg.t
					break
				}
			}
		}
	case tea.KeyMsg:
		switch m.mode {
		case stateDetail:
			switch msg.String() {
			case "esc", "enter", "q":
				m.mode = stateList
			case "e", "o":
				return m, editTaskCmd(m.filtered[m.cursor])
			}
			return m, nil

		case stateFilterName, stateFilterTag:
			switch msg.String() {
			case "esc":
				m.mode = stateList
				m.filterVal = ""
				m.filtered = m.all
				m.cursor = 0
				return m, nil
			case "enter":
				m.mode = stateList
				return m, nil
			case "backspace":
				if len(m.filterVal) > 0 {
					m.filterVal = m.filterVal[:len(m.filterVal)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.filterVal += msg.String()
				}
			}

			if m.mode == stateFilterName {
				m.filtered = filterByNameDesc(m.all, m.filterVal)
			} else {
				m.filtered = filterByTag(m.all, m.filterVal)
			}
			m.cursor = 0
			return m, nil

		case stateList:
			switch msg.String() {
			case "q", "ctrl+c":
				// saveAll(m.all)
				return m, tea.Quit
			case "/":
				m.mode = stateFilterName
				m.filterVal = ""
				return m, nil
			case "@":
				m.mode = stateFilterTag
				m.filterVal = ""
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.filtered) > 0 {
					m.mode = stateDetail
				}
			case " ":
				m.filtered[m.cursor].done = !m.filtered[m.cursor].done
				return m, nil
			case "e", "o":
				return m, editTaskCmd(m.filtered[m.cursor])
			}
		}
	}
	return m, nil
}

func renderDetail(t *task) string {
	if t == nil {
		return "no task selected\n"
	}
	status := openStyle.Render("OPEN")
	if t.done {
		status = closeStyle.Render("CLOSE")
	}
	for i := range t.tags {
		t.tags[i] = tagStyle.Render(t.tags[i])
	}
	tags := strings.Join(t.tags, ", ")

	metaStyle := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", titleStyle.Render(t.title))
	fmt.Fprintf(&b, "%s\n", metaStyle.Render(fmt.Sprintf("id: %s", t.id)))
	fmt.Fprintf(&b, "status: %s\n", status)
	priority := priorityStyle(t.priority).Render(fmt.Sprint(t.priority))
	fmt.Fprintf(&b, "priority: %s\n", priority)
	fmt.Fprintf(&b, "tags: %s\n\n", tags)
	fmt.Fprintf(&b, "%s\n\n", selectedStyle.Render(t.description))
	fmt.Fprintf(&b, "%s\n", metaStyle.Render("(esc/enter: back)"))

	return lipgloss.NewStyle().Margin(1, 2).Render(b.String())
}

var (
	openStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green
	closeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray
	tagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // blue
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	helpStyle     = lipgloss.NewStyle().Faint(true)
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

func priorityStyle(p int) lipgloss.Style {
	switch {
	case p >= 80:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red
	case p >= 50:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("220")) // yellow
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")) // green
	}
}

func (m model) View() string {
	switch m.mode {
	case stateDetail:
		return renderDetail(m.filtered[m.cursor])
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", headerStyle.Render("Tasks"))

	for i, t := range m.filtered {
		status := openStyle.Render("OPEN")
		if t.done {
			status = closeStyle.Render("CLOSE")
		}

		tags := ""
		if len(t.tags) > 0 {
			tags = tagStyle.Render(strings.Join(t.tags, ","))
		}

		prio := priorityStyle(t.priority).Render(fmt.Sprintf("%-3d", t.priority))
		title := titleStyle.Render(fmt.Sprintf("%-30s", t.title))
		id := helpStyle.Render(fmt.Sprintf("%-8s", shortID(t.id)))

		line := fmt.Sprintf("%s [%-14s] %s  %s  %s", id, status, prio, title, tags)

		if i == m.cursor {
			fmt.Fprintf(&b, "%s%s\n", cursorStyle.Render("> "), selectedStyle.Render(line))
		} else {
			fmt.Fprintf(&b, " %s\n", line)
		}
	}

	switch m.mode {
	case stateFilterName:
		fmt.Fprintf(&b, "\n%s", filterStyle.Render("/"+m.filterVal))
	case stateFilterTag:
		fmt.Fprintf(&b, "\n%s", filterStyle.Render("@"+m.filterVal))
	default:
		fmt.Fprintf(&b, "\n%s", helpStyle.Render("(/: name+desc  @: tag  enter: view,  e: edit  space: status  q: quit)"))
	}

	return b.String()
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func filterByNameDesc(tasks []*task, q string) []*task {
	if q == "" {
		return tasks
	}
	q = strings.ToLower(q)
	var out []*task
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.title), q) ||
			strings.Contains(strings.ToLower(t.description), q) {
			out = append(out, t)
		}
	}
	return out
}

func filterByTag(tasks []*task, q string) []*task {
	if q == "" {
		return tasks
	}
	q = strings.ToLower(q)
	var out []*task
	for _, t := range tasks {
		for _, tag := range t.tags {
			if strings.Contains(strings.ToLower(tag), q) {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// --- loading + entrypoint ---

func startTUI() {
	rootPath, err := getRootDir()
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	var tasks []*task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := loadTask(rootPath, entry.Name())
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].priority > tasks[j].priority })

	p := tea.NewProgram(newModel(tasks))
	if _, err := p.Run(); err != nil {
		fmt.Println("error running tui:", err)
		os.Exit(1)
	}
}

type editFinishedMsg struct {
	t   *task
	err error
}

func editTaskCmd(t *task) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	rootPath, err := getRootDir()
	if err != nil {
		return func() tea.Msg { return editFinishedMsg{t: t, err: err} }
	}
	filePath := path.Join(rootPath, t.id, "TASK.md")

	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return editFinishedMsg{t: t, err: err}
		}
		reloaded, err := loadTask(rootPath, t.id)
		return editFinishedMsg{t: reloaded, err: err}
	})
}
