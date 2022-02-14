package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang-collections/collections/stack"
	"golang.org/x/term"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/buger/jsonparser"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
)

const (
	apiurl          = "https://hacker-news.firebaseio.com/v0"
	listSize        = 5
	loadBacklogSize = 10
	rootStoryId     = -1
)

var (
	bg        = lipgloss.Color("#0F0F04")
	orange    = lipgloss.Color("#FF6600")
	primary   = lipgloss.Color("#B8AFA1")
	secondary = lipgloss.Color("#706758")
)

type story struct {
	hidden      bool // whether the story has been hidden or not
	id          int  // -1 when not loaded
	by          string
	time        int
	timestr     string
	storytype   string
	title       string
	text        string
	url         string
	domain      string
	score       int
	descendants int
	kids        []int
	parts       []int
	poll        int
}

type model struct {
	stories    map[int]story
	cursor     int
	prevCursor stack.Stack
	selected   stack.Stack
}

func initialModel() model {
	initModel := model{
		stories:    make(map[int]story),
		cursor:     0,
		prevCursor: stack.Stack{},
		selected:   stack.Stack{},
	}
	// add root "story" => top stories are its children
	initModel.stories[rootStoryId] = story{
		id:          rootStoryId,
		descendants: 0,
		kids:        []int{},
	}
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
	return fetchTopStories
}

func timestampToString(timestamp int64) string {
	diff := int64(time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds())
	if diff < 60 {
		return fmt.Sprintf("%d seconds ago", diff)
	}
	diff = diff / 60
	if diff < 60 {
		return fmt.Sprintf("%d minutes ago", diff)
	}
	diff = diff / 60
	if diff < 60 {
		return fmt.Sprintf("%d hours ago", diff)
	}
	diff = diff / 24
	if diff == 1 {
		return "a day ago"
	} else if diff < 7 {
		return fmt.Sprintf("%d days ago", diff)
	} else if diff < 30 {
		diff = diff / 7
		return fmt.Sprintf("%d weeks ago", diff)
	}
	diff = diff / 30
	if diff == 1 {
		return "a month ago"
	} else if diff < 12 {
		return fmt.Sprintf("%d months ago", diff)
	}
	diff = diff / 365
	if diff == 1 {
		return "a year ago"
	}
	return fmt.Sprintf("%d years ago", diff)
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

		data := story{}

		paths := [][]string{
			{"id"},
			{"by"},
			{"time"},
			{"type"},
			{"title"},
			{"text"},
			{"url"},
			{"score"},
			{"descendants"},
			{"kids"},
			{"parts"},
			{"poll"},
		}
		jsonparser.EachKey(bodyBytes, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
				v, _ := jsonparser.ParseInt(value)
				data.id = int(v)
			case 1:
				v, _ := jsonparser.ParseString(value)
				data.by = v
			case 2:
				v, _ := jsonparser.ParseInt(value)
				data.time = int(v)
				data.timestr = timestampToString(int64(data.time))
			case 3:
				v, _ := jsonparser.ParseString(value)
				data.storytype = v
			case 4:
				v, _ := jsonparser.ParseString(value)
				data.title = v
			case 5:
				v, _ := jsonparser.ParseString(value)
				data.text, err = md.NewConverter("", true, nil).ConvertString(v)
				if err != nil {
					// fallback
					data.text = html.UnescapeString(v)
				} else {
					data.text = strings.ReplaceAll(data.text, "\n\n", "\n")
				}
			case 6:
				v, _ := jsonparser.ParseString(value)
				data.url = v
				u, _ := url.Parse(data.url)
				parts := strings.Split(u.Hostname(), ".")
				data.domain = parts[len(parts)-2] + "." + parts[len(parts)-1]
			case 7:
				v, _ := jsonparser.ParseInt(value)
				data.score = int(v)
			case 8:
				v, _ := jsonparser.ParseInt(value)
				data.descendants = int(v)
			case 9:
				_, _ = jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					v, _ := jsonparser.ParseInt(value)
					data.kids = append(data.kids, int(v))
				})
			case 10:
				_, _ = jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					v, _ := jsonparser.ParseInt(value)
					data.parts = append(data.parts, int(v))
				})
			case 11:
				v, _ := jsonparser.ParseInt(value)
				data.poll = int(v)
			}
		}, paths...)

		data.hidden = false
		return data
	}
}

func (m model) batchKidsFetch(st story) tea.Cmd {
	var batch []tea.Cmd
	for i := m.cursor; i < len(st.kids) && i < m.cursor+loadBacklogSize; i++ {
		stId := st.kids[i]
		loadSt, exists := m.stories[stId]
		if !exists {
			// set loading and start loading
			loadSt.id = -1
			m.stories[stId] = loadSt
			batch = append(batch, fetchStory(strconv.Itoa(stId)))
		}
	}
	return tea.Batch(batch...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		fmt.Println(msg)
		return m, tea.Quit
	case topStoriesMsg:
		msg.stories[0] = 126809 // TODO remove this poll test
		rootStory := story{
			id:          rootStoryId,
			kids:        msg.stories,
			descendants: len(msg.stories),
		}
		m.stories[rootStoryId] = rootStory
		return m, m.batchKidsFetch(rootStory)
	case story:
		// we have a story => we're ready
		m.stories[msg.id] = msg
		if msg.storytype == "poll" {
			var batch []tea.Cmd
			for _, polloptId := range msg.parts {
				pollopt, exists := m.stories[polloptId]
				if !exists {
					// set loading and start loading
					pollopt.id = -1
					m.stories[polloptId] = pollopt
					batch = append(batch, fetchStory(strconv.Itoa(polloptId)))
				}
			}
			return m, tea.Batch(batch...)
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "down", "j":
			st, _ := m.stories[m.selected.Peek().(int)]
			if m.cursor < len(st.kids)-1 {
				m.cursor++
				// load missing stories
				return m, m.batchKidsFetch(st)
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", "right", "l":
			parentStory, _ := m.stories[m.selected.Peek().(int)]
			if len(parentStory.kids) > 0 {
				// can only go in if there's kids
				stId := parentStory.kids[m.cursor]
				st, exists := m.stories[stId]
				if exists && st.id > 0 {
					// loaded => we can go in
					// save previous state for when we go back
					m.prevCursor.Push(m.cursor)
					m.selected.Push(stId)
					// go in and load kids (if needed)
					m.cursor = 0
					return m, m.batchKidsFetch(st)
				}
			}
		case "escape", "left", "h":
			// recover previous state
			if m.selected.Len() > 1 {
				// we're nested (rootStory can't be popped)
				m.cursor = m.prevCursor.Pop().(int)
				m.selected.Pop()
			}
		case " ":
			// hide/unhide given story
			parentStory, _ := m.stories[m.selected.Peek().(int)]
			if len(parentStory.kids) > 0 {
				st, exists := m.stories[parentStory.kids[m.cursor]]
				if exists && st.id > 0 {
					st.hidden = !st.hidden
					m.stories[parentStory.kids[m.cursor]] = st
				}
			}
		case "o":
			parentStory, _ := m.stories[m.selected.Peek().(int)]

			var targetStory story
			if m.selected.Len() > 1 {
				targetStory = parentStory
			} else if len(parentStory.kids) > 0 {
				targetStory = m.stories[parentStory.kids[m.cursor]]
			}

			if len(targetStory.url) > 0 {
				_ = browser.OpenURL(targetStory.url)
			}
		}
	}

	return m, nil
}

func (m model) mainItemView(st story) string {
	if st.id < 0 {
		// still loading/hasn't started loading => shouldn't happen but it doesn't hurt to check
		ret := fmt.Sprintf("%s (%s)\n", "...", "...")
		ret += fmt.Sprintf("%d points by %s %s | %d comments", 0, "...", "...", 0)
		return ret
	}

	switch st.storytype {
	case "comment":
		ret := fmt.Sprintf("%s %s\n", st.by, st.timestr)
		ret += st.text
		return ret
	case "poll":
		ret := fmt.Sprintf("%s\n", st.title)
		ret += fmt.Sprintf("%d points by %s %s | %d comments", st.score, st.by, st.timestr, st.descendants)
		for i := 0; i < len(st.parts); i++ {
			polloptId := st.parts[i]
			pollopt := m.stories[polloptId]
			ret += "\n" + listItemView(pollopt, " ")
		}
		return ret
	default:
		ret := fmt.Sprintf("%s", st.title)
		if len(st.domain) > 0 {
			ret += fmt.Sprintf(" (%s)", st.domain)
		}
		ret += "\n"
		ret += fmt.Sprintf("%d points by %s %s | %d comments", st.score, st.by, st.timestr, st.descendants)
		if len(st.text) > 0 {
			ret += "\n" + st.text
		}
		return ret
	}
}

func listItemView(st story, prefix string) string {
	if st.id < 0 {
		// still loading/hasn't started loading
		ret := fmt.Sprintf("%s%s (%s)\n", prefix, "...", "...")
		for i := 0; i < len(prefix); i++ {
			ret += " "
		}
		ret += fmt.Sprintf("%d points by %s %s | %d comments", 0, "...", "...", 0)
		return ret
	}

	if st.hidden {
		// hidden post
		ret := fmt.Sprintf("%s(hidden) %s %s", prefix, st.by, st.timestr)
		return ret
	}

	padding := ""
	for i := 0; i < len(prefix); i++ {
		padding += " "
	}

	switch st.storytype {
	case "comment":
		ret := fmt.Sprintf("%s%s %s\n", prefix, st.by, st.timestr)
		ret += padding
		ret += strings.ReplaceAll(st.text, "\n", "\n"+padding)
		return ret
	case "pollopt":
		ret := prefix + st.text + "\n"
		ret += padding
		ret += fmt.Sprintf("%d points", st.score)
		return ret
	default:
		ret := fmt.Sprintf("%s%s", prefix, st.title)
		if len(st.domain) > 0 {
			ret += fmt.Sprintf(" (%s)", st.domain)
		}
		ret += "\n" + padding
		ret += fmt.Sprintf("%d points by %s %s | %d comments", st.score, st.by, st.timestr, st.descendants)
		return ret
	}
}

func (m model) storyScreen(int, int) string {
	parentStory, exists := m.stories[m.selected.Peek().(int)]
	if !exists {
		return ""
	}

	s := m.mainItemView(parentStory) + "\n\n"

	// Iterate over comments
	starti := m.cursor - 2
	if starti < 0 {
		starti = 0
	}
	for i := starti; i < len(parentStory.kids) && i < starti+listSize; i++ {
		stId := parentStory.kids[i]
		st, _ := m.stories[stId]

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		prefix := fmt.Sprintf("%s %d. ", cursor, i)
		s += listItemView(st, prefix) + "\n"
	}

	return s
}

func (m model) rootScreen(w int, h int) string {
	// header
	parentStory, exists := m.stories[m.selected.Peek().(int)]
	if !exists {
		return ""
	}

	s := "HackerReader\n\n"

	// iterate over stories
	starti := m.cursor - 2
	if starti < 0 {
		starti = 0
	}
	for i := starti; i < len(parentStory.kids) && i < starti+listSize; i++ {
		stId := parentStory.kids[i]
		st, _ := m.stories[stId]

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		prefix := fmt.Sprintf("%s %d. ", cursor, i)
		s += listItemView(st, prefix) + "\n"
	}

	return s
}

func (m model) View() string {
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	if m.selected.Len() > 1 {
		// we're nested
		return m.storyScreen(w, h)
	}
	// we're at root
	return m.rootScreen(w, h)
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
