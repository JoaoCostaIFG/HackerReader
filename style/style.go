package style

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

var (
	// colors
	Primary   = lipgloss.Color("#EEEEEE")
	Secondary = lipgloss.Color("#867f74")
	Black     = lipgloss.Color("#222222")
	Green     = lipgloss.Color("#3ED71C")
	Orange    = lipgloss.Color("#FF6600")
	// title bar
	TitleBar = lipgloss.NewStyle().
			Background(Orange).
			Foreground(Black).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)
	// main item
	MainItemBorder = lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
	}
	MainItem = lipgloss.NewStyle().
			Border(MainItemBorder).
			BorderForeground(Primary)
	// check mark
	Checkmark = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Render
	// list items
	ListItemBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "├",
		BottomRight: "┤",
	}
	ListItem = lipgloss.NewStyle().
			Border(ListItemBorder).
			BorderForeground(Primary)
	// url stuff
	UrlStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true)
	// other
	PrimaryStyle = lipgloss.NewStyle().
			Foreground(Primary)
	SecondaryStyle = lipgloss.NewStyle().
			Foreground(Secondary)
	// spinner
	SpinnerSpinner = spinner.Line
	SpinnerStyle   = lipgloss.NewStyle().
			Foreground(Orange)
	// md
	MdStyleConfig = ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("#EEEEEE"),
			},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
			Indent:         uintPtr(1),
			IndentToken:    stringPtr("│ "),
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{},
			},
			LevelIndent: 2,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "# ",
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "## ",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "### ",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "#### ",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "##### ",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "###### ",
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Format: "---",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "- ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: ansi.StyleTask{
			Ticked:   "[X] ",
			Unticked: "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Underline:   boolPtr(true),
			BlockPrefix: "(",
			BlockSuffix: ")",
		},
		ImageText: ansi.StylePrimitive{
			Format: "Image: {{.text}} →",
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "`",
				BlockSuffix: "`",
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: uintPtr(2),
			},
		},
		Table: ansi.StyleTable{
			CenterSeparator: stringPtr("┼"),
			ColumnSeparator: stringPtr("│"),
			RowSeparator:    stringPtr("─"),
		},
		DefinitionDescription: ansi.StylePrimitive{},
	}
)

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }
