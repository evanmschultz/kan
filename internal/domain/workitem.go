package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// LifecycleState represents canonical lifecycle state values.
type LifecycleState string

// Canonical lifecycle states.
const (
	StateTodo     LifecycleState = "todo"
	StateProgress LifecycleState = "progress"
	StateDone     LifecycleState = "done"
	StateArchived LifecycleState = "archived"
)

// ActorType describes the actor class that last updated an item.
type ActorType string

// ActorType values.
const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeSystem ActorType = "system"
)

// WorkKind represents a configurable item kind.
type WorkKind string

// Built-in kind defaults.
const (
	WorkKindTask     WorkKind = "task"
	WorkKindSubtask  WorkKind = "subtask"
	WorkKindPhase    WorkKind = "phase"
	WorkKindDecision WorkKind = "decision"
	WorkKindNote     WorkKind = "note"
)

// ContextType classifies planning context snippets attached to an item.
type ContextType string

// ContextType values.
const (
	ContextTypeNote       ContextType = "note"
	ContextTypeConstraint ContextType = "constraint"
	ContextTypeDecision   ContextType = "decision"
	ContextTypeReference  ContextType = "reference"
	ContextTypeWarning    ContextType = "warning"
	ContextTypeRunbook    ContextType = "runbook"
)

// ContextImportance represents relative importance for context blocks.
type ContextImportance string

// ContextImportance values.
const (
	ContextImportanceLow      ContextImportance = "low"
	ContextImportanceNormal   ContextImportance = "normal"
	ContextImportanceHigh     ContextImportance = "high"
	ContextImportanceCritical ContextImportance = "critical"
)

// ResourceType defines resource reference categories.
type ResourceType string

// ResourceType values.
const (
	ResourceTypeLocalFile ResourceType = "local_file"
	ResourceTypeLocalDir  ResourceType = "local_dir"
	ResourceTypeURL       ResourceType = "url"
	ResourceTypeDoc       ResourceType = "doc"
	ResourceTypeTicket    ResourceType = "ticket"
	ResourceTypeSnippet   ResourceType = "snippet"
)

// PathMode identifies whether a resource path is relative or absolute.
type PathMode string

// PathMode values.
const (
	PathModeRelative PathMode = "relative"
	PathModeAbsolute PathMode = "absolute"
)

// ChecklistItem describes a completion-contract checklist item.
type ChecklistItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// CompletionPolicy controls parent/child completion requirements.
type CompletionPolicy struct {
	RequireChildrenDone bool `json:"require_children_done"`
}

// CompletionContract stores start/complete checks and completion evidence.
type CompletionContract struct {
	StartCriteria      []ChecklistItem  `json:"start_criteria"`
	CompletionCriteria []ChecklistItem  `json:"completion_criteria"`
	CompletionChecklist []ChecklistItem `json:"completion_checklist"`
	CompletionEvidence []string         `json:"completion_evidence"`
	CompletionNotes    string           `json:"completion_notes"`
	Policy             CompletionPolicy `json:"policy"`
}

// ContextBlock stores typed contextual notes attached to a work item.
type ContextBlock struct {
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Type       ContextType       `json:"type"`
	Importance ContextImportance `json:"importance"`
}

// ResourceRef stores a path/URL reference that supports future context hydration.
type ResourceRef struct {
	ID             string     `json:"id"`
	ResourceType   ResourceType `json:"resource_type"`
	Location       string     `json:"location"`
	PathMode       PathMode   `json:"path_mode"`
	BaseAlias      string     `json:"base_alias"`
	Title          string     `json:"title"`
	Notes          string     `json:"notes"`
	Tags           []string   `json:"tags"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
}

// TaskMetadata stores rich planning context for an item.
type TaskMetadata struct {
	Objective               string             `json:"objective"`
	ImplementationNotesUser string             `json:"implementation_notes_user"`
	ImplementationNotesAgent string            `json:"implementation_notes_agent"`
	AcceptanceCriteria      string             `json:"acceptance_criteria"`
	DefinitionOfDone        string             `json:"definition_of_done"`
	ValidationPlan          string             `json:"validation_plan"`
	BlockedReason           string             `json:"blocked_reason"`
	RiskNotes               string             `json:"risk_notes"`
	CommandSnippets         []string           `json:"command_snippets"`
	ExpectedOutputs         []string           `json:"expected_outputs"`
	DecisionLog             []string           `json:"decision_log"`
	RelatedItems            []string           `json:"related_items"`
	TransitionNotes         string             `json:"transition_notes"`
	DependsOn               []string           `json:"depends_on"`
	BlockedBy               []string           `json:"blocked_by"`
	ContextBlocks           []ContextBlock     `json:"context_blocks"`
	ResourceRefs            []ResourceRef      `json:"resource_refs"`
	CompletionContract      CompletionContract `json:"completion_contract"`
}

// normalizeLifecycleState canonicalizes lifecycle state aliases.
func normalizeLifecycleState(state LifecycleState) LifecycleState {
	switch strings.TrimSpace(strings.ToLower(string(state))) {
	case "to-do", "todo":
		return StateTodo
	case "in-progress", "progress", "doing":
		return StateProgress
	case "done", "complete", "completed":
		return StateDone
	case "archived", "archive":
		return StateArchived
	default:
		return LifecycleState(strings.TrimSpace(strings.ToLower(string(state))))
	}
}

// isValidLifecycleState reports whether the lifecycle state is canonical.
func isValidLifecycleState(state LifecycleState) bool {
	state = normalizeLifecycleState(state)
	return slices.Contains([]LifecycleState{StateTodo, StateProgress, StateDone, StateArchived}, state)
}

// isValidActorType reports whether actor type is supported.
func isValidActorType(actorType ActorType) bool {
	actorType = ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	return slices.Contains([]ActorType{ActorTypeUser, ActorTypeAgent, ActorTypeSystem}, actorType)
}

// isValidWorkKind reports whether kind is non-empty after normalization.
func isValidWorkKind(kind WorkKind) bool {
	return strings.TrimSpace(string(kind)) != ""
}

// normalizeTaskMetadata trims and validates rich metadata.
func normalizeTaskMetadata(meta TaskMetadata) (TaskMetadata, error) {
	meta.Objective = strings.TrimSpace(meta.Objective)
	meta.ImplementationNotesUser = strings.TrimSpace(meta.ImplementationNotesUser)
	meta.ImplementationNotesAgent = strings.TrimSpace(meta.ImplementationNotesAgent)
	meta.AcceptanceCriteria = strings.TrimSpace(meta.AcceptanceCriteria)
	meta.DefinitionOfDone = strings.TrimSpace(meta.DefinitionOfDone)
	meta.ValidationPlan = strings.TrimSpace(meta.ValidationPlan)
	meta.BlockedReason = strings.TrimSpace(meta.BlockedReason)
	meta.RiskNotes = strings.TrimSpace(meta.RiskNotes)
	meta.TransitionNotes = strings.TrimSpace(meta.TransitionNotes)
	meta.CommandSnippets = normalizeStringList(meta.CommandSnippets)
	meta.ExpectedOutputs = normalizeStringList(meta.ExpectedOutputs)
	meta.DecisionLog = normalizeStringList(meta.DecisionLog)
	meta.RelatedItems = normalizeStringList(meta.RelatedItems)
	meta.DependsOn = normalizeStringList(meta.DependsOn)
	meta.BlockedBy = normalizeStringList(meta.BlockedBy)
	meta.CompletionContract.CompletionEvidence = normalizeStringList(meta.CompletionContract.CompletionEvidence)
	meta.CompletionContract.CompletionNotes = strings.TrimSpace(meta.CompletionContract.CompletionNotes)

	var err error
	meta.CompletionContract.StartCriteria, err = normalizeChecklist(meta.CompletionContract.StartCriteria)
	if err != nil {
		return TaskMetadata{}, err
	}
	meta.CompletionContract.CompletionCriteria, err = normalizeChecklist(meta.CompletionContract.CompletionCriteria)
	if err != nil {
		return TaskMetadata{}, err
	}
	meta.CompletionContract.CompletionChecklist, err = normalizeChecklist(meta.CompletionContract.CompletionChecklist)
	if err != nil {
		return TaskMetadata{}, err
	}

	contextBlocks := make([]ContextBlock, 0, len(meta.ContextBlocks))
	for i, block := range meta.ContextBlocks {
		block.Title = strings.TrimSpace(block.Title)
		block.Body = strings.TrimSpace(block.Body)
		if block.Body == "" {
			continue
		}
		block.Type = ContextType(strings.TrimSpace(strings.ToLower(string(block.Type))))
		if block.Type == "" {
			block.Type = ContextTypeNote
		}
		if !slices.Contains([]ContextType{
			ContextTypeNote,
			ContextTypeConstraint,
			ContextTypeDecision,
			ContextTypeReference,
			ContextTypeWarning,
			ContextTypeRunbook,
		}, block.Type) {
			return TaskMetadata{}, fmt.Errorf("invalid context block type at index %d", i)
		}
		block.Importance = ContextImportance(strings.TrimSpace(strings.ToLower(string(block.Importance))))
		if block.Importance == "" {
			block.Importance = ContextImportanceNormal
		}
		if !slices.Contains([]ContextImportance{
			ContextImportanceLow,
			ContextImportanceNormal,
			ContextImportanceHigh,
			ContextImportanceCritical,
		}, block.Importance) {
			return TaskMetadata{}, fmt.Errorf("invalid context block importance at index %d", i)
		}
		contextBlocks = append(contextBlocks, block)
	}
	meta.ContextBlocks = contextBlocks

	resourceRefs := make([]ResourceRef, 0, len(meta.ResourceRefs))
	for i, ref := range meta.ResourceRefs {
		ref.ID = strings.TrimSpace(ref.ID)
		ref.Location = strings.TrimSpace(ref.Location)
		ref.BaseAlias = strings.TrimSpace(ref.BaseAlias)
		ref.Title = strings.TrimSpace(ref.Title)
		ref.Notes = strings.TrimSpace(ref.Notes)
		ref.Tags = normalizeLabels(ref.Tags)
		if ref.Location == "" {
			continue
		}
		ref.ResourceType = ResourceType(strings.TrimSpace(strings.ToLower(string(ref.ResourceType))))
		if ref.ResourceType == "" {
			ref.ResourceType = ResourceTypeDoc
		}
		if !slices.Contains([]ResourceType{
			ResourceTypeLocalFile,
			ResourceTypeLocalDir,
			ResourceTypeURL,
			ResourceTypeDoc,
			ResourceTypeTicket,
			ResourceTypeSnippet,
		}, ref.ResourceType) {
			return TaskMetadata{}, fmt.Errorf("invalid resource type at index %d", i)
		}
		ref.PathMode = PathMode(strings.TrimSpace(strings.ToLower(string(ref.PathMode))))
		if ref.PathMode == "" {
			ref.PathMode = PathModeRelative
		}
		if !slices.Contains([]PathMode{PathModeRelative, PathModeAbsolute}, ref.PathMode) {
			return TaskMetadata{}, fmt.Errorf("invalid path mode at index %d", i)
		}
		if ref.LastVerifiedAt != nil {
			ts := ref.LastVerifiedAt.UTC().Truncate(time.Second)
			ref.LastVerifiedAt = &ts
		}
		resourceRefs = append(resourceRefs, ref)
	}
	meta.ResourceRefs = resourceRefs

	return meta, nil
}

// normalizeChecklist trims checklist ids/text and removes empty rows.
func normalizeChecklist(in []ChecklistItem) ([]ChecklistItem, error) {
	out := make([]ChecklistItem, 0, len(in))
	seen := map[string]struct{}{}
	for i, item := range in {
		item.ID = strings.TrimSpace(item.ID)
		item.Text = strings.TrimSpace(item.Text)
		if item.Text == "" {
			continue
		}
		if item.ID == "" {
			item.ID = fmt.Sprintf("item-%d", i+1)
		}
		if _, exists := seen[item.ID]; exists {
			return nil, fmt.Errorf("duplicate checklist id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

// normalizeStringList trims and deduplicates string slices.
func normalizeStringList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
