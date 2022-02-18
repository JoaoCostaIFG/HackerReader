package main

import (
	"encoding/json"
	"fmt"
	"hackerreader/posts"
	"hackerreader/set"
	"hackerreader/style"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/golang-collections/collections/stack"
	"github.com/pkg/browser"
	"golang.org/x/term"
)

const (
	apiurl          = "https://hacker-news.firebaseio.com/v0"
	itemUrl         = "https://news.ycombinator.com/item?id="
	loadBacklogSize = 2
	rootStoryId     = 0
	maxWidth        = 135
)

type model struct {
	loaded        bool
	toLoad        *set.Set
	stories       map[int]*posts.Post
	cursor        int
	prevCursor    *stack.Stack
	selected      *stack.Stack
	spinner       spinner.Model
	inFocus       int
	inFocusCursor int
}

func initialModel() model {
	initModel := model{
		loaded:        false,
		toLoad:        set.New(),
		stories:       make(map[int]*posts.Post),
		cursor:        0,
		prevCursor:    stack.New(),
		selected:      stack.New(),
		spinner:       spinner.New(),
		inFocus:       -1,
		inFocusCursor: 0,
	}
	// spinner
	initModel.spinner.Spinner = style.SpinnerSpinner
	initModel.spinner.Style = style.SpinnerStyle
	// add root "story" => top stories are its children
	rootSt := posts.New()
	rootSt.Id = rootStoryId
	initModel.stories[rootStoryId] = &rootSt

	initModel.selected.Push(rootStoryId)
	return initModel
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type loadTickMsg struct{}

func (m model) loadTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return loadTickMsg{}
	})
}

type topStoriesMsg struct {
	stories []int
}

func fetchTopStories() tea.Msg {
	c := &http.Client{Timeout: 10 * time.Second}
	res, err := c.Get(apiurl + "/topstories.json")

	if err != nil {
		return errMsg{err}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	var data []int
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return errMsg{err}
	}
	return topStoriesMsg{stories: data}
}

func fetchStory(item string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{Timeout: 10 * time.Second}
		res, err := c.Get(apiurl + "/item/" + item + ".json")
		if err != nil {
			return errMsg{err}
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		if err != nil {
			return errMsg{err}
		}
		return posts.FromJSON(bodyBytes)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchTopStories,
		m.spinner.Tick,
		m.loadTick(),
	)
}

// Returns the post/story and ques lazy loading if needed
func (m *model) getPost(stId int) *posts.Post {
	st, exists := m.stories[stId]
	if !exists {
		// create new post
		newSt := posts.New()
		m.stories[stId] = &newSt
		// queue for loading
		m.toLoad.Insert(stId)
		return &newSt
	}
	return st
}

func (m *model) moveCursor(newCursor int) {
	st := m.getPost(m.selected.Peek().(int))
	if newCursor < 0 {
		// allows passing -1 to go to last
		newCursor = st.KidCount() - 1
	}
	if newCursor > st.KidCount()-1 {
		newCursor = st.KidCount() - 1
	}
	// move the cursor
	m.cursor = newCursor

	// queues some children for loading
	if st.KidCount() > 0 {
		child := m.getPost(st.Kids[m.cursor])
		for i := 0; i < loadBacklogSize && i < child.KidCount(); i++ {
			grandChildId := child.Kids[i]
			m.getPost(grandChildId) // will trigger loading if needed
		}
	}
}

func (m *model) focusKeyHandler(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q": // quit
		return m, tea.Quit
	case "down", "j":
		m.inFocusCursor++
	case "up", "k":
		m.inFocusCursor = max(0, m.inFocusCursor-1)
	case "f":
		m.inFocus = -1
		m.inFocusCursor = 0
	}

	return m, nil
}

func (m *model) keyHandler(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.inFocus > 0 {
		return m.focusKeyHandler(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q": // quit
		return m, tea.Quit
	case "g", "home":
		m.moveCursor(0)
	case "G", "end":
		m.moveCursor(-1)
	case "pgup":
		m.moveCursor(m.cursor - min(10, m.cursor))
	case "pgdown":
		m.moveCursor(m.cursor + 10)
	case "down", "j": // point down
		m.moveCursor(m.cursor + 1)
	case "up", "k": // point up
		m.moveCursor(max(m.cursor-1, 0))
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		newCursor, _ := strconv.Atoi(msg.String())
		m.moveCursor(newCursor)
	case "enter", "right", "l": // go in
		parentStory := m.getPost(m.selected.Peek().(int))
		if parentStory.HasKids() {
			// can only go in if there's kids
			stId := parentStory.Kids[m.cursor]
			st := m.getPost(stId)
			if st.IsLoaded() {
				// loaded => we can go in
				m.prevCursor.Push(m.cursor) // save previous state for when we go back
				m.selected.Push(stId)
				m.cursor = 0 // go in and load kids (if needed)
			}
		}
	case "escape", "left", "h": // go back
		// recover previous state
		if m.selected.Len() > 1 {
			// we're nested (rootStory can't be popped)
			m.cursor = m.prevCursor.Pop().(int)
			m.selected.Pop()
		}
	case " ": // hide/unhide given story
		parentStory := m.getPost(m.selected.Peek().(int))
		if parentStory.HasKids() {
			st := m.getPost(parentStory.Kids[m.cursor])
			if st.IsLoaded() {
				st.ToggleHidden()
			}
		}
	case "o": // open story URL is browser
		parentStory := m.getPost(m.selected.Peek().(int))

		var targetStory *posts.Post
		if m.selected.Len() > 1 {
			targetStory = parentStory
		} else if parentStory.HasKids() {
			targetStory = m.getPost(parentStory.Kids[m.cursor])
		}

		if targetStory != nil && targetStory.HasUrl() {
			_ = browser.OpenURL(targetStory.Url)
		}
	case "O": // open story in browser
		parentStory := m.getPost(m.selected.Peek().(int))
		if parentStory.HasKids() {
			targetSt := m.getPost(parentStory.Kids[m.cursor])
			_ = browser.OpenURL(itemUrl + strconv.Itoa(targetSt.Id))
		}
	case "f":
		parent := m.getPost(m.selected.Peek().(int))
		if parent.KidCount() > 0 {
			childId := parent.Kids[m.cursor]
			m.inFocus = childId
		}
	}

	return m, nil
}

func (m *model) MouseHandler(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelDown:
		m.moveCursor(m.cursor + 1)
	case tea.MouseWheelUp:
		m.moveCursor(max(m.cursor-1, 0))
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		fmt.Println(msg)
		return m, tea.Quit
	case topStoriesMsg:
		m.loaded = true
		msg.stories[0] = 126809 // TODO remove this poll test
		msg.stories[1] = 26734955
		rootStory := m.getPost(rootStoryId)
		rootStory.Kids = msg.stories
		rootStory.Descendants = len(msg.stories)
	case posts.Post:
		m.stories[msg.Id] = &msg
		if msg.Storytype == "poll" {
			// load poll opts
			for _, pollOptId := range msg.Parts {
				m.getPost(pollOptId) // will trigger loading if needed
			}
		}
	case spinner.TickMsg:
		// tick spinner
		var tickCmd tea.Cmd
		m.spinner, tickCmd = m.spinner.Update(msg)
		return m, tickCmd
	case loadTickMsg:
		var batch []tea.Cmd
		batch = append(batch, m.loadTick()) // queue next tick
		for stId := range m.toLoad.Hash {
			stIdStr := strconv.Itoa(stId.(int))
			batch = append(batch, fetchStory(stIdStr))
		}
		m.toLoad.Clear()
		return m, tea.Batch(batch...)
	case tea.KeyMsg:
		// handle keyboard
		return m.keyHandler(msg)
	case tea.MouseMsg:
		// handle mouse
		return m.MouseHandler(msg)
	}

	return m, nil
}

func (m model) View() string {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	cappedW := min(w, maxWidth)

	// top bar
	remainingH := h
	ret := style.TitleBar.Width(w).Render("HackerReader")
	remainingH -= lipgloss.Height(ret)
	if !m.loaded {
		// app not loaded yet
		return lipgloss.JoinVertical(lipgloss.Left,
			ret,
			lipgloss.JoinHorizontal(lipgloss.Top,
				m.spinner.View(), " ", style.SecondaryStyle.Render("Loading..."),
			),
		)
	}

	if m.inFocus > 0 {
		// in focus mode
		focusedSt := m.getPost(m.inFocus)
		focusedStr := focusedSt.View(false, true, cappedW, m.stories, &m.spinner)
		focusedStrSplit := strings.Split(focusedStr, "\n")
		return lipgloss.JoinVertical(lipgloss.Left,
			ret,
			strings.Join(focusedStrSplit[min(m.inFocusCursor, len(focusedStrSplit)-1):], "\n"),
		)
	}

	// current story (if any selected (can be root))
	parentStory := m.getPost(m.selected.Peek().(int))
	if parentStory.Id != rootStoryId {
		mainItemStr := parentStory.View(true, true, cappedW-4, m.stories, &m.spinner)
		mainItemStr = style.MainItem.
			Width(cappedW - 2).
			MaxHeight(remainingH).
			Render(lipgloss.JoinHorizontal(lipgloss.Top, " ", mainItemStr))
		remainingH -= lipgloss.Height(mainItemStr)
		ret = lipgloss.JoinVertical(
			lipgloss.Left,
			ret,
			mainItemStr,
		)
	}

	// iterate over children
	starti := m.cursor
	for i := starti; i < len(parentStory.Kids) && remainingH > 0; i++ {
		st := m.getPost(parentStory.Kids[i])
		highlight := m.cursor == i // is this the current selected entry?

		orderI := style.PrimaryStyle.Copy().
			Bold(highlight).Render(fmt.Sprintf(" %d. ", i))
		cursor := " "
		if highlight {
			cursor = style.Checkmark(">")
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, orderI, cursor)
		// 2 for borders + 1 for end padding
		remainingW := cappedW - lipgloss.Width(row) - 3
		listItemStr := st.View(highlight, false, remainingW, m.stories, &m.spinner)
		itemStr := lipgloss.JoinHorizontal(
			lipgloss.Top,
			cursor,
			orderI,
			listItemStr,
		)

		nextRemainingH := remainingH - (lipgloss.Height(itemStr) + 1)
		if i == starti {
			nextRemainingH--
		}
		if !(i+1 < len(parentStory.Kids) && nextRemainingH > 0) {
			// last item
			style.ListItemBorder.BottomLeft = "└"
			style.ListItemBorder.BottomRight = "┘"
		} else {
			style.ListItemBorder.BottomLeft = "├"
			style.ListItemBorder.BottomRight = "┤"
		}
		listItemStyle := style.ListItem.Copy().
			Width(cappedW-2).
			MaxHeight(remainingH).
			Border(style.ListItemBorder, i == starti, true, true)
		if highlight {
			listItemStyle.BorderForeground(style.GreenColor)
		}
		itemStr = listItemStyle.Render(itemStr)

		remainingH -= lipgloss.Height(itemStr)
		if remainingH >= 0 {
			ret = lipgloss.JoinVertical(
				lipgloss.Left,
				ret,
				itemStr,
			)
		}
	}

	return ret
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
