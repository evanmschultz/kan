package domain

import (
	"slices"
	"strings"
	"time"
)

// CapabilityRole identifies the role of a capability lease owner.
type CapabilityRole string

// Capability role values.
const (
	CapabilityRoleOrchestrator CapabilityRole = "orchestrator"
	CapabilityRoleWorker       CapabilityRole = "worker"
	CapabilityRoleSystem       CapabilityRole = "system"
)

// CapabilityScopeType identifies the scope a capability lease is bound to.
type CapabilityScopeType string

// Capability scope values.
const (
	CapabilityScopeProject  CapabilityScopeType = "project"
	CapabilityScopeBranch   CapabilityScopeType = "branch"
	CapabilityScopePhase    CapabilityScopeType = "phase"
	CapabilityScopeSubphase CapabilityScopeType = "subphase"
	CapabilityScopeTask     CapabilityScopeType = "task"
	CapabilityScopeSubtask  CapabilityScopeType = "subtask"
)

// validCapabilityRoles stores supported capability roles.
var validCapabilityRoles = []CapabilityRole{
	CapabilityRoleOrchestrator,
	CapabilityRoleWorker,
	CapabilityRoleSystem,
}

// validCapabilityScopes stores supported capability scope values.
var validCapabilityScopes = []CapabilityScopeType{
	CapabilityScopeProject,
	CapabilityScopeBranch,
	CapabilityScopePhase,
	CapabilityScopeSubphase,
	CapabilityScopeTask,
	CapabilityScopeSubtask,
}

// CapabilityLease stores one scoped, revocable capability token lease.
type CapabilityLease struct {
	InstanceID                string
	LeaseToken                string
	AgentName                 string
	ProjectID                 string
	ScopeType                 CapabilityScopeType
	ScopeID                   string
	Role                      CapabilityRole
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	IssuedAt                  time.Time
	ExpiresAt                 time.Time
	HeartbeatAt               time.Time
	RevokedAt                 *time.Time
	RevokedReason             string
}

// CapabilityLeaseInput holds values used to issue a new lease.
type CapabilityLeaseInput struct {
	InstanceID                string
	LeaseToken                string
	AgentName                 string
	ProjectID                 string
	ScopeType                 CapabilityScopeType
	ScopeID                   string
	Role                      CapabilityRole
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	ExpiresAt                 time.Time
}

// NewCapabilityLease normalizes and validates one lease issuance request.
func NewCapabilityLease(in CapabilityLeaseInput, now time.Time) (CapabilityLease, error) {
	in.InstanceID = strings.TrimSpace(in.InstanceID)
	in.LeaseToken = strings.TrimSpace(in.LeaseToken)
	in.AgentName = strings.TrimSpace(in.AgentName)
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ScopeType = NormalizeCapabilityScopeType(in.ScopeType)
	in.ScopeID = strings.TrimSpace(in.ScopeID)
	in.Role = NormalizeCapabilityRole(in.Role)
	in.ParentInstanceID = strings.TrimSpace(in.ParentInstanceID)

	if in.InstanceID == "" {
		return CapabilityLease{}, ErrInvalidID
	}
	if in.LeaseToken == "" {
		return CapabilityLease{}, ErrInvalidCapabilityToken
	}
	if in.AgentName == "" {
		return CapabilityLease{}, ErrInvalidName
	}
	if in.ProjectID == "" {
		return CapabilityLease{}, ErrInvalidID
	}
	if !IsValidCapabilityScopeType(in.ScopeType) {
		return CapabilityLease{}, ErrInvalidCapabilityScope
	}
	if in.ScopeType != CapabilityScopeProject && in.ScopeID == "" {
		return CapabilityLease{}, ErrInvalidCapabilityScope
	}
	if !IsValidCapabilityRole(in.Role) {
		return CapabilityLease{}, ErrInvalidCapabilityRole
	}
	if in.ExpiresAt.IsZero() || !in.ExpiresAt.After(now.UTC()) {
		return CapabilityLease{}, ErrInvalidCapabilityExpiry
	}

	ts := now.UTC()
	return CapabilityLease{
		InstanceID:                in.InstanceID,
		LeaseToken:                in.LeaseToken,
		AgentName:                 in.AgentName,
		ProjectID:                 in.ProjectID,
		ScopeType:                 in.ScopeType,
		ScopeID:                   in.ScopeID,
		Role:                      in.Role,
		ParentInstanceID:          in.ParentInstanceID,
		AllowEqualScopeDelegation: in.AllowEqualScopeDelegation,
		IssuedAt:                  ts,
		ExpiresAt:                 in.ExpiresAt.UTC(),
		HeartbeatAt:               ts,
	}, nil
}

// NormalizeCapabilityRole canonicalizes role values.
func NormalizeCapabilityRole(role CapabilityRole) CapabilityRole {
	return CapabilityRole(strings.TrimSpace(strings.ToLower(string(role))))
}

// NormalizeCapabilityScopeType canonicalizes scope values.
func NormalizeCapabilityScopeType(scope CapabilityScopeType) CapabilityScopeType {
	return CapabilityScopeType(strings.TrimSpace(strings.ToLower(string(scope))))
}

// IsValidCapabilityRole reports whether a role value is supported.
func IsValidCapabilityRole(role CapabilityRole) bool {
	role = NormalizeCapabilityRole(role)
	return slices.Contains(validCapabilityRoles, role)
}

// IsValidCapabilityScopeType reports whether a scope value is supported.
func IsValidCapabilityScopeType(scope CapabilityScopeType) bool {
	scope = NormalizeCapabilityScopeType(scope)
	return slices.Contains(validCapabilityScopes, scope)
}

// IsExpired reports whether the lease expired at the provided time.
func (l CapabilityLease) IsExpired(now time.Time) bool {
	return !now.UTC().Before(l.ExpiresAt.UTC())
}

// IsRevoked reports whether the lease was revoked.
func (l CapabilityLease) IsRevoked() bool {
	return l.RevokedAt != nil
}

// IsActive reports whether a lease is currently valid for mutation use.
func (l CapabilityLease) IsActive(now time.Time) bool {
	if l.IsRevoked() {
		return false
	}
	return !l.IsExpired(now)
}

// MatchesScope reports whether the lease can operate on a requested scope.
func (l CapabilityLease) MatchesScope(scopeType CapabilityScopeType, scopeID string) bool {
	scopeType = NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if l.ScopeType == CapabilityScopeProject {
		return true
	}
	if l.ScopeType != scopeType {
		return false
	}
	if strings.TrimSpace(l.ScopeID) == "" {
		return true
	}
	return l.ScopeID == scopeID
}

// MatchesIdentity reports whether the lease matches a request identity tuple.
func (l CapabilityLease) MatchesIdentity(agentName, leaseToken string) bool {
	return strings.TrimSpace(l.AgentName) == strings.TrimSpace(agentName) &&
		strings.TrimSpace(l.LeaseToken) == strings.TrimSpace(leaseToken)
}

// Heartbeat updates the lease heartbeat timestamp when active.
func (l *CapabilityLease) Heartbeat(now time.Time) {
	if l == nil {
		return
	}
	l.HeartbeatAt = now.UTC()
}

// Renew extends the lease expiry after validation.
func (l *CapabilityLease) Renew(expiresAt, now time.Time) error {
	if l == nil {
		return ErrInvalidID
	}
	if l.IsRevoked() {
		return ErrMutationLeaseRevoked
	}
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now.UTC()) {
		return ErrInvalidCapabilityExpiry
	}
	l.ExpiresAt = expiresAt
	l.HeartbeatAt = now.UTC()
	return nil
}

// Revoke marks a lease as revoked and captures the revocation reason.
func (l *CapabilityLease) Revoke(reason string, now time.Time) {
	if l == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "revoked"
	}
	ts := now.UTC()
	l.RevokedAt = &ts
	l.RevokedReason = reason
}
