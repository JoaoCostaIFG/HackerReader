package main

import (
	"encoding/json"
	"fmt"
	"hackerreader/posts"
	"hackerreader/style"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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
	loadBacklogSize = 25
	rootStoryId     = 0
	maxWidth        = 135
)

type model struct {
	stories    map[int]posts.Post
	cursor     int
	prevCursor stack.Stack
	selected   stack.Stack
	spinner    spinner.Model
}

func initialModel() model {
	initModel := model{
		stories:    make(map[int]posts.Post),
		cursor:     0,
		prevCursor: stack.Stack{},
		selected:   stack.Stack{},
		spinner:    spinner.New(),
	}
	// spinner
	initModel.spinner.Spinner = style.SpinnerSpinner
	initModel.spinner.Style = style.SpinnerStyle
	// add root "story" => top stories are its children
	rootSt := posts.New()
	rootSt.Id = rootStoryId
	initModel.stories[rootStoryId] = rootSt

	initModel.selected.Push(rootStoryId)
	return initModel
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

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

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchTopStories, m.spinner.Tick)
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

func (m model) batchKidsFetch(st *posts.Post) tea.Cmd {
	var batch []tea.Cmd
	for i := m.cursor; i < len(st.Kids) && i < m.cursor+loadBacklogSize; i++ {
		stId := st.Kids[i]
		loadSt, exists := m.stories[stId]
		if !exists {
			// set loading and start loading
			loadSt.Id = -1
			m.stories[stId] = loadSt
			batch = append(batch, fetchStory(strconv.Itoa(stId)))
		}
	}
	return tea.Batch(batch...)
}

func (m model) KeyHandler(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q": // quit
		return m, tea.Quit
	case "g":
		m.cursor = 0
	case "G":
		st, _ := m.stories[m.selected.Peek().(int)]
		m.cursor = len(st.Kids) - 1
	case "down", "j": // point down
		st, _ := m.stories[m.selected.Peek().(int)]
		if m.cursor < len(st.Kids)-1 {
			m.cursor++
			// load missing stories
			return m, m.batchKidsFetch(&st)
		}
	case "up", "k": // point up
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter", "right", "l": // go in
		parentStory, _ := m.stories[m.selected.Peek().(int)]
		if len(parentStory.Kids) > 0 {
			// can only go in if there's kids
			stId := parentStory.Kids[m.cursor]
			st, exists := m.stories[stId]
			if exists && st.Id > 0 {
				// loaded => we can go in
				// save previous state for when we go back
				m.prevCursor.Push(m.cursor)
				m.selected.Push(stId)
				// go in and load kids (if needed)
				m.cursor = 0
				return m, m.batchKidsFetch(&st)
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
		parentStory, _ := m.stories[m.selected.Peek().(int)]
		if len(parentStory.Kids) > 0 {
			st, exists := m.stories[parentStory.Kids[m.cursor]]
			if exists && st.Id > 0 {
				st.Hidden = !st.Hidden
				m.stories[parentStory.Kids[m.cursor]] = st
			}
		}
	case "o": // open story URL is browser
		parentStory, _ := m.stories[m.selected.Peek().(int)]

		var targetStory posts.Post
		if m.selected.Len() > 1 {
			targetStory = parentStory
		} else if len(parentStory.Kids) > 0 {
			targetStory = m.stories[parentStory.Kids[m.cursor]]
		}

		if len(targetStory.Url) > 0 {
			_ = browser.OpenURL(targetStory.Url)
		}
	case "O": // open story in browser
		parentStory, _ := m.stories[m.selected.Peek().(int)]
		if len(parentStory.Kids) > 0 {
			targetSt := m.stories[parentStory.Kids[m.cursor]]
			targetId := targetSt.Id
			_ = browser.OpenURL(itemUrl + strconv.Itoa(targetId))
		}
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		fmt.Println(msg)
		return m, tea.Quit
	case topStoriesMsg:
		msg.stories[0] = 126809 // TODO remove this poll test
		rootStory := posts.Post{
			Id:          rootStoryId,
			Kids:        msg.stories,
			Descendants: len(msg.stories),
		}
		m.stories[rootStoryId] = rootStory
		return m, m.batchKidsFetch(&rootStory)
	case posts.Post:
		// we have a story => we're ready
		m.stories[msg.Id] = msg
		if msg.Storytype == "poll" {
			var batch []tea.Cmd
			for _, pollOptId := range msg.Parts {
				pollOpt, exists := m.stories[pollOptId]
				if !exists {
					// set loading and start loading
					pollOpt.Id = -1
					m.stories[pollOptId] = pollOpt
					batch = append(batch, fetchStory(strconv.Itoa(pollOptId)))
				}
			}
			return m, tea.Batch(batch...)
		}
	case spinner.TickMsg:
		// tick spinner
		var tickCmd tea.Cmd
		m.spinner, tickCmd = m.spinner.Update(msg)
		return m, tickCmd
	case tea.KeyMsg:
		return m.KeyHandler(msg)
	}

	return m, nil
}

func (m model) View() string {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	cappedW := min(w, maxWidth)

	// current story (can be root)
	parentStory, exists := m.stories[m.selected.Peek().(int)]
	if !exists {
		return ""
	}

	// top bar
	remainingH := h
	ret := style.TitleBar.Width(w).Render("HackerReader")
	remainingH -= lipgloss.Height(ret)

	// current story (if any selected)
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
		st, _ := m.stories[parentStory.Kids[i]]
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
		st.Text = fmt.Sprintf("%s\n%d", st.Text, remainingW)
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
		itemStr = style.ListItem.
			Width(cappedW-2).
			MaxHeight(remainingH).
			Border(style.ListItemBorder, i == starti, true, true).
			Render(itemStr)

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
	)
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
