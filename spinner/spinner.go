package spinner

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"hackerreader/style"
)

// This is a wrapper around github.com/charmbracelet/bubbles spinner.
// It keeps track if it was used on the last frame.
// This helps me decide if I can reuse the frame I calculated on the last run
//   of the View function, or if I'll have to recalculate it.
// The rationale is: if nothing was loading on the last frame, the update of
//   the spinner won't have an impact on the frame

type Spinner struct {
	spinner spinner.Model
	enabled bool
}

func New() Spinner {
	ret := Spinner{
		spinner: spinner.New(),
		enabled: false,
	}

	ret.spinner.Spinner = style.SpinnerSpinner
	ret.spinner.Style = style.SpinnerStyle

	return ret
}

func (this *Spinner) Disable() {
	this.enabled = false
}

func (this *Spinner) Enable() {
	this.enabled = true
}

func (this *Spinner) IsEnabled() bool {
	return this.enabled
}

func (this *Spinner) Tick() tea.Msg {
	return this.spinner.Tick()
}

func (this *Spinner) Update(msg tea.Msg) tea.Cmd {
	m, c := this.spinner.Update(msg)
	this.spinner = m
	return c
}

func (this *Spinner) View() string {
	this.Enable() // used in this frame
	return this.spinner.View()
}
