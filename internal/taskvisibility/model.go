// Package taskvisibility normalizes Gas Town work signals into a dashboard read model.
package taskvisibility

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/constants"
)

// Status is the normalized lifecycle state for founder-facing task visibility.
type Status string

const (
	StatusReady     Status = "ready"
	StatusAssigned  Status = "assigned"
	StatusWorking   Status = "working"
	StatusBlocked   Status = "blocked"
	StatusReview    Status = "review"
	StatusMerging   Status = "merging"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusStale     Status = "stale"
)

// DefaultStaleAfter is used when Snapshot.StaleAfter is unset.
const DefaultStaleAfter = time.Hour

// Snapshot is the raw current-state input for the read model.
type Snapshot struct {
	Now           time.Time
	StaleAfter    time.Duration
	Issues        []Issue
	Hooks         []Hook
	Convoys       []Convoy
	Workers       []Worker
	MergeRequests []MergeRequest
	PullRequests  []PullRequest
}

// Issue is the beads issue subset needed for task visibility.
type Issue struct {
	ID             string
	Title          string
	Type           string
	Status         string
	Priority       int
	Assignee       string
	Labels         []string
	BlockedBy      []string
	BlockedByCount int
	UpdatedAt      time.Time
	ClosedAt       time.Time
}

// Hook is work pinned to an agent.
type Hook struct {
	IssueID   string
	Assignee  string
	UpdatedAt time.Time
	Stale     bool
}

// Convoy groups tracked issues.
type Convoy struct {
	ID       string
	Title    string
	Status   string
	IssueIDs []string
}

// Worker is the agent/polecat runtime signal for a task.
type Worker struct {
	Agent        string
	Role         string
	State        string
	WorkStatus   string
	IssueID      string
	HookIssueID  string
	ActiveMR     string
	LastActivity time.Time
	MRFailed     bool
	PushFailed   bool
	Stale        bool
}

// MergeRequest is merge-queue evidence linked to a source task.
type MergeRequest struct {
	ID          string
	Title       string
	SourceIssue string
	Status      string
	Phase       string
	Assignee    string
	Worker      string
	Branch      string
	Target      string
	URL         string
	CIStatus    string
	Mergeable   string
	CloseReason string
	Error       string
	Blocked     bool
	BlockedBy   []string
	UpdatedAt   time.Time
}

// PullRequest is external PR review evidence linked to a source task.
type PullRequest struct {
	ID          string
	Title       string
	SourceIssue string
	State       string
	ReviewState string
	CIStatus    string
	Mergeable   string
	URL         string
	Draft       bool
	Stale       bool
	UpdatedAt   time.Time
}

// Model is the normalized task visibility read model.
type Model struct {
	Tasks   []Task
	Summary Summary
}

// Task is one normalized work item with the source evidence preserved.
type Task struct {
	ID            string
	Title         string
	Type          string
	Priority      int
	Assignee      string
	Status        Status
	Reason        string
	UpdatedAt     time.Time
	Issue         *Issue
	Hooks         []Hook
	Convoys       []ConvoyRef
	Workers       []Worker
	MergeRequests []MergeRequest
	PullRequests  []PullRequest
}

// ConvoyRef is a task's membership in a convoy.
type ConvoyRef struct {
	ID     string
	Title  string
	Status string
}

// Summary counts tasks by normalized state.
type Summary struct {
	Ready     int
	Assigned  int
	Working   int
	Blocked   int
	Review    int
	Merging   int
	Completed int
	Failed    int
	Stale     int
}

// Build normalizes the current task visibility snapshot.
func Build(snapshot Snapshot) Model {
	if snapshot.Now.IsZero() {
		snapshot.Now = time.Now()
	}
	if snapshot.StaleAfter == 0 {
		snapshot.StaleAfter = DefaultStaleAfter
	}

	builders := make(map[string]*taskBuilder)
	ensure := func(id string) *taskBuilder {
		id = strings.TrimSpace(id)
		if id == "" {
			id = "(unknown)"
		}
		if existing := builders[id]; existing != nil {
			return existing
		}
		b := &taskBuilder{task: Task{ID: id}}
		builders[id] = b
		return b
	}

	for _, issue := range snapshot.Issues {
		if strings.TrimSpace(issue.ID) == "" {
			continue
		}
		b := ensure(issue.ID)
		issueCopy := issue
		b.hasIssue = true
		b.task.Issue = &issueCopy
		b.task.Title = firstNonEmpty(issue.Title, b.task.Title)
		b.task.Type = firstNonEmpty(issue.Type, b.task.Type)
		b.task.Priority = firstNonZero(issue.Priority, b.task.Priority)
		b.task.Assignee = firstNonEmpty(issue.Assignee, b.task.Assignee)
		b.task.UpdatedAt = latestTime(b.task.UpdatedAt, issue.UpdatedAt, issue.ClosedAt)
	}

	for _, hook := range snapshot.Hooks {
		if strings.TrimSpace(hook.IssueID) == "" {
			continue
		}
		b := ensure(hook.IssueID)
		b.task.Hooks = append(b.task.Hooks, hook)
		b.task.Assignee = firstNonEmpty(hook.Assignee, b.task.Assignee)
		b.task.UpdatedAt = latestTime(b.task.UpdatedAt, hook.UpdatedAt)
	}

	for _, convoy := range snapshot.Convoys {
		ref := ConvoyRef{ID: convoy.ID, Title: convoy.Title, Status: convoy.Status}
		for _, issueID := range convoy.IssueIDs {
			if strings.TrimSpace(issueID) == "" {
				continue
			}
			b := ensure(issueID)
			b.task.Convoys = append(b.task.Convoys, ref)
		}
	}

	for _, worker := range snapshot.Workers {
		issueID := firstNonEmpty(worker.IssueID, worker.HookIssueID)
		if strings.TrimSpace(issueID) == "" {
			continue
		}
		b := ensure(issueID)
		b.task.Workers = append(b.task.Workers, worker)
		b.task.Assignee = firstNonEmpty(worker.Agent, b.task.Assignee)
		b.task.UpdatedAt = latestTime(b.task.UpdatedAt, worker.LastActivity)
	}

	for _, mr := range snapshot.MergeRequests {
		issueID := firstNonEmpty(mr.SourceIssue, mr.ID)
		if strings.TrimSpace(issueID) == "" {
			continue
		}
		b := ensure(issueID)
		b.task.MergeRequests = append(b.task.MergeRequests, mr)
		b.task.Assignee = firstNonEmpty(mr.Worker, mr.Assignee, b.task.Assignee)
		b.task.UpdatedAt = latestTime(b.task.UpdatedAt, mr.UpdatedAt)
	}

	for _, pr := range snapshot.PullRequests {
		issueID := firstNonEmpty(pr.SourceIssue, pr.ID)
		if strings.TrimSpace(issueID) == "" {
			continue
		}
		b := ensure(issueID)
		b.task.PullRequests = append(b.task.PullRequests, pr)
		b.task.UpdatedAt = latestTime(b.task.UpdatedAt, pr.UpdatedAt)
	}

	model := Model{Tasks: make([]Task, 0, len(builders))}
	for _, b := range builders {
		b.task.Status, b.task.Reason = classify(b, snapshot.Now, snapshot.StaleAfter)
		model.Summary.add(b.task.Status)
		model.Tasks = append(model.Tasks, b.task)
	}

	sort.SliceStable(model.Tasks, func(i, j int) bool {
		left, right := model.Tasks[i], model.Tasks[j]
		if StatusRank(left.Status) != StatusRank(right.Status) {
			return StatusRank(left.Status) < StatusRank(right.Status)
		}
		leftPriority, rightPriority := normalizedPriority(left.Priority), normalizedPriority(right.Priority)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return left.ID < right.ID
	})

	return model
}

// Count returns the number of tasks in the requested state.
func (m Model) Count(status Status) int {
	return m.Summary.Count(status)
}

// Count returns the number of tasks in the requested state.
func (s Summary) Count(status Status) int {
	switch status {
	case StatusReady:
		return s.Ready
	case StatusAssigned:
		return s.Assigned
	case StatusWorking:
		return s.Working
	case StatusBlocked:
		return s.Blocked
	case StatusReview:
		return s.Review
	case StatusMerging:
		return s.Merging
	case StatusCompleted:
		return s.Completed
	case StatusFailed:
		return s.Failed
	case StatusStale:
		return s.Stale
	default:
		return 0
	}
}

// StatusRank gives active and attention-needed work a stable display order.
func StatusRank(status Status) int {
	switch status {
	case StatusFailed:
		return 0
	case StatusStale:
		return 1
	case StatusBlocked:
		return 2
	case StatusWorking:
		return 3
	case StatusReview:
		return 4
	case StatusMerging:
		return 5
	case StatusAssigned:
		return 6
	case StatusReady:
		return 7
	case StatusCompleted:
		return 8
	default:
		return 9
	}
}

// IssueFromBead adapts a beads issue into the read-model input shape.
func IssueFromBead(issue *beads.Issue) Issue {
	if issue == nil {
		return Issue{}
	}
	return Issue{
		ID:             issue.ID,
		Title:          issue.Title,
		Type:           issue.Type,
		Status:         issue.Status,
		Priority:       issue.Priority,
		Assignee:       issue.Assignee,
		Labels:         append([]string(nil), issue.Labels...),
		BlockedBy:      append([]string(nil), issue.BlockedBy...),
		BlockedByCount: issue.BlockedByCount,
		UpdatedAt:      parseBeadTime(issue.UpdatedAt),
		ClosedAt:       parseBeadTime(issue.ClosedAt),
	}
}

// WorkerFromAgentBead adapts an agent bead into worker evidence.
func WorkerFromAgentBead(issue *beads.Issue) Worker {
	if issue == nil {
		return Worker{}
	}
	fields := beads.ParseAgentFields(issue.Description)
	if fields == nil {
		return Worker{}
	}
	rig, role, name, _ := beads.ParseAgentBeadID(issue.ID)
	if fields.Rig != "" {
		rig = fields.Rig
	}
	if fields.RoleType != "" {
		role = fields.RoleType
	}
	hookBead := firstNonEmpty(fields.HookBead, issue.HookBead)
	return Worker{
		Agent:       agentAddress(rig, role, name, issue.ID),
		Role:        role,
		State:       beads.ResolveAgentState(issue.Description, issue.AgentState),
		IssueID:     hookBead,
		HookIssueID: hookBead,
		ActiveMR:    fields.ActiveMR,
		MRFailed:    fields.MRFailed,
		PushFailed:  fields.PushFailed,
	}
}

// MergeRequestFromBead adapts a gt:merge-request bead into merge evidence.
func MergeRequestFromBead(issue *beads.Issue) MergeRequest {
	if issue == nil {
		return MergeRequest{}
	}
	fields := beads.ParseMRFields(issue)
	mr := MergeRequest{
		ID:        issue.ID,
		Title:     issue.Title,
		Status:    issue.Status,
		Assignee:  issue.Assignee,
		Blocked:   issue.BlockedByCount > 0 || len(issue.BlockedBy) > 0,
		BlockedBy: append([]string(nil), issue.BlockedBy...),
		UpdatedAt: parseBeadTime(issue.UpdatedAt),
		Phase:     phaseFromMetadata(issue.Metadata),
	}
	if fields != nil {
		mr.SourceIssue = fields.SourceIssue
		mr.Worker = fields.Worker
		mr.Branch = fields.Branch
		mr.Target = fields.Target
		mr.CloseReason = fields.CloseReason
	}
	return mr
}

type taskBuilder struct {
	task     Task
	hasIssue bool
}

func classify(b *taskBuilder, now time.Time, staleAfter time.Duration) (Status, string) {
	if hasFailedEvidence(b) {
		return StatusFailed, "failure evidence present"
	}
	if isCompleted(b) {
		return StatusCompleted, "issue or merge request completed"
	}
	if isStale(b, now, staleAfter) {
		return StatusStale, "active work is stale"
	}
	if isBlocked(b) {
		return StatusBlocked, "blocked by dependency or merge gate"
	}
	if isMerging(b) {
		return StatusMerging, "merge request queued or processing"
	}
	if isReview(b) {
		return StatusReview, "pull request is in review"
	}
	if isWorking(b) {
		return StatusWorking, "worker is active"
	}
	if isAssigned(b) {
		return StatusAssigned, "hooked or assigned"
	}
	if isReady(b) {
		return StatusReady, "open and unassigned"
	}
	return StatusAssigned, "task has non-ready evidence"
}

func hasFailedEvidence(b *taskBuilder) bool {
	for _, worker := range b.task.Workers {
		if worker.MRFailed || worker.PushFailed {
			return true
		}
		if normalize(worker.State) == "failed" {
			return true
		}
	}
	for _, mr := range b.task.MergeRequests {
		if normalize(mr.Phase) == "failed" || normalize(mr.Status) == "failed" {
			return true
		}
		switch normalize(mr.CloseReason) {
		case "rejected", "conflict", "failed":
			return true
		}
		if strings.TrimSpace(mr.Error) != "" {
			return true
		}
		if isFailStatus(mr.CIStatus) || isConflictStatus(mr.Mergeable) {
			return true
		}
	}
	for _, pr := range b.task.PullRequests {
		if isFailStatus(pr.CIStatus) || isConflictStatus(pr.Mergeable) {
			return true
		}
		if normalize(pr.State) == "closed" && normalize(pr.Mergeable) != "merged" {
			return true
		}
	}
	return false
}

func isCompleted(b *taskBuilder) bool {
	if b.hasIssue && isTerminalStatus(b.task.Issue.Status) {
		return true
	}
	for _, mr := range b.task.MergeRequests {
		if normalize(mr.Phase) == "merged" || normalize(mr.CloseReason) == "merged" {
			return true
		}
		if normalize(mr.Status) == "closed" && normalize(mr.CloseReason) == "" {
			return true
		}
	}
	for _, pr := range b.task.PullRequests {
		state := normalize(pr.State)
		if state == "merged" || (state == "closed" && normalize(pr.Mergeable) == "merged") {
			return true
		}
	}
	return false
}

func isStale(b *taskBuilder, now time.Time, staleAfter time.Duration) bool {
	for _, hook := range b.task.Hooks {
		if hook.Stale || olderThan(now, hook.UpdatedAt, staleAfter) {
			return true
		}
	}
	for _, worker := range b.task.Workers {
		switch normalize(worker.WorkStatus) {
		case "stale", "stuck":
			return true
		}
		switch normalize(worker.State) {
		case "stale", "stuck":
			return true
		}
		if worker.Stale || olderThan(now, worker.LastActivity, staleAfter) {
			return true
		}
	}
	for _, mr := range b.task.MergeRequests {
		if isActiveMR(mr) && olderThan(now, mr.UpdatedAt, staleAfter) {
			return true
		}
	}
	for _, pr := range b.task.PullRequests {
		if pr.Stale || (isOpenPR(pr) && olderThan(now, pr.UpdatedAt, staleAfter)) {
			return true
		}
	}
	if b.hasIssue && isAssignedIssueStatus(b.task.Issue.Status) && olderThan(now, b.task.Issue.UpdatedAt, staleAfter) {
		return true
	}
	return false
}

func isBlocked(b *taskBuilder) bool {
	if b.hasIssue {
		if normalize(b.task.Issue.Status) == "blocked" || b.task.Issue.BlockedByCount > 0 || len(b.task.Issue.BlockedBy) > 0 {
			return true
		}
	}
	for _, mr := range b.task.MergeRequests {
		if mr.Blocked || len(mr.BlockedBy) > 0 {
			return true
		}
	}
	for _, worker := range b.task.Workers {
		switch normalize(worker.State) {
		case "awaiting-gate", "escalated":
			return true
		}
	}
	return false
}

func isMerging(b *taskBuilder) bool {
	for _, mr := range b.task.MergeRequests {
		if isActiveMR(mr) {
			return true
		}
	}
	return false
}

func isReview(b *taskBuilder) bool {
	for _, pr := range b.task.PullRequests {
		if isOpenPR(pr) {
			return true
		}
	}
	return false
}

func isWorking(b *taskBuilder) bool {
	if b.hasIssue && normalize(b.task.Issue.Status) == "in_progress" {
		return true
	}
	for _, worker := range b.task.Workers {
		switch normalize(worker.WorkStatus) {
		case "working", "active":
			return true
		}
		switch normalize(worker.State) {
		case "working", "running", "spawning":
			return true
		}
	}
	return false
}

func isAssigned(b *taskBuilder) bool {
	if b.hasIssue {
		if strings.TrimSpace(b.task.Issue.Assignee) != "" || isAssignedIssueStatus(b.task.Issue.Status) {
			return true
		}
	}
	return len(b.task.Hooks) > 0 || strings.TrimSpace(b.task.Assignee) != "" || len(b.task.Workers) > 0
}

func isReady(b *taskBuilder) bool {
	if !b.hasIssue {
		return len(b.task.Hooks) == 0 && len(b.task.Workers) == 0 && len(b.task.MergeRequests) == 0 && len(b.task.PullRequests) == 0
	}
	return normalize(b.task.Issue.Status) == "open" && strings.TrimSpace(b.task.Issue.Assignee) == "" && !isBlocked(b)
}

func isActiveMR(mr MergeRequest) bool {
	switch normalize(mr.Phase) {
	case "ready", "claimed", "preparing", "prepared", "merging":
		return true
	case "merged", "rejected", "failed":
		return false
	}
	switch normalize(mr.Status) {
	case "open", "in_progress":
		return true
	default:
		return false
	}
}

func isOpenPR(pr PullRequest) bool {
	state := normalize(pr.State)
	return state == "" || state == "open" || state == "draft"
}

func isTerminalStatus(status string) bool {
	switch normalize(status) {
	case "closed", "tombstone":
		return true
	default:
		return false
	}
}

func isFailStatus(status string) bool {
	switch normalize(status) {
	case "fail", "failed", "failure", "error", "cancelled", "canceled", "timed_out", "action_required":
		return true
	default:
		return false
	}
}

func isConflictStatus(status string) bool {
	switch normalize(status) {
	case "conflict", "conflicting", "dirty":
		return true
	default:
		return false
	}
}

func isAssignedIssueStatus(status string) bool {
	switch normalize(status) {
	case "hooked", "in_progress":
		return true
	default:
		return false
	}
}

func olderThan(now, t time.Time, threshold time.Duration) bool {
	if now.IsZero() || t.IsZero() || threshold <= 0 {
		return false
	}
	return now.Sub(t) > threshold
}

func (s *Summary) add(status Status) {
	switch status {
	case StatusReady:
		s.Ready++
	case StatusAssigned:
		s.Assigned++
	case StatusWorking:
		s.Working++
	case StatusBlocked:
		s.Blocked++
	case StatusReview:
		s.Review++
	case StatusMerging:
		s.Merging++
	case StatusCompleted:
		s.Completed++
	case StatusFailed:
		s.Failed++
	case StatusStale:
		s.Stale++
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func latestTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}

func normalizedPriority(priority int) int {
	if priority <= 0 {
		return 5
	}
	return priority
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseBeadTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05Z", value); err == nil {
		return t
	}
	t, _ := time.Parse("2006-01-02", value)
	return t
}

func phaseFromMetadata(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return ""
	}
	for _, key := range []string{"phase", "mr_phase", "merge_phase"} {
		if phase, ok := metadata[key].(string); ok {
			return phase
		}
	}
	return ""
}

func agentAddress(rig, role, name, fallback string) string {
	switch role {
	case constants.RolePolecat:
		if rig != "" && name != "" {
			return rig + "/polecats/" + name
		}
	case constants.RoleCrew:
		if rig != "" && name != "" {
			return rig + "/crew/" + name
		}
	case constants.RoleWitness, constants.RoleRefinery:
		if rig != "" {
			return rig + "/" + role
		}
	case constants.RoleMayor:
		return "mayor/"
	case constants.RoleDeacon:
		return "deacon/"
	}
	return fallback
}
