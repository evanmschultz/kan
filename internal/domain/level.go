package domain

import (
	"slices"
	"strings"
)

// ScopeLevel identifies one canonical hierarchy level.
type ScopeLevel string

// ScopeLevel values.
const (
	ScopeLevelProject  ScopeLevel = "project"
	ScopeLevelBranch   ScopeLevel = "branch"
	ScopeLevelPhase    ScopeLevel = "phase"
	ScopeLevelSubphase ScopeLevel = "subphase"
	ScopeLevelTask     ScopeLevel = "task"
	ScopeLevelSubtask  ScopeLevel = "subtask"
)

// validScopeLevels stores all supported level values.
var validScopeLevels = []ScopeLevel{
	ScopeLevelProject,
	ScopeLevelBranch,
	ScopeLevelPhase,
	ScopeLevelSubphase,
	ScopeLevelTask,
	ScopeLevelSubtask,
}

// LevelTuple stores one canonical scope tuple for level-scoped operations.
type LevelTuple struct {
	ProjectID string     `json:"project_id"`
	BranchID  string     `json:"branch_id,omitempty"`
	ScopeType ScopeLevel `json:"scope_type"`
	ScopeID   string     `json:"scope_id"`
}

// LevelTupleInput holds write-time values for LevelTuple normalization.
type LevelTupleInput struct {
	ProjectID string
	BranchID  string
	ScopeType ScopeLevel
	ScopeID   string
}

// NewLevelTuple validates and normalizes one level tuple.
func NewLevelTuple(in LevelTupleInput) (LevelTuple, error) {
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.BranchID = strings.TrimSpace(in.BranchID)
	in.ScopeType = NormalizeScopeLevel(in.ScopeType)
	in.ScopeID = strings.TrimSpace(in.ScopeID)

	if in.ProjectID == "" {
		return LevelTuple{}, ErrInvalidID
	}
	if in.ScopeType == "" {
		in.ScopeType = ScopeLevelProject
	}
	if !IsValidScopeLevel(in.ScopeType) {
		return LevelTuple{}, ErrInvalidScopeType
	}
	if in.ScopeType == ScopeLevelProject && in.ScopeID == "" {
		in.ScopeID = in.ProjectID
	}
	if in.ScopeType != ScopeLevelProject && in.ScopeID == "" {
		return LevelTuple{}, ErrInvalidScopeID
	}
	if in.ScopeType == ScopeLevelBranch && in.BranchID == "" {
		in.BranchID = in.ScopeID
	}

	return LevelTuple{
		ProjectID: in.ProjectID,
		BranchID:  in.BranchID,
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
	}, nil
}

// NormalizeScopeLevel canonicalizes one scope-level value.
func NormalizeScopeLevel(level ScopeLevel) ScopeLevel {
	return ScopeLevel(strings.TrimSpace(strings.ToLower(string(level))))
}

// IsValidScopeLevel reports whether a scope-level value is supported.
func IsValidScopeLevel(level ScopeLevel) bool {
	level = NormalizeScopeLevel(level)
	return slices.Contains(validScopeLevels, level)
}

// ScopeLevelFromKindAppliesTo converts a kind applies_to value into a scope level.
func ScopeLevelFromKindAppliesTo(scope KindAppliesTo) ScopeLevel {
	switch NormalizeKindAppliesTo(scope) {
	case KindAppliesToProject:
		return ScopeLevelProject
	case KindAppliesToBranch:
		return ScopeLevelBranch
	case KindAppliesToPhase:
		return ScopeLevelPhase
	case KindAppliesToSubphase:
		return ScopeLevelSubphase
	case KindAppliesToSubtask:
		return ScopeLevelSubtask
	case KindAppliesToTask:
		return ScopeLevelTask
	default:
		return ""
	}
}

// ScopeLevelFromCapabilityScopeType converts a capability scope into a scope level.
func ScopeLevelFromCapabilityScopeType(scope CapabilityScopeType) ScopeLevel {
	switch NormalizeCapabilityScopeType(scope) {
	case CapabilityScopeProject:
		return ScopeLevelProject
	case CapabilityScopeBranch:
		return ScopeLevelBranch
	case CapabilityScopePhase:
		return ScopeLevelPhase
	case CapabilityScopeSubphase:
		return ScopeLevelSubphase
	case CapabilityScopeSubtask:
		return ScopeLevelSubtask
	case CapabilityScopeTask:
		return ScopeLevelTask
	default:
		return ""
	}
}

// ToCapabilityScopeType maps one level value into a capability scope value.
func (level ScopeLevel) ToCapabilityScopeType() CapabilityScopeType {
	switch NormalizeScopeLevel(level) {
	case ScopeLevelProject:
		return CapabilityScopeProject
	case ScopeLevelBranch:
		return CapabilityScopeBranch
	case ScopeLevelPhase:
		return CapabilityScopePhase
	case ScopeLevelSubphase:
		return CapabilityScopeSubphase
	case ScopeLevelSubtask:
		return CapabilityScopeSubtask
	case ScopeLevelTask:
		return CapabilityScopeTask
	default:
		return ""
	}
}

// ToKindAppliesTo maps one level value into a kind applies_to value.
func (level ScopeLevel) ToKindAppliesTo() KindAppliesTo {
	switch NormalizeScopeLevel(level) {
	case ScopeLevelProject:
		return KindAppliesToProject
	case ScopeLevelBranch:
		return KindAppliesToBranch
	case ScopeLevelPhase:
		return KindAppliesToPhase
	case ScopeLevelSubphase:
		return KindAppliesToSubphase
	case ScopeLevelSubtask:
		return KindAppliesToSubtask
	case ScopeLevelTask:
		return KindAppliesToTask
	default:
		return ""
	}
}
