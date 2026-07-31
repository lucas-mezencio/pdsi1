package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

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

type formScreen struct {
	inputs  [5]textinput.Model
	focused int
	err     string
}

func newFormScreen() formScreen {
	placeholders := [5]string{
		"e.g. Aspirin",
		"e.g. 100mg",
		"e.g. 24:00",
		"e.g. 08:00",
		"e.g. 1",
	}
	var inputs [5]textinput.Model
	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 64
		inputs[i] = ti
	}
	inputs[0].Focus()
	return formScreen{inputs: inputs, focused: 0}
}

func (s *formScreen) validate() (sessionData, error) {
	name := s.inputs[0].Value()
	dose := s.inputs[1].Value()
	freq := s.inputs[2].Value()
	start := s.inputs[3].Value()
	dosesStr := s.inputs[4].Value()

	if name == "" || dose == "" || freq == "" || start == "" || dosesStr == "" {
		return sessionData{}, errors.New("all fields are required")
	}
	if !hhmmRe.MatchString(freq) {
		return sessionData{}, fmt.Errorf("frequency must match HH:MM (24h), got %q", freq)
	}
	if !hhmmRe.MatchString(start) {
		return sessionData{}, fmt.Errorf("start time must match HH:MM (24h), got %q", start)
	}
	var n int
	if _, err := fmt.Sscanf(dosesStr, "%d", &n); err != nil || n <= 0 {
		return sessionData{}, fmt.Errorf("doses must be a positive integer, got %q", dosesStr)
	}
	return sessionData{Name: name, Dosage: dose, Frequency: freq, StartTime: start, Doses: n}, nil
}

func (s formScreen) update(msg tea.Msg) (formScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			s.focused = (s.focused + 1) % 5
		case "shift+tab", "up":
			s.focused = (s.focused + 4) % 5
		case "enter":
			if s.focused < 4 {
				s.focused++
				s.inputs[s.focused-1].Blur()
				s.inputs[s.focused].Focus()
				return s, nil
			}
			data, err := s.validate()
			if err != nil {
				s.err = err.Error()
				return s, nil
			}
			s.err = ""
			return s, formSubmitCmd(data)
		}
	}

	for i := range s.inputs {
		if i == s.focused {
			s.inputs[i].Focus()
		} else {
			s.inputs[i].Blur()
		}
	}
	var cmd tea.Cmd
	s.inputs[s.focused], cmd = s.inputs[s.focused].Update(msg)
	return s, cmd
}

func formSubmitCmd(d sessionData) tea.Cmd {
	return func() tea.Msg { return formSubmitMsg{data: d} }
}

func (s formScreen) view() string {
	var body string
	labels := [5]string{
		"Medication name",
		"Dose",
		"Frequency (HH:MM)",
		"Start time HH:MM (+3h on submit)",
		"Doses",
	}
	for i := 0; i < 5; i++ {
		body += StyleFieldLabel.Render(labels[i]) + "\n"
		body += s.inputs[i].View() + "\n"
	}
	if s.err != "" {
		body += "\n" + StyleError.Render(s.err)
	}
	body += "\n" + StyleSubtle.Render("tab/enter to advance · ctrl+c to quit")
	return StyleTitle.Render("prescriptioncli — new prescription") + "\n" + body
}

type confirmScreen struct {
	data    sessionData
	userID  string
	medicID string
	shifted string
	err     error
}

func newConfirmScreen(d sessionData) confirmScreen {
	shifted, err := shiftStartTime(d.StartTime)
	return confirmScreen{data: d, shifted: shifted, err: err}
}

func (s confirmScreen) update(msg tea.Msg) (confirmScreen, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "y", "Y":
			if s.err != nil {
				return s, nil
			}
			return s, tea.Batch(transitionMsg(stageSubmitting), submitCmd(currentAPI(), s.medicID, s.userID, s.data))
		case "n", "N", "esc":
			return s, transitionMsg(stageForm)
		}
	}
	return s, nil
}

func (s confirmScreen) view() string {
	if s.err != nil {
		return StyleError.Render(fmt.Sprintf("invalid start time: %v", s.err))
	}
	summary := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s\n%s %s (+3h -> %s)\n%s %d\n%s %s\n%s %s",
		StyleSummaryKey.Render("Medication:"), StyleSummaryValue.Render(s.data.Name),
		StyleSummaryKey.Render("Dose:"), StyleSummaryValue.Render(s.data.Dosage),
		StyleSummaryKey.Render("Frequency:"), StyleSummaryValue.Render(s.data.Frequency),
		StyleSummaryKey.Render("Start time:"), StyleSummaryValue.Render(s.data.StartTime), StyleSummaryValue.Render(s.shifted),
		StyleSummaryKey.Render("Doses:"), s.data.Doses,
		StyleSummaryKey.Render("User ID:"), StyleSummaryValue.Render(s.userID),
		StyleSummaryKey.Render("Doctor ID:"), StyleSummaryValue.Render(s.medicID),
	)
	return StyleTitle.Render("prescriptioncli — confirm") + "\n" +
		StyleBox.Render(summary) + "\n" +
		StyleSubtle.Render("y to submit · n to go back · ctrl+c to quit")
}

type submitRef struct {
	medicID string
	userID  string
	data    sessionData
}

func submitCmd(api *API, medicID, userID string, d sessionData) tea.Cmd {
	return func() tea.Msg {
		resp, err := api.CreatePrescription(context.Background(), Prescription{
			UserID:    userID,
			MedicID:   medicID,
			Name:      d.Name,
			Dosage:    d.Dosage,
			Frequency: d.Frequency,
			StartTime: d.StartTime,
			Doses:     d.Doses,
		})
		if err != nil {
			return prescriptionFailedMsg{err: err}
		}
		return prescriptionCreatedMsg{resp: resp}
	}
}

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

func (s doneScreen) update(msg tea.Msg) (doneScreen, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "q", "esc", "ctrl+c":
			return s, tea.Quit
		}
		if !s.success && (km.String() == "r" || km.String() == "R") {
			return s, transitionMsg(stageForm)
		}
	}
	return s, nil
}

func (s doneScreen) view() string {
	if s.success {
		body := StyleSuccess.Render("Prescription created!") + "\n\n"
		body += StyleSummaryKey.Render("ID:") + " " + s.id + "\n"
		body += StyleSummaryKey.Render("Scheduled:") + " " + s.shifted + "\n"
		body += "\n" + StyleSubtle.Render("press q to quit")
		return StyleTitle.Render("prescriptioncli — done") + "\n" + StyleBox.Render(body)
	}
	body := StyleError.Render("Failed to create prescription") + "\n\n"
	body += s.err + "\n\n"
	body += StyleSubtle.Render("r to retry · q to quit")
	return StyleTitle.Render("prescriptioncli — error") + "\n" + StyleBox.Render(body)
}
