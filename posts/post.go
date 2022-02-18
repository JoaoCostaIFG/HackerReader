package posts

import (
	"fmt"
	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/buger/jsonparser"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"hackerreader/style"
	"html"
	"strings"
)

const (
	loadingId = -1
)

var (
	html2mdConverter = html2md.NewConverter("", true, nil)
)

type Post struct {
	// API fields
	Id          int    // -1 when not loaded
	By          string // The username of the item's author.
	Time        int    // Creation date of the item, in Unix Time.
	Storytype   string // The type of item. One of "job", "story", "comment", "poll", or "pollopt".
	Title       string // The title of the story, poll or job (HTML).
	Text        string // The comment, story or poll text (HTML).
	Url         string // The URL of the story.
	Score       int    // The story's score, or the votes for a pollopt.
	Descendants int    // In the case of stories or polls, the total comment count.
	Kids        []int  // The ids of the item's comments, in ranked display order.
	Parts       []int  // A list of related pollopts, in display order.
	Poll        int    // The pollopt's associated poll.
	Parent      int    // The comment's parent: either another comment or the relevant story.
	Dead        bool   // true if the item is dead.
	Deleted     bool   // true, if the item is deleted.
	//
	Hidden  bool   // whether the story has been hidden or not
	TimeStr string // time in cool string format
	Domain  string // the URL's domain
}

func New() Post {
	return Post{
		Id:     loadingId,
		Hidden: false,
	}
}

func FromJSON(bytes []byte) Post {
	data := Post{}

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
	jsonparser.EachKey(bytes, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
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
			data.TimeStr = timestampToString(int64(data.Time))
		case 3:
			v, _ := jsonparser.ParseString(value)
			data.Storytype = v
		case 4:
			v, _ := jsonparser.ParseString(value)
			data.Title = v
		case 5:
			v, _ := jsonparser.ParseString(value)
			// default to C-like syntax
			v = strings.ReplaceAll(v, "<code>", "<code class=\"language-c\">")
			data.Text, err = html2mdConverter.ConvertString(v)
			if err != nil {
				// fallback
				data.Text = html.UnescapeString(v)
			} else {
				//data.Text = strings.ReplaceAll(data.Text, "\n\n", "\n")
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
			data.Domain = domainFromURL(v)
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

	return data
}

func (st *Post) KidCount() int {
	return len(st.Kids)
}

func (st *Post) HasKids() bool {
	return len(st.Kids) > 0
}

func (st *Post) HasUrl() bool {
	return len(st.Url) > 0
}

func (st *Post) HasText() bool {
	return len(st.Text) > 0
}

func (st *Post) IsLoaded() bool {
	return st.Id > 0
}

func (st *Post) ToggleHidden() {
	st.Hidden = !st.Hidden
}

// View
func (st *Post) deletedView(highlight bool, w int) string {
	return style.SecondaryStyle.Copy().
		Bold(highlight).
		MaxWidth(w).
		Render(fmt.Sprintf("[deleted] %s", st.TimeStr))
}

func (st *Post) hiddenView(highlight bool, w int) string {
	return style.SecondaryStyle.Copy().
		Bold(highlight).
		MaxWidth(w).
		Render(fmt.Sprintf("(hidden) %s %s", st.By, st.TimeStr))
}

func (st *Post) loadingView(highlight bool, w int, spinner *spinner.Model) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		spinner.View(),
		" ",
		style.SecondaryStyle.Copy().
			Bold(highlight).
			MaxWidth(w).
			Render("Loading...\n..."),
	)
}

func (st *Post) commentView(highlight bool, w int) string {
	// -1 so wordwrap doesn't feel like ignoring the wrap
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(style.MdStyleConfig),
		glamour.WithEmoji(),
		glamour.WithWordWrap(w-1),
	)
	commentTxt, _ := mdRenderer.Render(st.Text)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		style.SecondaryStyle.Copy().
			Bold(highlight).
			MaxWidth(w).
			Render(st.By+" "+st.TimeStr),
		strings.TrimRight(commentTxt, "\n"),
	)
}

func (st *Post) pollOptView(highlight bool, w int) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		style.PrimaryStyle.Copy().
			Bold(highlight).
			Width(w).
			Render(st.Text),
		style.SecondaryStyle.Copy().
			Bold(highlight).
			MaxWidth(w).
			Render(fmt.Sprintf("%d points", st.Score)),
	)
}

func (st *Post) View(highlight bool, selected bool, w int, stories map[int]*Post, spinner *spinner.Model) string {
	if st.Deleted || st.Dead {
		// deleted story
		return st.deletedView(highlight, w)
	}

	if st.Hidden && !selected {
		// hidden post (and not selected (parent))
		return st.hiddenView(highlight, w)
	}

	if st.Id < 0 {
		// still loading/hasn't started loading
		return st.loadingView(highlight, w, spinner)
	}

	switch st.Storytype {
	case "comment":
		return st.commentView(highlight, w)
	case "pollopt":
		return st.pollOptView(highlight, w)
	default:
		// title should wrap if needed, but leave space for domain if possible
		stTitleStyle := style.PrimaryStyle.Copy().Bold(highlight)
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
					style.UrlStyle.Copy().
						Bold(highlight).
						MaxWidth(w).
						Render(fmt.Sprintf("(%s)", st.Domain)),
				)
			} else {
				row = lipgloss.JoinHorizontal(
					lipgloss.Top,
					row,
					style.UrlStyle.Copy().
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
					glamour.WithStyles(style.MdStyleConfig),
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
					pollOpt := stories[st.Parts[i]]
					row = lipgloss.JoinVertical(
						lipgloss.Left,
						row,
						lipgloss.JoinHorizontal(
							lipgloss.Top,
							"  ",
							pollOpt.View(false, false, w-2, stories, spinner),
						),
					)
				}
			}
		}

		row = lipgloss.JoinVertical(
			lipgloss.Left,
			row,
			style.SecondaryStyle.Copy().
				Bold(highlight).
				MaxWidth(w).
				Render(
					fmt.Sprintf("%d points by %s %s | %d comments", st.Score, st.By, st.TimeStr, st.Descendants),
				),
		)

		return row
	}
}
