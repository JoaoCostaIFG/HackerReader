package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strconv"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/buger/jsonparser"
)

const url = "https://hacker-news.firebaseio.com/v0"

type story struct {
    id          int     `json:"id"`
    by          string  `json:"by"`
    time        int     `json:"time"`
    storytype   string  `json:"type"`
    title       string  `json:"title"`
    text        string  `json:"text"`
    url         string  `json:"url"`
    score       int     `json:"score"`
    descendants int     `json:"descendants"`
    kids        []int   `json:"kids"`
    parts       []int
    poll        int
}

type model struct {
    stories     map[int]story
    cursor      int
    selected    map[int]struct{}
    topstories  []int
    state       int // 0 - loading; 1 - inited; 2 - in story
}

func initialModel() model {
	return model{
		stories:    make(map[int]story),
		selected:   make(map[int]struct{}),
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
    res, err := c.Get(url + "/topstories.json")

    if err != nil {
        return errMsg{err}
    }
    defer res.Body.Close()

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
        res, err := c.Get(url + "/item/" + item + ".json")
        if err != nil {
            return errMsg{err}
        }
        defer res.Body.Close()

        bodyBytes, _ := ioutil.ReadAll(res.Body)
        if err != nil {
            return errMsg{err}
        }

        data := story{}

        paths := [][]string{
          []string{"id"},
          []string{"by"},
          []string{"time"},
          []string{"type"},
          []string{"title"},
          []string{"text"},
          []string{"url"},
          []string{"score"},
          []string{"descendants"},
          []string{"kids"},
          []string{"parts"},
          []string{"poll"},
        }
        jsonparser.EachKey(bodyBytes, func(idx int, value []byte, vt jsonparser.ValueType, err error){
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
            if m.cursor < len(m.stories) - 1 {
                m.cursor++
            }
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "enter", "l", " ":
            _, ok := m.selected[m.cursor]
            if ok {
                delete(m.selected, m.cursor)
            } else {
                m.selected[m.cursor] = struct{}{}
            }
        }
    }

    return m, nil
}

func (m model) View() string {
    if (m.state == 0) {
        return "Still loading..."
    }

    // The header
    s := "Stories\n\n"

    // Iterate over stories
    cnt := 0
    for i:=0; i < len(m.topstories) && cnt < 5; i++ {
        st_id := m.topstories[i]
        st, exists := m.stories[st_id]
        if (!exists) {
            continue
        }

        cursor := " "
        if m.cursor == cnt {
            cursor = ">"
        }
        checked := " "
        if _, ok := m.selected[cnt]; ok {
            checked = "x"
        }
        // Render the row
        s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, st.title)
        cnt += 1
    }

    // The footer
    s += "\nPress q to quit.\n"
    return s
}

func main() {
    p := tea.NewProgram(initialModel())
    if err := p.Start(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
        os.Exit(1)
    }
}
