package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyTab() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyTab} }

func TestLoginScreen_InitialFocusIsEmail(t *testing.T) {
	s := newLoginScreen()
	if !s.email.Focused() {
		t.Error("email should be focused initially")
	}
	if s.password.Focused() {
		t.Error("password should not be focused initially")
	}
}

func TestLoginScreen_TabFocusesPassword(t *testing.T) {
	s := newLoginScreen()
	s, _ = s.update(keyTab())
	if s.email.Focused() {
		t.Error("email should be blurred after tab")
	}
	if !s.password.Focused() {
		t.Error("password should be focused after tab")
	}
}

func TestLoginScreen_TabPersistsAcrossUpdates(t *testing.T) {
	s := newLoginScreen()
	s, _ = s.update(keyTab())
	if !s.password.Focused() {
		t.Fatal("precondition: password must be focused after tab")
	}

	s, _ = s.update(tea.KeyMsg{Type: tea.KeyEnter})
	if !s.password.Focused() {
		t.Error("password lost focus on subsequent update — bug regressed")
	}
}

func TestLoginScreen_TypingInFocusedFieldAppends(t *testing.T) {
	s := newLoginScreen()
	s, _ = s.update(keyTab())
	for _, r := range "hunter2" {
		s, _ = s.update(keyRune(r))
	}
	if got := s.password.Value(); got != "hunter2" {
		t.Errorf("password.Value() = %q, want %q", got, "hunter2")
	}
}

func TestLoginScreen_ShiftTabReturnsToEmail(t *testing.T) {
	s := newLoginScreen()
	s, _ = s.update(keyTab())
	s, _ = s.update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if !s.email.Focused() {
		t.Error("shift+tab should re-focus email")
	}
	if s.password.Focused() {
		t.Error("shift+tab should blur password")
	}
}

func TestLoginScreen_ViewDoesNotMutateFocus(t *testing.T) {
	s := newLoginScreen()
	s, _ = s.update(keyTab())
	before := s.email.Focused()
	_ = s.view()
	after := s.email.Focused()
	if before != after {
		t.Error("view() changed email focus state — it must be pure")
	}
}
