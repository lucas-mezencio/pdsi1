package main

import (
	"context"

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

func newLoginScreen() loginScreen {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 120
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 128

	return loginScreen{email: email, password: password, focused: 0}
}

func loginCmd(api *API, email, pw string) tea.Cmd {
	return func() tea.Msg {
		id, err := api.Login(context.Background(), email, pw)
		if err != nil {
			return loginErrMsg{err: err}
		}
		return loginSuccessMsg{userID: id}
	}
}

func (s loginScreen) update(msg tea.Msg) (loginScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case loginErrMsg:
		s.err = msg.err.Error()
		s.password.SetValue("")
		s.password.Focus()
		s.focused = 1
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			s.focused = (s.focused + 1) % 2
		case "shift+tab", "up":
			s.focused = (s.focused + 1) % 2
		case "enter":
			if s.email.Value() == "" || s.password.Value() == "" {
				s.err = "email and password are required"
				return s, nil
			}
			s.err = ""
			return s, loginCmd(currentAPI(), s.email.Value(), s.password.Value())
		}
	}

	var cmd tea.Cmd
	if s.focused == 0 {
		s.email, cmd = s.email.Update(msg)
	} else {
		s.password, cmd = s.password.Update(msg)
	}
	return s, cmd
}

func (s loginScreen) view() string {
	if s.focused == 0 {
		s.email.Focus()
		s.password.Blur()
	} else {
		s.email.Blur()
		s.password.Focus()
	}

	var body string
	body += StyleFieldLabel.Render("Email") + "\n"
	body += s.email.View() + "\n"
	body += StyleFieldLabel.Render("Password") + "\n"
	body += s.password.View() + "\n"
	if s.err != "" {
		body += "\n" + StyleError.Render(s.err)
	}
	body += "\n" + StyleSubtle.Render("tab to switch · enter to submit · ctrl+c to quit")

	return StyleTitle.Render("prescriptioncli — login") + "\n" + body
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
