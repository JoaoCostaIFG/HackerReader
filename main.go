package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	tea "github.com/charmbracelet/bubbletea"
)

const apiurl = "https://hacker-news.firebaseio.com/v0"

type story struct {
	id          int // -1 when not loaded
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
	topstories []int
	state      int // 0 - loading; 1 - inited; 2 - in story
	selected   int
}

func initialModel() model {
	return model{
		stories:    make(map[int]story),
		topstories: []int{},
		state:      0,
	}
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
				data.by = string(v)
			case 2:
				v, _ := jsonparser.ParseInt(value)
				data.time = int(v)
				data.timestr = timestampToString(int64(data.time))
			case 3:
				v, _ := jsonparser.ParseString(value)
				data.storytype = string(v)
			case 4:
				v, _ := jsonparser.ParseString(value)
				data.title = string(v)
			case 5:
				v, _ := jsonparser.ParseString(value)
				data.text = string(v)
			case 6:
				v, _ := jsonparser.ParseString(value)
				data.url = string(v)
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
				// TODO
			case 10:
				// TODO
			case 11:
				v, _ := jsonparser.ParseInt(value)
				data.poll = int(v)
			}
		}, paths...)

		return data
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		return m, tea.Quit
	case topStoriesMsg:
		m.topstories = msg.stories
		var batch [5]tea.Cmd
		for i := 0; i < len(m.topstories) && i < 5; i++ {
			stId := m.topstories[i]
			m.stories[stId] = story{id: -1}
			batch[i] = fetchStory(strconv.Itoa(stId))
		}
		return m, tea.Batch(batch[:]...)
	case story:
		m.stories[msg.id] = msg
		// we have a story => we're ready
		m.state = 1
	case tea.KeyMsg:
		switch m.state {
		case 1:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "down", "j":
				if m.cursor < len(m.topstories)-1 {
					m.cursor++
					// load missing stories
					var batch []tea.Cmd
					for i := m.cursor; i < len(m.topstories) && i < m.cursor+5; i++ {
						stId := m.topstories[i]
						_, exists := m.stories[stId]
						if !exists {
							batch = append(batch, fetchStory(strconv.Itoa(stId)))
						}
					}
					return m, tea.Batch(batch...)
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "enter", "right", "l":
				m.selected = m.cursor
				stId := m.topstories[m.selected]
				st, exists := m.stories[stId]
				if exists && st.id > 0 {
					// loaded => we can go in
					m.state = 2
				}
			}
		case 2:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "left", "h":
				m.state = 1
			}
		default:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m model) selectionScreen() string {
	// The header
	s := "HackerReader\n\n"

	// Iterate over stories
	starti := m.cursor - 2
	if starti < 0 {
		starti = 0
	}
	for i := starti; i < len(m.topstories) && i < starti+5; i++ {
		stId := m.topstories[i]
		st, exists := m.stories[stId]

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		// Render the row
		if !exists || st.id < 0 {
			// still loading/hasn't started loading
			s += fmt.Sprintf("%s %d. %s (%s)\n", cursor, i, "...", "...")
			s += fmt.Sprintf("     %d points by %s %s | %d comments\n", 0, "...", "...", 0)
		} else {
			s += fmt.Sprintf("%s %d. %s (%s)\n", cursor, i, st.title, st.domain)
			s += fmt.Sprintf("     %d points by %s %s | %d comments\n", st.score, st.by, st.timestr, st.descendants)
		}
	}

	return s
}

func (m model) storyView() string {
	stId := m.topstories[m.selected]
	st, exists := m.stories[stId]
	if !exists {
		return ""
	}

	s := fmt.Sprintf("%s (%s)\n", st.title, st.domain)
	s += fmt.Sprintf("%d points by %s %s | %d comments\n", st.score, st.by, st.timestr, st.descendants)

	return s
}

func (m model) View() string {
	switch m.state {
	case 0:
		return "Still loading..."
	case 1:
		return m.selectionScreen()
	case 2:
		return m.storyView()
	default:
		return ""
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
