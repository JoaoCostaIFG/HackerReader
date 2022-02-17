package main

import (
	"encoding/json"
	"fmt"
	"hackerreader/posts"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/buger/jsonparser"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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
	initModel.spinner.Spinner = spinnerSpinner
	initModel.spinner.Style = spinnerStyle
	// add root "story" => top stories are its children
	initModel.stories[rootStoryId] = posts.Post{
		Id:          rootStoryId,
		Descendants: 0,
		Kids:        []int{},
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

		data := posts.Post{}

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
			{"parent"},
			{"dead"},
			{"deleted"},
		}
		jsonparser.EachKey(bodyBytes, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
				v, _ := jsonparser.ParseInt(value)
				data.Id = int(v)
			case 1:
				v, _ := jsonparser.ParseString(value)
				data.By = v
			case 2:
				v, _ := jsonparser.ParseInt(value)
				data.Time = int(v)
				data.Timestr = timestampToString(int64(data.Time))
			case 3:
				v, _ := jsonparser.ParseString(value)
				data.Storytype = v
			case 4:
				v, _ := jsonparser.ParseString(value)
				data.Title = v
			case 5:
				v, _ := jsonparser.ParseString(value)
				data.Text, err = html2md.NewConverter("", true, nil).ConvertString(v)
				if err != nil {
					// fallback
					data.Text = html.UnescapeString(v)
				} else {
					data.Text = strings.ReplaceAll(data.Text, "\n\n", "\n")
					// TODO wait for escape support to remove this
					// TODO https://github.com/charmbracelet/glamour/issues/106
					data.Text = strings.ReplaceAll(data.Text, "\\-", "-")
					data.Text = strings.ReplaceAll(data.Text, "\\>", ">")
					data.Text = strings.ReplaceAll(data.Text, "\\[", "[")
					data.Text = strings.ReplaceAll(data.Text, "\\]", "]")
				}
			case 6:
				v, _ := jsonparser.ParseString(value)
				data.Url = v
				u, _ := url.Parse(data.Url)
				parts := strings.Split(u.Hostname(), ".")
				data.Domain = parts[len(parts)-2] + "." + parts[len(parts)-1]
			case 7:
				v, _ := jsonparser.ParseInt(value)
				data.Score = int(v)
			case 8:
				v, _ := jsonparser.ParseInt(value)
				data.Descendants = int(v)
			case 9:
				_, _ = jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					v, _ := jsonparser.ParseInt(value)
					data.Kids = append(data.Kids, int(v))
				})
			case 10:
				_, _ = jsonparser.ArrayEach(value, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					v, _ := jsonparser.ParseInt(value)
					data.Parts = append(data.Parts, int(v))
				})
			case 11:
				v, _ := jsonparser.ParseInt(value)
				data.Poll = int(v)
			case 12:
				v, _ := jsonparser.ParseInt(value)
				data.Parent = int(v)
			case 13:
				v, _ := jsonparser.ParseBoolean(value)
				data.Dead = v
			case 14:
				v, _ := jsonparser.ParseBoolean(value)
				data.Deleted = v
			}
		}, paths...)

		data.Hidden = false
		return data
	}
}

func (m model) batchKidsFetch(st posts.Post) tea.Cmd {
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
			return m, m.batchKidsFetch(st)
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
				return m, m.batchKidsFetch(st)
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
		return m, m.batchKidsFetch(rootStory)
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

func (m model) listItemView(st posts.Post, highlight bool, selected bool, w int) string {
	if st.Deleted || st.Dead {
		// deleted story
		return secondaryStyle.Copy().
			Bold(highlight).
			MaxWidth(w).
			Render(fmt.Sprintf("[deleted] %s", st.Timestr))
	}

	if st.Hidden && !selected {
		// hidden post (and not selected (parent))
		return secondaryStyle.Copy().
			Bold(highlight).
			MaxWidth(w).
			Render(fmt.Sprintf("(hidden) %s %s", st.By, st.Timestr))
	}

	if st.Id < 0 {
		// still loading/hasn't started loading
		return lipgloss.JoinHorizontal(lipgloss.Top,
			m.spinner.View(),
			" ",
			secondaryStyle.Copy().
				Bold(highlight).
				MaxWidth(w).
				Render("Loading...\n..."),
		)
	}

	switch st.Storytype {
	case "comment":
		// -1 so wordwrap doesn't feel like ignoring the wrap
		mdRenderer, _ := glamour.NewTermRenderer(
			glamour.WithStyles(mdStyleConfig),
			glamour.WithEmoji(),
			glamour.WithWordWrap(w-1),
		)
		commentTxt, _ := mdRenderer.Render(st.Text)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			secondaryStyle.Copy().
				Bold(highlight).
				MaxWidth(w).
				Render(st.By+" "+st.Timestr),
			strings.TrimRight(commentTxt, "\n"),
		)
	case "pollopt":
		return lipgloss.JoinVertical(
			lipgloss.Left,
			primaryStyle.Copy().
				Bold(highlight).
				Width(w).
				Render(st.Text),
			secondaryStyle.Copy().
				Bold(highlight).
				MaxWidth(w).
				Render(fmt.Sprintf("%d points", st.Score)),
		)
	default:
		// title should wrap if needed, but leave space for domain if possible
		stTitleStyle := primaryStyle.Copy().Bold(highlight)
		if len(st.Title) > w {
			stTitleStyle.Width(w)
		} else {
			stTitleStyle.MaxWidth(w)
		}
		row := stTitleStyle.Render(st.Title)

		if len(st.Domain) > 0 {
			// story has a URL
			remainingW := w - lipgloss.Width(row)
			if remainingW < len(st.Domain)-3 { // 1 space + 2 parentheses
				// no space => go to next line
				row = lipgloss.JoinVertical(
					lipgloss.Left,
					row,
					urlStyle.Copy().
						Bold(highlight).
						MaxWidth(w).
						Render(fmt.Sprintf("(%s)", st.Domain)),
				)
			} else {
				row = lipgloss.JoinHorizontal(
					lipgloss.Top,
					row,
					urlStyle.Copy().
						Bold(highlight).
						MaxWidth(remainingW).
						Render(fmt.Sprintf(" (%s)", st.Domain)),
				)
			}
		}

		if selected {
			if len(st.Text) > 0 {
				// story has text
				mdRenderer, _ := glamour.NewTermRenderer(
					glamour.WithStyles(mdStyleConfig),
					glamour.WithEmoji(),
					glamour.WithWordWrap(w-1),
				)
				storyTxt, _ := mdRenderer.Render(st.Text)

				row = lipgloss.JoinVertical(
					lipgloss.Left,
					row,
					strings.TrimRight(storyTxt, "\n"),
				)
			}

			if st.Storytype == "poll" {
				// if it is a selected poll => show parts
				for i := 0; i < len(st.Parts); i++ {
					polloptId := st.Parts[i]
					pollopt := m.stories[polloptId]
					row = lipgloss.JoinVertical(
						lipgloss.Left,
						row,
						lipgloss.JoinHorizontal(
							lipgloss.Top,
							"  ",
							m.listItemView(pollopt, false, false, w-2),
						),
					)
				}
			}
		}

		row = lipgloss.JoinVertical(
			lipgloss.Left,
			row,
			secondaryStyle.Copy().
				Bold(highlight).
				MaxWidth(w).
				Render(
					fmt.Sprintf("%d points by %s %s | %d comments", st.Score, st.By, st.Timestr, st.Descendants),
				),
		)

		return row
	}
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
	ret := titleBar.Width(w).Render("HackerReader")
	remainingH -= lipgloss.Height(ret)

	// current story (if any selected)
	if parentStory.Id != rootStoryId {
		mainItemStr := m.listItemView(parentStory, true, true, cappedW-4)
		mainItemStr = mainItem.
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

		orderI := primaryStyle.Copy().
			Bold(highlight).Render(fmt.Sprintf(" %d. ", i))
		cursor := " "
		if highlight {
			cursor = checkmark(">")
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, orderI, cursor)
		// 2 for borders + 1 for end padding
		remainingW := cappedW - lipgloss.Width(row) - 3
		st.Text = fmt.Sprintf("%s\n%d", st.Text, remainingW)
		listItemStr := m.listItemView(st, highlight, false, remainingW)
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
			listItemBorder.BottomLeft = "└"
			listItemBorder.BottomRight = "┘"
		} else {
			listItemBorder.BottomLeft = "├"
			listItemBorder.BottomRight = "┤"
		}
		itemStr = listItem.
			Width(cappedW-2).
			MaxHeight(remainingH).
			Border(listItemBorder, i == starti, true, true).
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
