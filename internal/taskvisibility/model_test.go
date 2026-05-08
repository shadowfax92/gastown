package taskvisibility

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/beads"
)

func TestBuildDistinguishesAllTaskStatuses(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	old := now.Add(-2 * time.Hour)

	model := Build(Snapshot{
		Now:        now,
		StaleAfter: time.Hour,
		Issues: []Issue{
			{ID: "gt-ready", Title: "Ready", Status: "open", Priority: 2, UpdatedAt: recent},
			{ID: "gt-assigned", Title: "Assigned", Status: "hooked", Assignee: "gastown/polecats/nux", UpdatedAt: recent},
			{ID: "gt-working", Title: "Working", Status: "in_progress", Assignee: "gastown/polecats/ivy", UpdatedAt: recent},
			{ID: "gt-blocked", Title: "Blocked", Status: "open", BlockedBy: []string{"gt-dep"}, BlockedByCount: 1, UpdatedAt: recent},
			{ID: "gt-review", Title: "Review", Status: "open", UpdatedAt: recent},
			{ID: "gt-merging", Title: "Merging", Status: "hooked", UpdatedAt: recent},
			{ID: "gt-completed", Title: "Completed", Status: "closed", ClosedAt: recent, UpdatedAt: recent},
			{ID: "gt-failed", Title: "Failed", Status: "open", UpdatedAt: recent},
			{ID: "gt-stale", Title: "Stale", Status: "hooked", Assignee: "gastown/polecats/old", UpdatedAt: old},
		},
		Hooks: []Hook{
			{IssueID: "gt-assigned", Assignee: "gastown/polecats/nux", UpdatedAt: recent},
			{IssueID: "gt-stale", Assignee: "gastown/polecats/old", UpdatedAt: old},
		},
		Workers: []Worker{
			{Agent: "gastown/polecats/ivy", IssueID: "gt-working", State: "working", WorkStatus: "working", LastActivity: recent},
		},
		MergeRequests: []MergeRequest{
			{ID: "gt-mr-merge", SourceIssue: "gt-merging", Status: "open", UpdatedAt: recent},
			{ID: "gt-mr-fail", SourceIssue: "gt-failed", Status: "open", Phase: "failed", UpdatedAt: recent},
		},
		PullRequests: []PullRequest{
			{ID: "17", SourceIssue: "gt-review", State: "open", ReviewState: "PENDING", UpdatedAt: recent},
		},
	})

	want := map[string]Status{
		"gt-ready":     StatusReady,
		"gt-assigned":  StatusAssigned,
		"gt-working":   StatusWorking,
		"gt-blocked":   StatusBlocked,
		"gt-review":    StatusReview,
		"gt-merging":   StatusMerging,
		"gt-completed": StatusCompleted,
		"gt-failed":    StatusFailed,
		"gt-stale":     StatusStale,
	}

	for id, wantStatus := range want {
		task := findTask(t, model, id)
		if task.Status != wantStatus {
			t.Fatalf("%s status = %q (%s), want %q", id, task.Status, task.Reason, wantStatus)
		}
		if model.Count(wantStatus) == 0 {
			t.Fatalf("summary count for %q was not incremented", wantStatus)
		}
	}
}

func TestBuildPreservesConvoyMembership(t *testing.T) {
	model := Build(Snapshot{
		Now: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		Issues: []Issue{
			{ID: "gt-task", Title: "Tracked", Status: "open"},
		},
		Convoys: []Convoy{
			{ID: "gt-convoy", Title: "Launch", Status: "open", IssueIDs: []string{"gt-task"}},
		},
	})

	task := findTask(t, model, "gt-task")
	if len(task.Convoys) != 1 {
		t.Fatalf("convoys = %d, want 1", len(task.Convoys))
	}
	if task.Convoys[0].ID != "gt-convoy" || task.Convoys[0].Title != "Launch" {
		t.Fatalf("unexpected convoy ref: %+v", task.Convoys[0])
	}
}

func TestAdaptersFromBeads(t *testing.T) {
	issue := &beads.Issue{
		ID:          "gt-task",
		Title:       "Implement model",
		Type:        "task",
		Status:      "hooked",
		Priority:    1,
		Assignee:    "gastown/polecats/nux",
		Labels:      []string{"visibility"},
		BlockedBy:   []string{"gt-blocker"},
		UpdatedAt:   "2026-05-08T12:00:00Z",
		ClosedAt:    "2026-05-08T12:30:00Z",
		Description: "Task",
	}

	got := IssueFromBead(issue)
	if got.ID != issue.ID || got.Title != issue.Title || got.Status != issue.Status {
		t.Fatalf("IssueFromBead = %+v", got)
	}
	if got.UpdatedAt.IsZero() || got.ClosedAt.IsZero() {
		t.Fatalf("IssueFromBead did not parse timestamps: %+v", got)
	}
	if len(got.Labels) != 1 || len(got.BlockedBy) != 1 {
		t.Fatalf("IssueFromBead did not copy labels/blockers: %+v", got)
	}

	agent := &beads.Issue{
		ID: "gt-gastown-polecat-nux",
		Description: beads.FormatAgentDescription("Polecat nux", &beads.AgentFields{
			RoleType:   "polecat",
			Rig:        "gastown",
			AgentState: "working",
			HookBead:   "gt-task",
			ActiveMR:   "gt-mr",
			MRFailed:   true,
		}),
	}
	worker := WorkerFromAgentBead(agent)
	if worker.Agent != "gastown/polecats/nux" || worker.IssueID != "gt-task" || worker.ActiveMR != "gt-mr" || !worker.MRFailed {
		t.Fatalf("WorkerFromAgentBead = %+v", worker)
	}

	metadata, _ := json.Marshal(map[string]any{"phase": "merging"})
	mrIssue := &beads.Issue{
		ID:        "gt-mr",
		Title:     "Merge gt-task",
		Status:    "in_progress",
		Assignee:  "gastown/refinery",
		BlockedBy: []string{"gt-gate"},
		UpdatedAt: "2026-05-08T12:45:00Z",
		Metadata:  metadata,
		Description: beads.FormatMRFields(&beads.MRFields{
			SourceIssue: "gt-task",
			Worker:      "gastown/polecats/nux",
			Branch:      "polecat/nux/gt-task",
			Target:      "main",
			CloseReason: "merged",
		}),
	}
	mr := MergeRequestFromBead(mrIssue)
	if mr.ID != "gt-mr" || mr.SourceIssue != "gt-task" || mr.Phase != "merging" || !mr.Blocked {
		t.Fatalf("MergeRequestFromBead = %+v", mr)
	}
	if mr.Branch != "polecat/nux/gt-task" || mr.Target != "main" || mr.CloseReason != "merged" {
		t.Fatalf("MergeRequestFromBead did not parse MR fields: %+v", mr)
	}
}

func TestFailedBeatsStaleAndCompleted(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	model := Build(Snapshot{
		Now:        now,
		StaleAfter: time.Hour,
		Issues: []Issue{
			{ID: "gt-task", Status: "closed", UpdatedAt: now.Add(-2 * time.Hour)},
		},
		MergeRequests: []MergeRequest{
			{ID: "gt-mr", SourceIssue: "gt-task", Status: "open", Phase: "failed", UpdatedAt: now.Add(-2 * time.Hour)},
		},
	})

	task := findTask(t, model, "gt-task")
	if task.Status != StatusFailed {
		t.Fatalf("status = %q, want %q", task.Status, StatusFailed)
	}
}

func TestProviderStatusAliases(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

	model := Build(Snapshot{
		Now: now,
		Issues: []Issue{
			{ID: "gt-ci-failed", Status: "open"},
			{ID: "gt-conflict", Status: "open"},
			{ID: "gt-pr-merged", Status: "open"},
		},
		MergeRequests: []MergeRequest{
			{ID: "gt-mr-ci", SourceIssue: "gt-ci-failed", Status: "open", CIStatus: "FAILURE", UpdatedAt: now},
			{ID: "gt-mr-conflict", SourceIssue: "gt-conflict", Status: "open", Mergeable: "CONFLICTING", UpdatedAt: now},
		},
		PullRequests: []PullRequest{
			{ID: "42", SourceIssue: "gt-pr-merged", State: "closed", Mergeable: "merged", UpdatedAt: now},
		},
	})

	if task := findTask(t, model, "gt-ci-failed"); task.Status != StatusFailed {
		t.Fatalf("CI failure alias status = %q, want %q", task.Status, StatusFailed)
	}
	if task := findTask(t, model, "gt-conflict"); task.Status != StatusFailed {
		t.Fatalf("conflict alias status = %q, want %q", task.Status, StatusFailed)
	}
	if task := findTask(t, model, "gt-pr-merged"); task.Status != StatusCompleted {
		t.Fatalf("merged PR status = %q, want %q", task.Status, StatusCompleted)
	}
}

func findTask(t *testing.T, model Model, id string) Task {
	t.Helper()
	for _, task := range model.Tasks {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("task %s not found in %+v", id, model.Tasks)
	return Task{}
}
