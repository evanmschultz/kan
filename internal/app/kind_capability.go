package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/evanschultz/kan/internal/domain"
)

// CreateKindDefinitionInput holds write values for kind-catalog upsert behavior.
type CreateKindDefinitionInput struct {
	ID                  domain.KindID
	DisplayName         string
	DescriptionMarkdown string
	AppliesTo           []domain.KindAppliesTo
	AllowedParentScopes []domain.KindAppliesTo
	PayloadSchemaJSON   string
	Template            domain.KindTemplate
}

// SetProjectAllowedKindsInput holds project allowlist update values.
type SetProjectAllowedKindsInput struct {
	ProjectID string
	KindIDs   []domain.KindID
}

// IssueCapabilityLeaseInput holds capability-lease issuance values.
type IssueCapabilityLeaseInput struct {
	ProjectID                 string
	ScopeType                 domain.CapabilityScopeType
	ScopeID                   string
	Role                      domain.CapabilityRole
	AgentName                 string
	AgentInstanceID           string
	ParentInstanceID          string
	AllowEqualScopeDelegation bool
	RequestedTTL              time.Duration
	OverrideToken             string
}

// HeartbeatCapabilityLeaseInput holds heartbeat update values.
type HeartbeatCapabilityLeaseInput struct {
	AgentInstanceID string
	LeaseToken      string
}

// RenewCapabilityLeaseInput holds lease-renewal values.
type RenewCapabilityLeaseInput struct {
	AgentInstanceID string
	LeaseToken      string
	TTL             time.Duration
}

// RevokeCapabilityLeaseInput holds lease revoke values.
type RevokeCapabilityLeaseInput struct {
	AgentInstanceID string
	Reason          string
}

// RevokeAllCapabilityLeasesInput holds one-shot scope revoke-all values.
type RevokeAllCapabilityLeasesInput struct {
	ProjectID string
	ScopeType domain.CapabilityScopeType
	ScopeID   string
	Reason    string
}

// schemaCacheEntry stores one compiled schema cache item.
type schemaCacheEntry struct {
	hash      string
	validator *jsonSchemaValidator
}

// kindBootstrapState tracks one-time bootstrap initialization.
type kindBootstrapState struct {
	once sync.Once
	err  error
}

// defaultCapabilityLeaseTTL defines default lease expiration behavior.
const defaultCapabilityLeaseTTL = 24 * time.Hour

// ListKindDefinitions lists catalog entries with deterministic ordering.
func (s *Service) ListKindDefinitions(ctx context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {
		return nil, err
	}
	kinds, err := s.repo.ListKindDefinitions(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(kinds, func(i, j int) bool {
		if kinds[i].DisplayName == kinds[j].DisplayName {
			return kinds[i].ID < kinds[j].ID
		}
		return kinds[i].DisplayName < kinds[j].DisplayName
	})
	return kinds, nil
}

// UpsertKindDefinition creates or updates one catalog kind definition.
func (s *Service) UpsertKindDefinition(ctx context.Context, in CreateKindDefinitionInput) (domain.KindDefinition, error) {
	now := s.clock()
	kind, err := domain.NewKindDefinition(domain.KindDefinitionInput{
		ID:                  in.ID,
		DisplayName:         in.DisplayName,
		DescriptionMarkdown: in.DescriptionMarkdown,
		AppliesTo:           in.AppliesTo,
		AllowedParentScopes: in.AllowedParentScopes,
		PayloadSchemaJSON:   in.PayloadSchemaJSON,
		Template:            in.Template,
	}, now)
	if err != nil {
		return domain.KindDefinition{}, err
	}

	existing, err := s.repo.GetKindDefinition(ctx, kind.ID)
	if err == nil {
		kind.CreatedAt = existing.CreatedAt
		kind.UpdatedAt = now.UTC()
		kind.ArchivedAt = existing.ArchivedAt
		if updateErr := s.repo.UpdateKindDefinition(ctx, kind); updateErr != nil {
			return domain.KindDefinition{}, updateErr
		}
		s.clearCompiledSchema(kind.ID)
		return kind, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return domain.KindDefinition{}, err
	}
	if createErr := s.repo.CreateKindDefinition(ctx, kind); createErr != nil {
		return domain.KindDefinition{}, createErr
	}
	s.clearCompiledSchema(kind.ID)
	return kind, nil
}

// SetProjectAllowedKinds updates one project's explicit allowlist.
func (s *Service) SetProjectAllowedKinds(ctx context.Context, in SetProjectAllowedKindsInput) error {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		return err
	}
	if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {
		return err
	}
	kindIDs := normalizeKindIDList(in.KindIDs)
	if len(kindIDs) == 0 {
		return domain.ErrKindNotAllowed
	}
	for _, kindID := range kindIDs {
		if _, err := s.repo.GetKindDefinition(ctx, kindID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return fmt.Errorf("%w: %q", domain.ErrKindNotFound, kindID)
			}
			return err
		}
	}
	return s.repo.SetProjectAllowedKinds(ctx, projectID, kindIDs)
}

// ListProjectAllowedKinds lists one project's explicit allowlist.
func (s *Service) ListProjectAllowedKinds(ctx context.Context, projectID string) ([]domain.KindID, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	kindIDs, err := s.repo.ListProjectAllowedKinds(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return normalizeKindIDList(kindIDs), nil
}

// IssueCapabilityLease issues one scoped lease after overlap/policy validation.
func (s *Service) IssueCapabilityLease(ctx context.Context, in IssueCapabilityLeaseInput) (domain.CapabilityLease, error) {
	now := s.clock().UTC()
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return domain.CapabilityLease{}, domain.ErrInvalidID
	}
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return domain.CapabilityLease{}, err
	}

	ttl := in.RequestedTTL
	if ttl <= 0 {
		ttl = s.defaultLeaseTTL
	}
	if ttl <= 0 {
		ttl = defaultCapabilityLeaseTTL
	}

	instanceID := strings.TrimSpace(in.AgentInstanceID)
	if instanceID == "" {
		instanceID = s.idGen()
	}
	leaseToken := strings.TrimSpace(s.idGen())
	if leaseToken == "" {
		leaseToken = fmt.Sprintf("lease-%d", now.UnixNano())
	}

	lease, err := domain.NewCapabilityLease(domain.CapabilityLeaseInput{
		InstanceID:                instanceID,
		LeaseToken:                leaseToken,
		AgentName:                 strings.TrimSpace(in.AgentName),
		ProjectID:                 projectID,
		ScopeType:                 in.ScopeType,
		ScopeID:                   strings.TrimSpace(in.ScopeID),
		Role:                      in.Role,
		ParentInstanceID:          strings.TrimSpace(in.ParentInstanceID),
		AllowEqualScopeDelegation: in.AllowEqualScopeDelegation,
		ExpiresAt:                 now.Add(ttl),
	}, now)
	if err != nil {
		return domain.CapabilityLease{}, err
	}

	if strings.TrimSpace(in.ParentInstanceID) != "" {
		parent, parentErr := s.repo.GetCapabilityLease(ctx, strings.TrimSpace(in.ParentInstanceID))
		if parentErr != nil {
			return domain.CapabilityLease{}, parentErr
		}
		if !parent.IsActive(now) {
			return domain.CapabilityLease{}, domain.ErrMutationLeaseExpired
		}
		if parent.ProjectID != lease.ProjectID {
			return domain.CapabilityLease{}, domain.ErrInvalidCapabilityScope
		}
		if !in.AllowEqualScopeDelegation && parent.ScopeType == lease.ScopeType && parent.ScopeID == lease.ScopeID {
			return domain.CapabilityLease{}, domain.ErrInvalidCapabilityScope
		}
	}

	if lease.Role == domain.CapabilityRoleOrchestrator {
		if err := s.ensureOrchestratorOverlapPolicy(ctx, project, lease, strings.TrimSpace(in.OverrideToken)); err != nil {
			return domain.CapabilityLease{}, err
		}
	}

	if err := s.repo.CreateCapabilityLease(ctx, lease); err != nil {
		return domain.CapabilityLease{}, err
	}
	return lease, nil
}

// HeartbeatCapabilityLease refreshes heartbeat on one active lease.
func (s *Service) HeartbeatCapabilityLease(ctx context.Context, in HeartbeatCapabilityLeaseInput) (domain.CapabilityLease, error) {
	now := s.clock().UTC()
	instanceID := strings.TrimSpace(in.AgentInstanceID)
	lease, err := s.repo.GetCapabilityLease(ctx, instanceID)
	if err != nil {
		return domain.CapabilityLease{}, err
	}
	if strings.TrimSpace(in.LeaseToken) != strings.TrimSpace(lease.LeaseToken) {
		return domain.CapabilityLease{}, domain.ErrMutationLeaseInvalid
	}
	if lease.IsRevoked() {
		return domain.CapabilityLease{}, domain.ErrMutationLeaseRevoked
	}
	if lease.IsExpired(now) {
		return domain.CapabilityLease{}, domain.ErrMutationLeaseExpired
	}
	lease.Heartbeat(now)
	if err := s.repo.UpdateCapabilityLease(ctx, lease); err != nil {
		return domain.CapabilityLease{}, err
	}
	return lease, nil
}

// RenewCapabilityLease extends expiry for one existing lease.
func (s *Service) RenewCapabilityLease(ctx context.Context, in RenewCapabilityLeaseInput) (domain.CapabilityLease, error) {
	now := s.clock().UTC()
	lease, err := s.repo.GetCapabilityLease(ctx, strings.TrimSpace(in.AgentInstanceID))
	if err != nil {
		return domain.CapabilityLease{}, err
	}
	if strings.TrimSpace(in.LeaseToken) != strings.TrimSpace(lease.LeaseToken) {
		return domain.CapabilityLease{}, domain.ErrMutationLeaseInvalid
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = s.defaultLeaseTTL
	}
	if ttl <= 0 {
		ttl = defaultCapabilityLeaseTTL
	}
	if err := lease.Renew(now.Add(ttl), now); err != nil {
		return domain.CapabilityLease{}, err
	}
	if err := s.repo.UpdateCapabilityLease(ctx, lease); err != nil {
		return domain.CapabilityLease{}, err
	}
	return lease, nil
}

// RevokeCapabilityLease revokes one capability lease by instance id.
func (s *Service) RevokeCapabilityLease(ctx context.Context, in RevokeCapabilityLeaseInput) (domain.CapabilityLease, error) {
	now := s.clock().UTC()
	lease, err := s.repo.GetCapabilityLease(ctx, strings.TrimSpace(in.AgentInstanceID))
	if err != nil {
		return domain.CapabilityLease{}, err
	}
	lease.Revoke(strings.TrimSpace(in.Reason), now)
	if err := s.repo.UpdateCapabilityLease(ctx, lease); err != nil {
		return domain.CapabilityLease{}, err
	}
	return lease, nil
}

// RevokeAllCapabilityLeases revokes all scope-matching leases in one operation.
func (s *Service) RevokeAllCapabilityLeases(ctx context.Context, in RevokeAllCapabilityLeasesInput) error {
	projectID := strings.TrimSpace(in.ProjectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}
	scopeType := domain.NormalizeCapabilityScopeType(in.ScopeType)
	if !domain.IsValidCapabilityScopeType(scopeType) {
		return domain.ErrInvalidCapabilityScope
	}
	scopeID := strings.TrimSpace(in.ScopeID)
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "project scope revoke-all"
	}
	return s.repo.RevokeCapabilityLeasesByScope(ctx, projectID, scopeType, scopeID, s.clock().UTC(), reason)
}

// ensureOrchestratorOverlapPolicy enforces project policy for overlapping orchestrator leases.
func (s *Service) ensureOrchestratorOverlapPolicy(ctx context.Context, project domain.Project, next domain.CapabilityLease, overrideToken string) error {
	leases, err := s.repo.ListCapabilityLeasesByScope(ctx, next.ProjectID, next.ScopeType, next.ScopeID)
	if err != nil {
		return err
	}
	now := s.clock().UTC()
	for _, existing := range leases {
		if existing.InstanceID == next.InstanceID {
			continue
		}
		if existing.Role != domain.CapabilityRoleOrchestrator {
			continue
		}
		if !existing.IsActive(now) {
			continue
		}

		policy := project.Metadata.CapabilityPolicy
		if !policy.AllowOrchestratorOverride {
			return domain.ErrOrchestratorOverlap
		}
		expected := strings.TrimSpace(policy.OrchestratorOverrideToken)
		if expected == "" {
			return domain.ErrOverrideTokenRequired
		}
		if strings.TrimSpace(overrideToken) == "" {
			return domain.ErrOverrideTokenRequired
		}
		if strings.TrimSpace(overrideToken) != expected {
			return domain.ErrOverrideTokenInvalid
		}
	}
	return nil
}

// enforceMutationGuard validates capability lease requirements for non-user actors.
func (s *Service) enforceMutationGuard(ctx context.Context, projectID string, actorType domain.ActorType, scopeType domain.CapabilityScopeType, scopeID string) error {
	if !s.requireAgentLease {
		return nil
	}
	actorType = domain.ActorType(strings.TrimSpace(strings.ToLower(string(actorType))))
	if actorType == "" {
		actorType = domain.ActorTypeUser
	}
	guard, ok := MutationGuardFromContext(ctx)
	if actorType == domain.ActorTypeUser && !ok {
		return nil
	}

	if !ok {
		log.Error("mutation blocked: missing agent lease", "project_id", projectID, "actor_type", actorType)
		return domain.ErrMutationLeaseRequired
	}
	lease, err := s.repo.GetCapabilityLease(ctx, guard.AgentInstanceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Error("mutation blocked: lease not found", "project_id", projectID, "agent_instance_id", guard.AgentInstanceID)
			return domain.ErrMutationLeaseInvalid
		}
		return err
	}
	if !lease.MatchesIdentity(guard.AgentName, guard.LeaseToken) {
		log.Error("mutation blocked: lease identity mismatch", "project_id", projectID, "agent_instance_id", guard.AgentInstanceID)
		return domain.ErrMutationLeaseInvalid
	}
	if lease.ProjectID != strings.TrimSpace(projectID) {
		log.Error("mutation blocked: lease project mismatch", "project_id", projectID, "lease_project_id", lease.ProjectID, "agent_instance_id", guard.AgentInstanceID)
		return domain.ErrMutationLeaseInvalid
	}
	now := s.clock().UTC()
	if lease.IsRevoked() {
		log.Error("mutation blocked: lease revoked", "project_id", projectID, "agent_instance_id", guard.AgentInstanceID)
		return domain.ErrMutationLeaseRevoked
	}
	if lease.IsExpired(now) {
		log.Error("mutation blocked: lease expired", "project_id", projectID, "agent_instance_id", guard.AgentInstanceID)
		return domain.ErrMutationLeaseExpired
	}
	if !lease.MatchesScope(scopeType, strings.TrimSpace(scopeID)) {
		log.Error("mutation blocked: lease scope mismatch", "project_id", projectID, "agent_instance_id", guard.AgentInstanceID, "lease_scope_type", lease.ScopeType, "lease_scope_id", lease.ScopeID, "requested_scope_type", scopeType, "requested_scope_id", scopeID)
		return domain.ErrMutationLeaseInvalid
	}
	lease.Heartbeat(now)
	if err := s.repo.UpdateCapabilityLease(ctx, lease); err != nil {
		return err
	}
	return nil
}

// ensureKindCatalogBootstrapped seeds built-in kind definitions when catalog is empty.
func (s *Service) ensureKindCatalogBootstrapped(ctx context.Context) error {
	s.kindBootstrap.once.Do(func() {
		kinds, err := s.repo.ListKindDefinitions(ctx, true)
		if err != nil {
			s.kindBootstrap.err = err
			return
		}
		if len(kinds) > 0 {
			return
		}
		now := s.clock()
		for _, in := range defaultKindDefinitionInputs() {
			kind, buildErr := domain.NewKindDefinition(in, now)
			if buildErr != nil {
				s.kindBootstrap.err = buildErr
				return
			}
			if createErr := s.repo.CreateKindDefinition(ctx, kind); createErr != nil {
				s.kindBootstrap.err = createErr
				return
			}
		}
	})
	return s.kindBootstrap.err
}

// validateProjectKind validates project kind and metadata payload constraints.
func (s *Service) validateProjectKind(ctx context.Context, projectID string, kindID domain.KindID, payload json.RawMessage) error {
	if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {
		return err
	}
	kindID = domain.NormalizeKindID(kindID)
	if kindID == "" {
		kindID = domain.DefaultProjectKind
	}
	kind, err := s.repo.GetKindDefinition(ctx, kindID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("%w: %q", domain.ErrKindNotFound, kindID)
		}
		return err
	}
	if !kind.AppliesToScope(domain.KindAppliesToProject) {
		return fmt.Errorf("%w: %q does not apply to project", domain.ErrKindNotAllowed, kind.ID)
	}
	if strings.TrimSpace(projectID) != "" {
		allowed, allowErr := s.resolveProjectAllowedKinds(ctx, projectID)
		if allowErr != nil {
			return allowErr
		}
		if _, ok := allowed[kind.ID]; !ok {
			return fmt.Errorf("%w: %q", domain.ErrKindNotAllowed, kind.ID)
		}
	}
	if err := s.validateKindPayload(kind, payload); err != nil {
		return err
	}
	return nil
}

// validateTaskKind validates project allowlist, applies_to rules, parent constraints, and schema payload.
func (s *Service) validateTaskKind(ctx context.Context, projectID string, kindID domain.KindID, scope domain.KindAppliesTo, parent *domain.Task, payload json.RawMessage) (domain.KindDefinition, error) {
	if err := s.ensureKindCatalogBootstrapped(ctx); err != nil {
		return domain.KindDefinition{}, err
	}
	kindID = domain.NormalizeKindID(kindID)
	if kindID == "" {
		kindID = domain.KindID(domain.WorkKindTask)
	}
	scope = domain.NormalizeKindAppliesTo(scope)
	if scope == "" {
		if parent != nil {
			scope = domain.KindAppliesToSubtask
		} else {
			scope = domain.KindAppliesToTask
		}
	}
	if !domain.IsValidWorkItemAppliesTo(scope) {
		return domain.KindDefinition{}, domain.ErrInvalidKindAppliesTo
	}

	kind, err := s.repo.GetKindDefinition(ctx, kindID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.KindDefinition{}, fmt.Errorf("%w: %q", domain.ErrKindNotFound, kindID)
		}
		return domain.KindDefinition{}, err
	}
	if !kind.AppliesToScope(scope) {
		return domain.KindDefinition{}, fmt.Errorf("%w: %q does not apply to %q", domain.ErrKindNotAllowed, kindID, scope)
	}
	if parent != nil {
		if !kind.AllowsParentScope(parent.Scope) {
			return domain.KindDefinition{}, fmt.Errorf("%w: %q parent scope %q", domain.ErrKindNotAllowed, kindID, parent.Scope)
		}
	}
	allowed, err := s.resolveProjectAllowedKinds(ctx, projectID)
	if err != nil {
		return domain.KindDefinition{}, err
	}
	if _, ok := allowed[kind.ID]; !ok {
		return domain.KindDefinition{}, fmt.Errorf("%w: %q", domain.ErrKindNotAllowed, kind.ID)
	}
	if err := s.validateKindPayload(kind, payload); err != nil {
		return domain.KindDefinition{}, err
	}
	return kind, nil
}

// resolveProjectAllowedKinds returns explicit project allowlist values or built-in fallback.
func (s *Service) resolveProjectAllowedKinds(ctx context.Context, projectID string) (map[domain.KindID]struct{}, error) {
	kindIDs, err := s.repo.ListProjectAllowedKinds(ctx, projectID)
	if err != nil {
		return nil, err
	}
	kindIDs = normalizeKindIDList(kindIDs)
	if len(kindIDs) == 0 {
		kinds, listErr := s.repo.ListKindDefinitions(ctx, false)
		if listErr != nil {
			return nil, listErr
		}
		for _, kind := range kinds {
			if len(kind.AppliesTo) == 0 {
				continue
			}
			kindIDs = append(kindIDs, kind.ID)
		}
		kindIDs = normalizeKindIDList(kindIDs)
	}
	allowed := make(map[domain.KindID]struct{}, len(kindIDs))
	for _, kindID := range kindIDs {
		allowed[kindID] = struct{}{}
	}
	return allowed, nil
}

// initializeProjectAllowedKinds assigns default allowlist entries for a new project.
func (s *Service) initializeProjectAllowedKinds(ctx context.Context, project domain.Project) error {
	kinds, err := s.repo.ListKindDefinitions(ctx, false)
	if err != nil {
		return err
	}
	kindIDs := make([]domain.KindID, 0, len(kinds))
	for _, kind := range kinds {
		kindIDs = append(kindIDs, kind.ID)
	}
	if len(kindIDs) == 0 {
		kindIDs = []domain.KindID{domain.DefaultProjectKind, domain.KindID(domain.WorkKindTask), domain.KindID(domain.WorkKindSubtask), domain.KindID(domain.WorkKindPhase), domain.KindID(domain.WorkKindDecision), domain.KindID(domain.WorkKindNote)}
	}
	if !slices.Contains(kindIDs, project.Kind) {
		kindIDs = append(kindIDs, project.Kind)
	}
	return s.repo.SetProjectAllowedKinds(ctx, project.ID, normalizeKindIDList(kindIDs))
}

// validateKindPayload validates one payload against a kind definition schema.
func (s *Service) validateKindPayload(kind domain.KindDefinition, payload json.RawMessage) error {
	validator, err := s.compiledSchemaForKind(kind)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidKindPayloadSchema, err)
	}
	if validator == nil {
		return nil
	}
	if err := validator.ValidatePayload(payload); err != nil {
		return fmt.Errorf("%w: kind %q %v", domain.ErrInvalidKindPayload, kind.ID, err)
	}
	return nil
}

// compiledSchemaForKind returns a cached schema validator for one kind definition.
func (s *Service) compiledSchemaForKind(kind domain.KindDefinition) (*jsonSchemaValidator, error) {
	schema := strings.TrimSpace(kind.PayloadSchemaJSON)
	if schema == "" {
		return nil, nil
	}
	hash := hashSchema(schema)
	cacheKey := string(kind.ID)

	s.schemaCacheMu.RLock()
	if entry, ok := s.schemaCache[cacheKey]; ok && entry.hash == hash {
		s.schemaCacheMu.RUnlock()
		return entry.validator, nil
	}
	s.schemaCacheMu.RUnlock()

	compiled, err := compileJSONSchema(schema)
	if err != nil {
		return nil, err
	}
	s.schemaCacheMu.Lock()
	s.schemaCache[cacheKey] = schemaCacheEntry{hash: hash, validator: compiled}
	s.schemaCacheMu.Unlock()
	return compiled, nil
}

// clearCompiledSchema removes one kind schema from the validator cache.
func (s *Service) clearCompiledSchema(kindID domain.KindID) {
	s.schemaCacheMu.Lock()
	defer s.schemaCacheMu.Unlock()
	delete(s.schemaCache, string(domain.NormalizeKindID(kindID)))
}

// hashSchema returns a deterministic digest for schema cache keys.
func hashSchema(schema string) string {
	sum := sha256.Sum256([]byte(schema))
	return hex.EncodeToString(sum[:])
}

// normalizeKindIDList trims, deduplicates, and sorts kind identifiers.
func normalizeKindIDList(in []domain.KindID) []domain.KindID {
	out := make([]domain.KindID, 0, len(in))
	seen := map[domain.KindID]struct{}{}
	for _, raw := range in {
		id := domain.NormalizeKindID(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

// defaultKindDefinitionInputs returns built-in kind definitions for first boot.
func defaultKindDefinitionInputs() []domain.KindDefinitionInput {
	return []domain.KindDefinitionInput{
		{ID: domain.DefaultProjectKind, DisplayName: "Project", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{ID: domain.KindID(domain.WorkKindTask), DisplayName: "Task", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{ID: domain.KindID(domain.WorkKindSubtask), DisplayName: "Subtask", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}, AllowedParentScopes: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask, domain.KindAppliesToPhase, domain.KindAppliesToBranch}},
		{ID: domain.KindID(domain.WorkKindPhase), DisplayName: "Phase", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase, domain.KindAppliesToTask}, AllowedParentScopes: []domain.KindAppliesTo{domain.KindAppliesToBranch, domain.KindAppliesToPhase, domain.KindAppliesToTask}},
		{ID: domain.KindID("branch"), DisplayName: "Branch", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToBranch}, AllowedParentScopes: []domain.KindAppliesTo{domain.KindAppliesToBranch}},
		{ID: domain.KindID(domain.WorkKindDecision), DisplayName: "Decision", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask}},
		{ID: domain.KindID(domain.WorkKindNote), DisplayName: "Note", AppliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask}},
	}
}

// applyKindTemplateSystemActions applies template checklist/child auto-actions after task creation.
func (s *Service) applyKindTemplateSystemActions(ctx context.Context, parent domain.Task, kind domain.KindDefinition) error {
	if len(kind.Template.CompletionChecklist) == 0 && len(kind.Template.AutoCreateChildren) == 0 {
		return nil
	}

	if len(kind.Template.CompletionChecklist) > 0 {
		merged := mergeChecklistItems(parent.Metadata.CompletionContract.CompletionChecklist, kind.Template.CompletionChecklist)
		if len(merged) != len(parent.Metadata.CompletionContract.CompletionChecklist) {
			updated := parent
			updated.Metadata.CompletionContract.CompletionChecklist = merged
			updated.UpdatedAt = s.clock().UTC()
			updated.UpdatedByType = domain.ActorTypeSystem
			updated.UpdatedByActor = "kan-system-template"
			if err := s.repo.UpdateTask(ctx, updated); err != nil {
				return err
			}
		}
	}

	if len(kind.Template.AutoCreateChildren) == 0 {
		return nil
	}
	columns, err := s.repo.ListColumns(ctx, parent.ProjectID, true)
	if err != nil {
		return err
	}
	lifecycleState := lifecycleStateForColumnID(columns, parent.ColumnID)
	if lifecycleState == "" {
		lifecycleState = domain.StateTodo
	}

	for _, childSpec := range kind.Template.AutoCreateChildren {
		childScope := childSpec.AppliesTo
		if childScope == "" {
			childScope = domain.KindAppliesToSubtask
		}
		childMetadata, buildErr := normalizeTaskMetadataFromKindPayload(childSpec.MetadataPayload)
		if buildErr != nil {
			return buildErr
		}
		if _, validateErr := s.validateTaskKind(ctx, parent.ProjectID, childSpec.Kind, childScope, &parent, childMetadata.KindPayload); validateErr != nil {
			return validateErr
		}
		position, posErr := s.nextTaskPosition(ctx, parent.ProjectID, parent.ColumnID)
		if posErr != nil {
			return posErr
		}
		child, childErr := domain.NewTask(domain.TaskInput{
			ID:             s.idGen(),
			ProjectID:      parent.ProjectID,
			ParentID:       parent.ID,
			Kind:           domain.WorkKind(childSpec.Kind),
			Scope:          childScope,
			LifecycleState: lifecycleState,
			ColumnID:       parent.ColumnID,
			Position:       position,
			Title:          childSpec.Title,
			Description:    childSpec.Description,
			Priority:       domain.PriorityMedium,
			Labels:         childSpec.Labels,
			Metadata:       childMetadata,
			CreatedByActor: "kan-system-template",
			UpdatedByActor: "kan-system-template",
			UpdatedByType:  domain.ActorTypeSystem,
		}, s.clock())
		if childErr != nil {
			return childErr
		}
		if err := s.repo.CreateTask(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

// nextTaskPosition calculates the next append position for a project column.
func (s *Service) nextTaskPosition(ctx context.Context, projectID, columnID string) (int, error) {
	tasks, err := s.repo.ListTasks(ctx, projectID, true)
	if err != nil {
		return 0, err
	}
	position := 0
	for _, task := range tasks {
		if task.ColumnID == columnID && task.Position >= position {
			position = task.Position + 1
		}
	}
	return position, nil
}

// normalizeTaskMetadataFromKindPayload constructs metadata for template children.
func normalizeTaskMetadataFromKindPayload(payload json.RawMessage) (domain.TaskMetadata, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) > 0 && !json.Valid(payload) {
		return domain.TaskMetadata{}, domain.ErrInvalidKindPayload
	}
	return domain.TaskMetadata{KindPayload: payload}, nil
}

// mergeChecklistItems appends checklist rows not already present by ID.
func mergeChecklistItems(existing, incoming []domain.ChecklistItem) []domain.ChecklistItem {
	out := append([]domain.ChecklistItem(nil), existing...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[strings.TrimSpace(item.ID)] = struct{}{}
	}
	for _, item := range incoming {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, item)
	}
	return out
}
