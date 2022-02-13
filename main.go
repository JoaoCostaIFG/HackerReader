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
	id          int
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
		return m, tea.Batch(
			fetchStory(strconv.Itoa(m.topstories[0])),
			fetchStory(strconv.Itoa(m.topstories[1])),
			fetchStory(strconv.Itoa(m.topstories[2])),
			fetchStory(strconv.Itoa(m.topstories[3])),
			fetchStory(strconv.Itoa(m.topstories[4])),
		)
	case story:
		m.stories[msg.id] = msg
		if len(m.stories) >= 5 {
			m.state = 1
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.stories)-1 {
				m.cursor++
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", "right", "l":
			if m.state == 1 {
				m.selected = m.cursor
				m.state = 2
			}
		case "left", "h":
			if m.state == 2 {
				m.state = 1
			}
		}
	}

	return m, nil
}

func timestampToString(timestamp int64) string {
	diff := int64(time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds())
	if diff < 60 {
		return fmt.Sprintf("%d seconds ago", diff)
	}
	diff = int64(diff / 60)
	if diff < 60 {
		return fmt.Sprintf("%d minutes ago", diff)
	}
	diff = int64(diff / 60)
	if diff < 60 {
		return fmt.Sprintf("%d hours ago", diff)
	}
	diff = int64(diff / 24)
	if diff == 1 {
		return "a day ago"
	} else if diff < 7 {
		return fmt.Sprintf("%d days ago", diff)
	} else if diff < 30 {
		diff = int64(diff / 7)
		return fmt.Sprintf("%d weeks ago", diff)
	}
	diff = int64(diff / 30)
	if diff == 1 {
		return "a month ago"
	} else if diff < 12 {
		return fmt.Sprintf("%d months ago", diff)
	}
	diff = int64(diff / 365)
	if diff == 1 {
		return "a year ago"
	}
	return fmt.Sprintf("%d years ago", diff)
}

func (m model) selectionScreen() string {
	// The header
	s := "HackerReader\n\n"

	// Iterate over stories
	cnt := 0
	starti := m.cursor - 2
	if starti < 0 {
		starti = 0
	}
	for i := starti; i < len(m.topstories) && cnt < 5; i++ {
		stId := m.topstories[i]
		st, exists := m.stories[stId]
		if !exists {
			continue
		}

		cursor := " "
		if m.cursor == cnt {
			cursor = ">"
		}
		// Render the row
		s += fmt.Sprintf("%s %d. %s (%s)\n", cursor, i, st.title, st.domain)
		s += fmt.Sprintf("     %d points by %s %s | %d comments\n", st.score, st.by, st.timestr, st.descendants)
		cnt += 1
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
