package workspace

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runoshun/git-crew/v2/internal/domain"
	"github.com/runoshun/git-crew/v2/internal/tui"
)

func TestModelSetsActiveRepoOnLoad(t *testing.T) {
	m := New()
	repos := []domain.WorkspaceRepo{{Path: "/repo/a"}, {Path: "/repo/b"}}

	updated, _ := m.Update(MsgReposLoaded{Repos: repos})
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model from Update")
	}

	if model.activeRepo != "/repo/a" {
		t.Fatalf("expected active repo to be first repo, got %q", model.activeRepo)
	}
	if model.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", model.cursor)
	}
	if !model.leftFocused {
		t.Fatalf("expected left pane to be focused")
	}
}

func TestModelFocusSwitchWithTab(t *testing.T) {
	m := New()
	if !m.leftFocused {
		t.Fatalf("expected left pane to be focused")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model from Update")
	}
	if model.leftFocused {
		t.Fatalf("expected focus to move to right pane")
	}
}

func TestModelKeepsActiveRepoOnReload(t *testing.T) {
	m := New()
	m.activeRepo = "/repo/b"
	m.cursor = 0
	repos := []domain.WorkspaceRepo{{Path: "/repo/a"}, {Path: "/repo/b"}}

	updated, _ := m.Update(MsgReposLoaded{Repos: repos})
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model from Update")
	}
	if model.activeRepo != "/repo/b" {
		t.Fatalf("expected active repo to stay /repo/b, got %q", model.activeRepo)
	}
	if model.cursor != 1 {
		t.Fatalf("expected cursor to move to active repo index, got %d", model.cursor)
	}
}

func TestModelLayoutSplitWidths(t *testing.T) {
	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected *Model from Update")
	}

	contentWidth := 160 - appPadding
	expectedLeft := int(float64(contentWidth) * leftPaneRatio)
	if expectedLeft < minPaneWidth {
		expectedLeft = minPaneWidth
	}
	expectedRight := contentWidth - expectedLeft
	if expectedRight < minPaneWidth {
		expectedRight = minPaneWidth
		expectedLeft = contentWidth - expectedRight
		if expectedLeft < minPaneWidth {
			expectedLeft = minPaneWidth
		}
	}

	if model.leftWidth != expectedLeft {
		t.Fatalf("expected left width %d, got %d", expectedLeft, model.leftWidth)
	}
	if model.rightWidth != expectedRight {
		t.Fatalf("expected right width %d, got %d", expectedRight, model.rightWidth)
	}
}

func TestViewRightPaneContentShowsError(t *testing.T) {
	m := New()
	m.activeRepo = "/repo/a"
	m.repoInfos[m.activeRepo] = domain.WorkspaceRepoInfo{
		State:    domain.RepoStateNotFound,
		ErrorMsg: "Path does not exist",
	}

	content := m.viewRightPaneContent()
	if !strings.Contains(content, "Cannot open") {
		t.Fatalf("expected error message to mention cannot open")
	}
	if !strings.Contains(content, "Path does not exist") {
		t.Fatalf("expected error message to include repo error details")
	}
}

func TestWrapRepoCmdPassesThroughNonTuiMsg(t *testing.T) {
	m := New()
	cmd := func() tea.Msg { return otherMsg{} }
	wrapped := m.wrapRepoCmd("/repo/a", cmd)
	if wrapped == nil {
		t.Fatalf("expected wrapped cmd to be non-nil")
	}
	msg := wrapped()
	if msg == nil {
		t.Fatalf("expected wrapped cmd to return a message")
	}
	if _, ok := msg.(otherMsg); !ok {
		t.Fatalf("expected otherMsg to pass through, got %T", msg)
	}
}

func TestWrapRepoCmdWrapsTuiMsg(t *testing.T) {
	m := New()
	cmd := func() tea.Msg { return tui.MsgReloadTasks{} }
	wrapped := m.wrapRepoCmd("/repo/a", cmd)
	if wrapped == nil {
		t.Fatalf("expected wrapped cmd to be non-nil")
	}
	msg := wrapped()
	repoMsg, ok := msg.(RepoMsg)
	if !ok {
		t.Fatalf("expected RepoMsg, got %T", msg)
	}
	if repoMsg.Path != "/repo/a" {
		t.Fatalf("expected repo path /repo/a, got %q", repoMsg.Path)
	}
	if _, ok := repoMsg.Msg.(tui.MsgReloadTasks); !ok {
		t.Fatalf("expected MsgReloadTasks, got %T", repoMsg.Msg)
	}
}

type otherMsg struct{}
