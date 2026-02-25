package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/evanschultz/kan/internal/domain"
)

// TestModelWithTeatest verifies behavior for the covered scenario.
func TestModelWithTeatest(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "First task",
		Priority:  domain.PriorityLow,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.Task{task},
	))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "First task")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestModelWithTeatestHelpAndProjectPicker verifies behavior for the covered scenario.
func TestModelWithTeatestHelpAndProjectPicker(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p1, _ := domain.NewProject("p1", "Inbox", "", now)
	p2, _ := domain.NewProject("p2", "Side", "", now)
	c1, _ := domain.NewColumn("c1", p1.ID, "To Do", 0, 0, now)
	c2, _ := domain.NewColumn("c2", p2.ID, "To Do", 0, 0, now)

	m := NewModel(newFakeService(
		[]domain.Project{p1, p2},
		[]domain.Column{c1, c2},
		nil,
	))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Inbox")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: '?', Text: "?"})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "hard delete")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'p', Text: "p"})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Projects")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Side")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestModelGoldenBoardOutput verifies behavior for the covered scenario.
func TestModelGoldenBoardOutput(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   p.ID,
		ColumnID:    c1.ID,
		Position:    0,
		Title:       "Golden board task",
		Description: "golden description",
		Priority:    domain.PriorityMedium,
		Labels:      []string{"alpha", "beta"},
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.Task{task},
	))
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(96, 28),
		teatest.WithProgramOptions(tea.WithEnvironment([]string{"TERM=dumb"})),
	)
	var captured bytes.Buffer
	stream := io.TeeReader(tm.Output(), &captured)

	teatest.WaitFor(t, stream, func(out []byte) bool {
		return strings.Contains(string(out), "Golden board task")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	_, err := io.ReadAll(io.TeeReader(tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second)), &captured))
	if err != nil {
		t.Fatalf("ReadAll(final output) error = %v", err)
	}
	teatest.RequireEqualOutput(t, captured.Bytes())
}

// TestModelGoldenHelpExpandedOutput verifies behavior for the covered scenario.
func TestModelGoldenHelpExpandedOutput(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 0, now)
	task, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "Help Golden Task",
		Priority:  domain.PriorityLow,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.Task{task},
	))
	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(96, 28),
		teatest.WithProgramOptions(tea.WithEnvironment([]string{"TERM=dumb"})),
	)
	var captured bytes.Buffer
	stream := io.TeeReader(tm.Output(), &captured)

	teatest.WaitFor(t, stream, func(out []byte) bool {
		return strings.Contains(string(out), "Help Golden Task")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: '?', Text: "?"})
	teatest.WaitFor(t, stream, func(out []byte) bool {
		return strings.Contains(string(out), "KAN Help")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	_, err := io.ReadAll(io.TeeReader(tm.FinalOutput(t, teatest.WithFinalTimeout(2*time.Second)), &captured))
	if err != nil {
		t.Fatalf("ReadAll(final output) error = %v", err)
	}
	teatest.RequireEqualOutput(t, captured.Bytes())
}

// TestModelWithTeatestWIPWarning verifies behavior for the covered scenario.
func TestModelWithTeatestWIPWarning(t *testing.T) {
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	p, _ := domain.NewProject("p1", "Inbox", "", now)
	c1, _ := domain.NewColumn("c1", p.ID, "To Do", 0, 1, now)
	t1, _ := domain.NewTask(domain.TaskInput{
		ID:        "t1",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  0,
		Title:     "First",
		Priority:  domain.PriorityLow,
	}, now)
	t2, _ := domain.NewTask(domain.TaskInput{
		ID:        "t2",
		ProjectID: p.ID,
		ColumnID:  c1.ID,
		Position:  1,
		Title:     "Second",
		Priority:  domain.PriorityMedium,
	}, now)

	m := NewModel(newFakeService(
		[]domain.Project{p},
		[]domain.Column{c1},
		[]domain.Task{t1, t2},
	), WithBoardConfig(BoardConfig{
		ShowWIPWarnings: true,
		GroupBy:         "none",
	}))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(140, 35))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "WIP limit exceeded")
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
