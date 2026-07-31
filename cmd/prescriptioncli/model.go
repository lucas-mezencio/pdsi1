package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type stage int

const (
	stageLogin stage = iota
	stageForm
	stageConfirm
	stageSubmitting
	stageDone
)

func (s stage) String() string {
	switch s {
	case stageLogin:
		return "login"
	case stageForm:
		return "form"
	case stageConfirm:
		return "confirm"
	case stageSubmitting:
		return "submitting"
	case stageDone:
		return "done"
	default:
		return "unknown"
	}
}

type sessionData struct {
	UserID    string
	Name      string
	Dosage    string
	Frequency string
	StartTime string
	Doses     int
}

type Model struct {
	stage stage
	data  sessionData
	api   *API
	width int
	err   error

	login  loginScreen
	form   formScreen
	confirm  confirmScreen
	submit submittingScreen
	done   doneScreen

	quitting bool
}

func NewModel(api *API) Model {
	return Model{
		stage: stageLogin,
		api:   api,
		login: newLoginScreen(),
		form:  newFormScreen(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	case stageChangeMsg:
		m.stage = msg.to
		switch m.stage {
		case stageForm:
			m.form = newFormScreen()
		case stageConfirm:
			m.confirm = newConfirmScreen(m.data)
			m.confirm.userID = m.data.UserID
			m.confirm.medicID = currentMedicID()
		case stageSubmitting:
			m.submit = newSubmittingScreen()
		}
		return m, nil
	case loginSuccessMsg:
		m.data.UserID = msg.userID
		return m, transitionMsg(stageForm)
	case formSubmitMsg:
		m.data.Name = msg.data.Name
		m.data.Dosage = msg.data.Dosage
		m.data.Frequency = msg.data.Frequency
		m.data.StartTime = msg.data.StartTime
		m.data.Doses = msg.data.Doses
		return m, transitionMsg(stageConfirm)
	case prescriptionCreatedMsg:
		shifted, _ := shiftStartTime(m.data.StartTime)
		m.done = doneScreen{success: true, id: msg.resp.ID.String(), shifted: shifted}
		return m, transitionMsg(stageDone)
	case prescriptionFailedMsg:
		m.done = doneScreen{success: false, err: msg.err.Error()}
		return m, transitionMsg(stageDone)
	}

	var cmd tea.Cmd
	switch m.stage {
	case stageLogin:
		m.login, cmd = m.login.update(msg)
	case stageForm:
		m.form, cmd = m.form.update(msg)
	case stageConfirm:
		m.confirm, cmd = m.confirm.update(msg)
	case stageSubmitting:
		m.submit, cmd = m.submit.update(msg)
	case stageDone:
		m.done, cmd = m.done.update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.stage {
	case stageLogin:
		return m.login.view()
	case stageForm:
		return m.form.view()
	case stageConfirm:
		return m.confirm.view()
	case stageSubmitting:
		return m.submit.view()
	case stageDone:
		return m.done.view()
	default:
		return StyleTitle.Render("prescriptioncli")
	}
}

// currentAPIPtr / currentMedicIDPtr / Set* are the package-level indirection
// used by screen-local tea.Cmd functions (loginCmd, submitCmd) to access the
// active API handle and doctor UUID without threading them through every
// per-screen update method.

var currentAPIPtr **API

func currentAPI() *API {
	if currentAPIPtr == nil || *currentAPIPtr == nil {
		panic("prescriptioncli: currentAPI called before SetCurrentAPI")
	}
	return *currentAPIPtr
}

func SetCurrentAPI(p **API) { currentAPIPtr = p }

var currentMedicIDPtr *string

func currentMedicID() string {
	if currentMedicIDPtr == nil {
		return defaultMedicID
	}
	return *currentMedicIDPtr
}

func SetCurrentMedicID(p *string) { currentMedicIDPtr = p }

type stageChangeMsg struct{ to stage }

func transitionMsg(to stage) tea.Cmd {
	return func() tea.Msg { return stageChangeMsg{to: to} }
}
