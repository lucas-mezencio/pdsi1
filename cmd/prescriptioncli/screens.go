package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginSuccessMsg struct{ userID string }
type loginErrMsg struct{ err error }
type formSubmitMsg struct{ data sessionData }
type prescriptionCreatedMsg struct{ resp *PrescriptionResponse }
type prescriptionFailedMsg struct{ err error }

type loginScreen struct {
	email    textinput.Model
	password textinput.Model
	focused  int
	err      string
}

func newLoginScreen() loginScreen { return loginScreen{} }

func (s loginScreen) update(msg tea.Msg) (loginScreen, tea.Cmd) { return s, nil }
func (s loginScreen) view() string {
	return StyleTitle.Render("prescriptioncli") + "\nlogin (coming soon)"
}

type formScreen struct{}

func newFormScreen() formScreen { return formScreen{} }

func (s formScreen) update(msg tea.Msg) (formScreen, tea.Cmd) { return s, nil }
func (s formScreen) view() string {
	return StyleTitle.Render("prescriptioncli") + "\nform (coming soon)"
}

type confirmScreen struct {
	data     sessionData
	userID   string
	medicID  string
	shifted  string
	err      error
}

func newConfirmScreen(d sessionData) confirmScreen { return confirmScreen{data: d} }

func (s confirmScreen) update(msg tea.Msg) (confirmScreen, tea.Cmd) { return s, nil }
func (s confirmScreen) view() string                              { return "" }

type submittingScreen struct {
	spinner spinner.Model
}

func newSubmittingScreen() submittingScreen {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	return submittingScreen{spinner: sp}
}

func (s submittingScreen) update(msg tea.Msg) (submittingScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}
func (s submittingScreen) view() string { return "" }

type doneScreen struct {
	success bool
	id      string
	shifted string
	err     string
}

func (s doneScreen) update(msg tea.Msg) (doneScreen, tea.Cmd) { return s, nil }
func (s doneScreen) view() string                             { return "" }
