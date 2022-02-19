package style

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

const (
	hnOrange = "#FF6600"
	// dracula: https://github.com/dracula/dracula-theme/
	background = "#282a37"
	foreground = "#f8f8f2"
	secondary  = "#867f74"
	cyan       = "#8be9fd"
	cyan2      = "#6EEFC0"
	green      = "#50fa7b"
	orange     = "#ffb86c"
	pink       = "#ff79c6"
	purple     = "#bd93f9"
	red        = "#ff5555"
	yellow     = "#f1fa8c"
)

var (
	// colors
	ForegroundColor = lipgloss.Color(foreground) // #EEEEEE
	SecondaryColor  = lipgloss.Color(secondary)
	GreenColor      = lipgloss.Color(green) // #3ED71C
	CyanColor       = lipgloss.Color(cyan)
	HNOrange        = lipgloss.Color(hnOrange)
	// title bar
	TitleBar = lipgloss.NewStyle().
			Background(HNOrange).
			Foreground(ForegroundColor).
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
			BorderForeground(CyanColor)
	VoteBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color(cyan2)).
		Render
	// check mark
	Checkmark = lipgloss.NewStyle().
			Foreground(GreenColor).
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
			BorderForeground(ForegroundColor)
	// url stuff
	UrlStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Italic(true)
	// other
	PrimaryStyle = lipgloss.NewStyle().
			Foreground(ForegroundColor)
	SecondaryStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor)
	// spinner
	SpinnerSpinner = spinner.Line
	SpinnerStyle   = lipgloss.NewStyle().
			Foreground(HNOrange)
	// md
	MdStyleConfig = ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(foreground),
			},
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr(purple),
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
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
			Color:  stringPtr(yellow),
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Color: stringPtr(orange),
			Bold:  boolPtr(true),
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
		Image: ansi.StylePrimitive{
			Underline: boolPtr(true),
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
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{
					Color: stringPtr(foreground),
				},
				Error: ansi.StylePrimitive{
					Color: stringPtr(foreground),
				},
				Comment: ansi.StylePrimitive{
					Color: stringPtr(secondary),
				},
				CommentPreproc: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				Keyword: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				KeywordReserved: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				KeywordNamespace: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				KeywordType: ansi.StylePrimitive{
					Color: stringPtr(cyan),
				},
				Operator: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				Punctuation: ansi.StylePrimitive{
					Color: stringPtr(foreground),
				},
				Name: ansi.StylePrimitive{
					Color: stringPtr(cyan),
				},
				NameBuiltin: ansi.StylePrimitive{
					Color: stringPtr(cyan),
				},
				NameTag: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				NameAttribute: ansi.StylePrimitive{
					Color: stringPtr(green),
				},
				NameClass: ansi.StylePrimitive{
					Color: stringPtr(cyan),
				},
				NameConstant: ansi.StylePrimitive{
					Color: stringPtr(purple),
				},
				NameDecorator: ansi.StylePrimitive{
					Color: stringPtr(green),
				},
				NameFunction: ansi.StylePrimitive{
					Color: stringPtr(green),
				},
				LiteralNumber: ansi.StylePrimitive{
					Color: stringPtr(cyan2),
				},
				LiteralString: ansi.StylePrimitive{
					Color: stringPtr(yellow),
				},
				LiteralStringEscape: ansi.StylePrimitive{
					Color: stringPtr(pink),
				},
				GenericDeleted: ansi.StylePrimitive{
					Color: stringPtr(red),
				},
				GenericEmph: ansi.StylePrimitive{
					Color:  stringPtr(yellow),
					Italic: boolPtr(true),
				},
				GenericInserted: ansi.StylePrimitive{
					Color: stringPtr(green),
				},
				GenericStrong: ansi.StylePrimitive{
					Color: stringPtr(orange),
					Bold:  boolPtr(true),
				},
				GenericSubheading: ansi.StylePrimitive{
					Color: stringPtr(purple),
				},
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
