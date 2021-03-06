package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"hackerreader/posts"
	"hackerreader/set"
	mySpinner "hackerreader/spinner"
	"hackerreader/style"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	w             int
	cappedW       int
	h             int
	loaded        bool
	toLoad        *set.Set
	stories       map[int]*posts.Post
	cursor        int
	prevCursor    *stack.Stack
	selected      *stack.Stack
	spinner       *mySpinner.Spinner
	collapseMain  bool
	inFocus       int
	inFocusCursor int
	lastFrame     *string
}

func initialModel() model {
	lastFrame := ""
	s := mySpinner.New()
	initModel := model{
		loaded:        false,
		toLoad:        set.New(),
		stories:       make(map[int]*posts.Post),
		cursor:        0,
		prevCursor:    stack.New(),
		selected:      stack.New(),
		spinner:       &s,
		collapseMain:  false,
		inFocus:       -1,
		inFocusCursor: 0,
		lastFrame:     &lastFrame, // first frame is empty
	}
	// term size
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	initModel.setTermSize(w, h)
	// add root "story" => top stories are its children
	rootSt := posts.New(initModel.spinner)
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

func (m *model) fetchStory(item string) tea.Cmd {
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
		return posts.FromJSON(bodyBytes, m.spinner)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchTopStories,
		m.spinner.Tick,
		m.loadTick(),
	)
}

func (m *model) setTermSize(w int, h int) {
	m.w = w
	m.h = h
	m.cappedW = min(w, maxWidth)
}

// Returns the post/story and ques lazy loading if needed
func (m *model) getPost(stId int) *posts.Post {
	st, exists := m.stories[stId]
	if !exists {
		// create new post
		newSt := posts.New(m.spinner)
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
	case "G", "alt+[":
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
		m.collapseMain = false // disable main collapsing
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
		m.collapseMain = false // disable main collapsing
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
	case "F":
		m.collapseMain = !m.collapseMain
	case "f": // enter focus mode on current hover
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

func (m *model) setRedraw() {
	*m.lastFrame = ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setTermSize(msg.Width, msg.Height)
		m.setRedraw()
		return m, nil
	case tea.KeyMsg:
		// handle keyboard
		m.setRedraw()
		return m.keyHandler(msg)
	case tea.MouseMsg:
		// handle mouse
		m.setRedraw()
		return m.MouseHandler(msg)
	case errMsg:
		fmt.Println(msg)
		return m, tea.Quit
	case topStoriesMsg:
		m.loaded = true
		rootStory := m.getPost(rootStoryId)
		rootStory.Kids = msg.stories
		rootStory.Descendants = len(msg.stories)
		m.setRedraw()
		return m, nil
	case loadTickMsg:
		var batch []tea.Cmd
		for stId := range m.toLoad.Hash {
			stIdStr := strconv.Itoa(stId.(int))
			batch = append(batch, m.fetchStory(stIdStr))
		}
		m.toLoad.Clear()
		batch = append(batch, m.loadTick()) // queue next tick
		return m, tea.Batch(batch...)
	case posts.Post:
		m.stories[msg.Id] = &msg
		if msg.Storytype == "poll" {
			// load poll opts
			for _, pollOptId := range msg.Parts {
				m.getPost(pollOptId) // will trigger loading if needed
			}
		}
		m.setRedraw()
		return m, nil
	case spinner.TickMsg:
		// tick spinner
		var tickCmd tea.Cmd
		tickCmd = m.spinner.Update(msg)
		if m.spinner.IsEnabled() {
			m.setRedraw()
			m.spinner.Disable()
		}
		return m, tickCmd
	}

	return m, nil
}

func (m *model) listItemView(parentStory *posts.Post, i int, w int) string {
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
	remainingW := w - lipgloss.Width(row) - 3
	listItemStr := st.View(highlight, false, remainingW, m.stories)
	itemStr := lipgloss.JoinHorizontal(lipgloss.Top,
		cursor, orderI, listItemStr)

	if i == parentStory.KidCount()-1 {
		// last item
		style.ListItemBorder.BottomLeft = "???"
		style.ListItemBorder.BottomRight = "???"
	} else {
		style.ListItemBorder.BottomLeft = "???"
		style.ListItemBorder.BottomRight = "???"
	}
	listItemStyle := style.ListItem.Copy().
		Width(w-2).
		Border(style.ListItemBorder, i == 0, true, true)
	if i == m.cursor-1 {
		listItemStyle.BorderBottomForeground(style.GreenColor)
	}
	if highlight {
		listItemStyle.BorderForeground(style.GreenColor)
	}

	return listItemStyle.Render(itemStr)
}

func (m model) View() string {
	if len(*m.lastFrame) > 0 {
		return *m.lastFrame
	}

	// top bar
	remainingH := m.h
	ret := style.TitleBar.Width(m.w).Render("HackerReader")
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
		focusedStr := focusedSt.View(false, true, m.cappedW, m.stories)
		focusedStrSplit := strings.Split(focusedStr, "\n")
		return lipgloss.JoinVertical(lipgloss.Left,
			ret,
			strings.Join(focusedStrSplit[min(m.inFocusCursor, len(focusedStrSplit)-1):], "\n"),
		)
	}

	// current story (if any selected (can be root))
	parentStory := m.getPost(m.selected.Peek().(int))
	if parentStory.Id != rootStoryId {
		var mainItemStr string
		if m.collapseMain {
			mainItemStr = style.PrimaryStyle.Copy().Bold(true).Render("Collapsed story")
		} else {
			mainItemStr = parentStory.View(true, true, m.cappedW-4, m.stories)
		}

		mainItemStr = style.MainItem.
			Width(m.cappedW - 2).
			MaxHeight(remainingH).
			Render(lipgloss.JoinHorizontal(lipgloss.Top, " ", mainItemStr))
		remainingH -= lipgloss.Height(mainItemStr)
		ret = lipgloss.JoinVertical(
			lipgloss.Left,
			ret,
			mainItemStr,
		)
	}

	if !parentStory.HasKids() {
		return ret
	}
	// iterate over children
	maxItemListH := remainingH
	itemList := m.listItemView(parentStory, m.cursor, m.cappedW)
	cursorTop := 0
	cursorBot := lipgloss.Height(itemList)
	remainingH -= cursorBot
	for offset := 1; offset < max(m.cursor, len(parentStory.Kids)) && remainingH > 0; offset++ {
		var i int
		// up
		i = m.cursor - offset
		if i >= 0 {
			itemStr := m.listItemView(parentStory, i, m.cappedW)
			itemStrHeight := lipgloss.Height(itemStr)
			cursorTop += itemStrHeight
			cursorBot += itemStrHeight
			remainingH -= itemStrHeight
			itemList = lipgloss.JoinVertical(lipgloss.Left, itemStr, itemList)
		}
		// down
		i = m.cursor + offset
		if i < parentStory.KidCount() {
			itemStr := m.listItemView(parentStory, i, m.cappedW)
			remainingH -= lipgloss.Height(itemStr)
			itemList = lipgloss.JoinVertical(lipgloss.Left, itemList, itemStr)
		}
	}

	itemListSplit := strings.Split(itemList, "\n")
	changed := 2
	alternator := 0
	for cursorBot-cursorTop < maxItemListH && changed > 0 {
		if alternator == 0 {
			if cursorTop > 0 {
				changed++
				cursorTop--
			} else {
				changed--
			}
		} else {
			if cursorBot < len(itemListSplit) {
				changed++
				cursorBot++
			} else {
				changed--
			}
		}
		alternator = (alternator + 1) % 2
	}
	if cursorBot-cursorTop > maxItemListH {
		// special case where only hovered post fits and is too big
		cursorBot -= cursorBot - cursorTop - maxItemListH
	}

	ret = lipgloss.JoinVertical(
		lipgloss.Left,
		ret,
		strings.Join(itemListSplit[cursorTop:cursorBot], "\n"),
	)
	*m.lastFrame = ret // save last frame
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
