package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/domain"
	_ "modernc.org/sqlite"
)

// driverName defines a package constant value.
const driverName = "sqlite"

// Repository represents repository data used by this package.
type Repository struct {
	db *sql.DB
}

// Open opens the requested operation.
func Open(path string) (*Repository, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	repo := &Repository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

// OpenInMemory opens in memory.
func OpenInMemory() (*Repository, error) {
	db, err := sql.Open(driverName, "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open sqlite memory: %w", err)
	}
	repo := &Repository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

// Close closes the requested operation.
func (r *Repository) Close() error {
	return r.db.Close()
}

// migrate handles migrate.
func (r *Repository) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'project',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS columns_v1 (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			wip_limit INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'task',
			scope TEXT NOT NULL DEFAULT 'task',
			lifecycle_state TEXT NOT NULL DEFAULT 'todo',
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_by_actor TEXT NOT NULL DEFAULT 'kan-user',
			updated_by_actor TEXT NOT NULL DEFAULT 'kan-user',
			updated_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			archived_at TEXT,
			canceled_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(column_id) REFERENCES columns_v1(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS work_items (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT 'task',
			scope TEXT NOT NULL DEFAULT 'task',
			lifecycle_state TEXT NOT NULL DEFAULT 'todo',
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_by_actor TEXT NOT NULL DEFAULT 'kan-user',
			updated_by_actor TEXT NOT NULL DEFAULT 'kan-user',
			updated_by_type TEXT NOT NULL DEFAULT 'user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT,
			archived_at TEXT,
			canceled_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(column_id) REFERENCES columns_v1(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS change_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL,
			work_item_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		// comments.target_id is polymorphic, so only project_id is enforced as a foreign key.
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			body_markdown TEXT NOT NULL,
			actor_type TEXT NOT NULL DEFAULT 'user',
			author_name TEXT NOT NULL DEFAULT 'kan-user',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS kind_catalog (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			description_markdown TEXT NOT NULL DEFAULT '',
			applies_to_json TEXT NOT NULL DEFAULT '[]',
			allowed_parent_scopes_json TEXT NOT NULL DEFAULT '[]',
			payload_schema_json TEXT NOT NULL DEFAULT '',
			template_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS project_allowed_kinds (
			project_id TEXT NOT NULL,
			kind_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY(project_id, kind_id),
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(kind_id) REFERENCES kind_catalog(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS capability_leases (
			instance_id TEXT PRIMARY KEY,
			lease_token TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			project_id TEXT NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL,
			parent_instance_id TEXT NOT NULL DEFAULT '',
			allow_equal_scope_delegation INTEGER NOT NULL DEFAULT 0,
			issued_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			heartbeat_at TEXT NOT NULL,
			revoked_at TEXT,
			revoke_reason TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_columns_project_position ON columns_v1(project_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_column_position ON tasks(project_id, column_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_project_column_position ON work_items(project_id, column_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_project_parent ON work_items(project_id, parent_id);`,
		`CREATE INDEX IF NOT EXISTS idx_change_events_project_created_at ON change_events(project_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project_target_created_at ON comments(project_id, target_type, target_id, created_at ASC, id ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_project_created_at ON comments(project_id, created_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_project_allowed_kinds_project ON project_allowed_kinds(project_id, kind_id);`,
		`CREATE INDEX IF NOT EXISTS idx_capability_leases_scope ON capability_leases(project_id, scope_type, scope_id, role);`,
		`CREATE INDEX IF NOT EXISTS idx_capability_leases_expiry ON capability_leases(expires_at, revoked_at);`,
	}

	for _, stmt := range stmts {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE projects ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add projects.metadata_json: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE projects ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add projects.kind: %w", err)
	}
	taskAlterStatements := []string{
		`ALTER TABLE tasks ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE tasks ADD COLUMN scope TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE tasks ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'todo'`,
		`ALTER TABLE tasks ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE tasks ADD COLUMN created_by_actor TEXT NOT NULL DEFAULT 'kan-user'`,
		`ALTER TABLE tasks ADD COLUMN updated_by_actor TEXT NOT NULL DEFAULT 'kan-user'`,
		`ALTER TABLE tasks ADD COLUMN updated_by_type TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE tasks ADD COLUMN started_at TEXT`,
		`ALTER TABLE tasks ADD COLUMN completed_at TEXT`,
		`ALTER TABLE tasks ADD COLUMN canceled_at TEXT`,
	}
	for _, stmt := range taskAlterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite tasks: %w", err)
		}
	}
	workItemAlterStatements := []string{
		`ALTER TABLE work_items ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE work_items ADD COLUMN kind TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE work_items ADD COLUMN scope TEXT NOT NULL DEFAULT 'task'`,
		`ALTER TABLE work_items ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'todo'`,
		`ALTER TABLE work_items ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE work_items ADD COLUMN created_by_actor TEXT NOT NULL DEFAULT 'kan-user'`,
		`ALTER TABLE work_items ADD COLUMN updated_by_actor TEXT NOT NULL DEFAULT 'kan-user'`,
		`ALTER TABLE work_items ADD COLUMN updated_by_type TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE work_items ADD COLUMN started_at TEXT`,
		`ALTER TABLE work_items ADD COLUMN completed_at TEXT`,
		`ALTER TABLE work_items ADD COLUMN canceled_at TEXT`,
	}
	for _, stmt := range workItemAlterStatements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumnErr(err) {
			return fmt.Errorf("migrate sqlite work_items: %w", err)
		}
	}
	if _, err := r.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_tasks_project_parent ON tasks(project_id, parent_id)`); err != nil {
		return fmt.Errorf("migrate sqlite task parent index: %w", err)
	}
	if err := r.bridgeLegacyTasksToWorkItems(ctx); err != nil {
		return err
	}
	if err := r.seedDefaultKindCatalog(ctx); err != nil {
		return err
	}
	return nil
}

// bridgeLegacyTasksToWorkItems copies legacy task rows into canonical work_items rows.
func (r *Repository) bridgeLegacyTasksToWorkItems(ctx context.Context) error {
	// Keep migration idempotent and non-destructive so existing tasks databases remain readable.
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO work_items(
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, updated_by_actor, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		)
		SELECT
			t.id,
			t.project_id,
			t.parent_id,
			t.kind,
			t.scope,
			t.lifecycle_state,
			t.column_id,
			t.position,
			t.title,
			t.description,
			t.priority,
			t.due_at,
			t.labels_json,
			t.metadata_json,
			t.created_by_actor,
			t.updated_by_actor,
			t.updated_by_type,
			t.created_at,
			t.updated_at,
			t.started_at,
			t.completed_at,
			t.archived_at,
			t.canceled_at
		FROM tasks t
		WHERE NOT EXISTS (
			SELECT 1
			FROM work_items wi
			WHERE wi.id = t.id
		)
	`)
	if err != nil {
		return fmt.Errorf("bridge legacy tasks to work_items: %w", err)
	}
	return nil
}

// seedDefaultKindCatalog inserts built-in kind catalog entries when absent.
func (r *Repository) seedDefaultKindCatalog(ctx context.Context) error {
	type seedRecord struct {
		id          domain.KindID
		displayName string
		description string
		appliesTo   []domain.KindAppliesTo
		parentScope []domain.KindAppliesTo
	}
	records := []seedRecord{
		{id: domain.DefaultProjectKind, displayName: "Project", description: "Built-in project kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToProject}},
		{id: domain.KindID(domain.WorkKindTask), displayName: "Task", description: "Built-in task kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask}},
		{id: domain.KindID(domain.WorkKindSubtask), displayName: "Subtask", description: "Built-in subtask kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToSubtask}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask, domain.KindAppliesToPhase, domain.KindAppliesToBranch}},
		{id: domain.KindID(domain.WorkKindPhase), displayName: "Phase", description: "Built-in phase kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToPhase, domain.KindAppliesToTask}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToBranch, domain.KindAppliesToPhase, domain.KindAppliesToTask}},
		{id: domain.KindID("branch"), displayName: "Branch", description: "Built-in branch kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToBranch}, parentScope: []domain.KindAppliesTo{domain.KindAppliesToBranch}},
		{id: domain.KindID(domain.WorkKindDecision), displayName: "Decision", description: "Built-in decision kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask}},
		{id: domain.KindID(domain.WorkKindNote), displayName: "Note", description: "Built-in note kind", appliesTo: []domain.KindAppliesTo{domain.KindAppliesToTask, domain.KindAppliesToSubtask}},
	}

	now := time.Now().UTC()
	for _, record := range records {
		appliesJSON, err := json.Marshal(record.appliesTo)
		if err != nil {
			return fmt.Errorf("encode kind applies_to for %q: %w", record.id, err)
		}
		parentJSON, err := json.Marshal(record.parentScope)
		if err != nil {
			return fmt.Errorf("encode kind parent scopes for %q: %w", record.id, err)
		}
		templateJSON, err := json.Marshal(domain.KindTemplate{})
		if err != nil {
			return fmt.Errorf("encode kind template for %q: %w", record.id, err)
		}
		_, err = r.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO kind_catalog(
				id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		`, string(record.id), record.displayName, record.description, string(appliesJSON), string(parentJSON), "", string(templateJSON), ts(now), ts(now))
		if err != nil {
			return fmt.Errorf("seed kind_catalog %q: %w", record.id, err)
		}
	}
	return nil
}

// CreateProject creates project.
func (r *Repository) CreateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	kindID := domain.NormalizeKindID(p.Kind)
	if kindID == "" {
		kindID = domain.DefaultProjectKind
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO projects(id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Slug, p.Name, p.Description, string(kindID), string(metaJSON), ts(p.CreatedAt), ts(p.UpdatedAt), nullableTS(p.ArchivedAt))
	return err
}

// UpdateProject updates state for the requested operation.
func (r *Repository) UpdateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	kindID := domain.NormalizeKindID(p.Kind)
	if kindID == "" {
		kindID = domain.DefaultProjectKind
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE projects
		SET slug = ?, name = ?, description = ?, kind = ?, metadata_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, p.Slug, p.Name, p.Description, string(kindID), string(metaJSON), ts(p.UpdatedAt), nullableTS(p.ArchivedAt), p.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetProject returns project.
func (r *Repository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at
		FROM projects
		WHERE id = ?
	`, id)
	return scanProject(row)
}

// ListProjects lists projects.
func (r *Repository) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	query := `
		SELECT id, slug, name, description, kind, metadata_json, created_at, updated_at, archived_at
		FROM projects
	`
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Project{}
	for rows.Next() {
		var (
			p           domain.Project
			kindRaw     string
			metadataRaw string
			createdRaw  string
			updatedRaw  string
			archived    sql.NullString
		)
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &kindRaw, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
			return nil, err
		}
		p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))
		if p.Kind == "" {
			p.Kind = domain.DefaultProjectKind
		}
		if strings.TrimSpace(metadataRaw) == "" {
			metadataRaw = "{}"
		}
		if err := json.Unmarshal([]byte(metadataRaw), &p.Metadata); err != nil {
			return nil, fmt.Errorf("decode project metadata_json: %w", err)
		}
		p.CreatedAt = parseTS(createdRaw)
		p.UpdatedAt = parseTS(updatedRaw)
		p.ArchivedAt = parseNullTS(archived)
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetProjectAllowedKinds replaces one project's kind allowlist.
func (r *Repository) SetProjectAllowedKinds(ctx context.Context, projectID string, kindIDs []domain.KindID) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ErrInvalidID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM project_allowed_kinds WHERE project_id = ?`, projectID); err != nil {
		return err
	}

	now := ts(time.Now().UTC())
	seen := map[domain.KindID]struct{}{}
	for _, raw := range kindIDs {
		kindID := domain.NormalizeKindID(raw)
		if kindID == "" {
			continue
		}
		if _, ok := seen[kindID]; ok {
			continue
		}
		seen[kindID] = struct{}{}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO project_allowed_kinds(project_id, kind_id, created_at)
			VALUES (?, ?, ?)
		`, projectID, string(kindID), now); err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}

// ListProjectAllowedKinds lists one project's explicit kind allowlist.
func (r *Repository) ListProjectAllowedKinds(ctx context.Context, projectID string) ([]domain.KindID, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidID
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT kind_id
		FROM project_allowed_kinds
		WHERE project_id = ?
		ORDER BY kind_id ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.KindID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		kindID := domain.NormalizeKindID(domain.KindID(raw))
		if kindID == "" {
			continue
		}
		out = append(out, kindID)
	}
	return out, rows.Err()
}

// CreateKindDefinition creates one kind catalog entry.
func (r *Repository) CreateKindDefinition(ctx context.Context, kind domain.KindDefinition) error {
	appliesJSON, err := json.Marshal(kind.AppliesTo)
	if err != nil {
		return fmt.Errorf("encode kind applies_to_json: %w", err)
	}
	parentJSON, err := json.Marshal(kind.AllowedParentScopes)
	if err != nil {
		return fmt.Errorf("encode kind allowed_parent_scopes_json: %w", err)
	}
	templateJSON, err := json.Marshal(kind.Template)
	if err != nil {
		return fmt.Errorf("encode kind template_json: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO kind_catalog(
			id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		string(domain.NormalizeKindID(kind.ID)),
		strings.TrimSpace(kind.DisplayName),
		strings.TrimSpace(kind.DescriptionMarkdown),
		string(appliesJSON),
		string(parentJSON),
		strings.TrimSpace(kind.PayloadSchemaJSON),
		string(templateJSON),
		ts(kind.CreatedAt),
		ts(kind.UpdatedAt),
		nullableTS(kind.ArchivedAt),
	)
	return err
}

// UpdateKindDefinition updates one kind catalog entry.
func (r *Repository) UpdateKindDefinition(ctx context.Context, kind domain.KindDefinition) error {
	appliesJSON, err := json.Marshal(kind.AppliesTo)
	if err != nil {
		return fmt.Errorf("encode kind applies_to_json: %w", err)
	}
	parentJSON, err := json.Marshal(kind.AllowedParentScopes)
	if err != nil {
		return fmt.Errorf("encode kind allowed_parent_scopes_json: %w", err)
	}
	templateJSON, err := json.Marshal(kind.Template)
	if err != nil {
		return fmt.Errorf("encode kind template_json: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE kind_catalog
		SET display_name = ?, description_markdown = ?, applies_to_json = ?, allowed_parent_scopes_json = ?, payload_schema_json = ?, template_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`,
		strings.TrimSpace(kind.DisplayName),
		strings.TrimSpace(kind.DescriptionMarkdown),
		string(appliesJSON),
		string(parentJSON),
		strings.TrimSpace(kind.PayloadSchemaJSON),
		string(templateJSON),
		ts(kind.UpdatedAt),
		nullableTS(kind.ArchivedAt),
		string(domain.NormalizeKindID(kind.ID)),
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetKindDefinition loads one kind catalog entry by id.
func (r *Repository) GetKindDefinition(ctx context.Context, kindID domain.KindID) (domain.KindDefinition, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		FROM kind_catalog
		WHERE id = ?
	`, string(domain.NormalizeKindID(kindID)))
	return scanKindDefinition(row)
}

// ListKindDefinitions lists kind catalog entries.
func (r *Repository) ListKindDefinitions(ctx context.Context, includeArchived bool) ([]domain.KindDefinition, error) {
	query := `
		SELECT id, display_name, description_markdown, applies_to_json, allowed_parent_scopes_json, payload_schema_json, template_json, created_at, updated_at, archived_at
		FROM kind_catalog
	`
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY display_name ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.KindDefinition, 0)
	for rows.Next() {
		kind, scanErr := scanKindDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, kind)
	}
	return out, rows.Err()
}

// CreateColumn creates column.
func (r *Repository) CreateColumn(ctx context.Context, c domain.Column) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO columns_v1(id, project_id, name, wip_limit, position, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.ProjectID, c.Name, c.WIPLimit, c.Position, ts(c.CreatedAt), ts(c.UpdatedAt), nullableTS(c.ArchivedAt))
	return err
}

// UpdateColumn updates state for the requested operation.
func (r *Repository) UpdateColumn(ctx context.Context, c domain.Column) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE columns_v1
		SET name = ?, wip_limit = ?, position = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, c.Name, c.WIPLimit, c.Position, ts(c.UpdatedAt), nullableTS(c.ArchivedAt), c.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// ListColumns lists columns.
func (r *Repository) ListColumns(ctx context.Context, projectID string, includeArchived bool) ([]domain.Column, error) {
	query := `
		SELECT id, project_id, name, wip_limit, position, created_at, updated_at, archived_at
		FROM columns_v1
		WHERE project_id = ?
	`
	if !includeArchived {
		query += ` AND archived_at IS NULL`
	}
	query += ` ORDER BY position ASC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Column{}
	for rows.Next() {
		var (
			c          domain.Column
			createdRaw string
			updatedRaw string
			archived   sql.NullString
		)
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.WIPLimit, &c.Position, &createdRaw, &updatedRaw, &archived); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTS(createdRaw)
		c.UpdatedAt = parseTS(updatedRaw)
		c.ArchivedAt = parseNullTS(archived)
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateTask creates task.
func (r *Repository) CreateTask(ctx context.Context, t domain.Task) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return err
	}

	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		if strings.TrimSpace(t.ParentID) == "" {
			scope = domain.KindAppliesToTask
		} else {
			scope = domain.KindAppliesToSubtask
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_items(
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, updated_by_actor, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID,
		t.ProjectID,
		t.ParentID,
		string(t.Kind),
		string(scope),
		string(t.LifecycleState),
		t.ColumnID,
		t.Position,
		t.Title,
		t.Description,
		t.Priority,
		nullableTS(t.DueAt),
		string(labelsJSON),
		string(metadataJSON),
		t.CreatedByActor,
		t.UpdatedByActor,
		string(t.UpdatedByType),
		ts(t.CreatedAt),
		ts(t.UpdatedAt),
		nullableTS(t.StartedAt),
		nullableTS(t.CompletedAt),
		nullableTS(t.ArchivedAt),
		nullableTS(t.CanceledAt),
	)
	if err != nil {
		return err
	}

	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  t.ProjectID,
		WorkItemID: t.ID,
		Operation:  domain.ChangeOperationCreate,
		ActorID:    chooseActorID(t.CreatedByActor, t.UpdatedByActor),
		ActorType:  normalizeActorType(t.UpdatedByType),
		Metadata: map[string]string{
			"column_id": t.ColumnID,
			"position":  strconv.Itoa(t.Position),
			"title":     t.Title,
		},
		OccurredAt: t.CreatedAt,
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// UpdateTask updates state for the requested operation.
func (r *Repository) UpdateTask(ctx context.Context, t domain.Task) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return err
	}

	scope := domain.NormalizeKindAppliesTo(t.Scope)
	if scope == "" {
		if strings.TrimSpace(t.ParentID) == "" {
			scope = domain.KindAppliesToTask
		} else {
			scope = domain.KindAppliesToSubtask
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	prev, err := getTaskByID(ctx, tx, t.ID)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE work_items
		SET parent_id = ?, kind = ?, scope = ?, lifecycle_state = ?, column_id = ?, position = ?, title = ?, description = ?, priority = ?, due_at = ?,
		    labels_json = ?, metadata_json = ?, updated_by_actor = ?, updated_by_type = ?, updated_at = ?, started_at = ?, completed_at = ?, archived_at = ?, canceled_at = ?
		WHERE id = ?
	`,
		t.ParentID,
		string(t.Kind),
		string(scope),
		string(t.LifecycleState),
		t.ColumnID,
		t.Position,
		t.Title,
		t.Description,
		t.Priority,
		nullableTS(t.DueAt),
		string(labelsJSON),
		string(metadataJSON),
		t.UpdatedByActor,
		string(t.UpdatedByType),
		ts(t.UpdatedAt),
		nullableTS(t.StartedAt),
		nullableTS(t.CompletedAt),
		nullableTS(t.ArchivedAt),
		nullableTS(t.CanceledAt),
		t.ID,
	)
	if err != nil {
		return err
	}
	if err := translateNoRows(res); err != nil {
		return err
	}

	op, metadata := classifyTaskTransition(prev, t)
	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  t.ProjectID,
		WorkItemID: t.ID,
		Operation:  op,
		ActorID:    chooseActorID(t.UpdatedByActor, prev.UpdatedByActor),
		ActorType:  normalizeActorType(t.UpdatedByType),
		Metadata:   metadata,
		OccurredAt: t.UpdatedAt,
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// GetTask returns task.
func (r *Repository) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return getTaskByID(ctx, r.db, id)
}

// ListTasks lists tasks.
func (r *Repository) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	query := `
		SELECT
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, updated_by_actor, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		FROM work_items
		WHERE project_id = ?
	`
	if !includeArchived {
		query += ` AND archived_at IS NULL`
	}
	query += ` ORDER BY column_id ASC, position ASC`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// DeleteTask deletes task.
func (r *Repository) DeleteTask(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM work_items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := translateNoRows(res); err != nil {
		return err
	}

	err = insertTaskChangeEvent(ctx, tx, domain.ChangeEvent{
		ProjectID:  task.ProjectID,
		WorkItemID: task.ID,
		Operation:  domain.ChangeOperationDelete,
		ActorID:    chooseActorID(task.UpdatedByActor, task.CreatedByActor),
		ActorType:  normalizeActorType(task.UpdatedByType),
		Metadata: map[string]string{
			"column_id": task.ColumnID,
			"position":  strconv.Itoa(task.Position),
			"title":     task.Title,
		},
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// CreateComment creates comment.
func (r *Repository) CreateComment(ctx context.Context, comment domain.Comment) error {
	commentID := strings.TrimSpace(comment.ID)
	if commentID == "" {
		return domain.ErrInvalidID
	}

	target, err := domain.NormalizeCommentTarget(domain.CommentTarget{
		ProjectID:  comment.ProjectID,
		TargetType: comment.TargetType,
		TargetID:   comment.TargetID,
	})
	if err != nil {
		return err
	}

	bodyMarkdown := strings.TrimSpace(comment.BodyMarkdown)
	if bodyMarkdown == "" {
		return domain.ErrInvalidBodyMarkdown
	}

	authorName := strings.TrimSpace(comment.AuthorName)
	if authorName == "" {
		authorName = "kan-user"
	}
	createdAt := comment.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := comment.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO comments(id, project_id, target_type, target_id, body_markdown, actor_type, author_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		commentID,
		target.ProjectID,
		string(target.TargetType),
		target.TargetID,
		bodyMarkdown,
		string(normalizeActorType(comment.ActorType)),
		authorName,
		ts(createdAt),
		ts(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

// ListCommentsByTarget lists comments for a concrete project target.
func (r *Repository) ListCommentsByTarget(ctx context.Context, target domain.CommentTarget) ([]domain.Comment, error) {
	target, err := domain.NormalizeCommentTarget(target)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, target_type, target_id, body_markdown, actor_type, author_name, created_at, updated_at
		FROM comments
		WHERE project_id = ? AND target_type = ? AND target_id = ?
		ORDER BY created_at ASC, id ASC
	`, target.ProjectID, string(target.TargetType), target.TargetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Comment, 0)
	for rows.Next() {
		comment, scanErr := scanComment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, comment)
	}
	return out, rows.Err()
}

// ListProjectChangeEvents lists recent project events for activity-log consumption.
func (r *Repository) ListProjectChangeEvents(ctx context.Context, projectID string, limit int) ([]domain.ChangeEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, work_item_id, operation, actor_id, actor_type, metadata_json, created_at
		FROM change_events
		WHERE project_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ChangeEvent, 0)
	for rows.Next() {
		var (
			event       domain.ChangeEvent
			opRaw       string
			actorType   string
			metadataRaw string
			createdRaw  string
		)
		if err := rows.Scan(&event.ID, &event.ProjectID, &event.WorkItemID, &opRaw, &event.ActorID, &actorType, &metadataRaw, &createdRaw); err != nil {
			return nil, err
		}
		event.Operation = normalizeChangeOperation(opRaw)
		event.ActorType = normalizeActorType(domain.ActorType(actorType))
		event.OccurredAt = parseTS(createdRaw)
		if strings.TrimSpace(metadataRaw) == "" {
			metadataRaw = "{}"
		}
		if err := json.Unmarshal([]byte(metadataRaw), &event.Metadata); err != nil {
			return nil, fmt.Errorf("decode change_events.metadata_json: %w", err)
		}
		if event.Metadata == nil {
			event.Metadata = map[string]string{}
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

// CreateCapabilityLease creates one capability lease row.
func (r *Repository) CreateCapabilityLease(ctx context.Context, lease domain.CapabilityLease) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO capability_leases(
			instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		strings.TrimSpace(lease.InstanceID),
		strings.TrimSpace(lease.LeaseToken),
		strings.TrimSpace(lease.AgentName),
		strings.TrimSpace(lease.ProjectID),
		string(domain.NormalizeCapabilityScopeType(lease.ScopeType)),
		strings.TrimSpace(lease.ScopeID),
		string(domain.NormalizeCapabilityRole(lease.Role)),
		strings.TrimSpace(lease.ParentInstanceID),
		boolToInt(lease.AllowEqualScopeDelegation),
		ts(lease.IssuedAt),
		ts(lease.ExpiresAt),
		ts(lease.HeartbeatAt),
		nullableTS(lease.RevokedAt),
		strings.TrimSpace(lease.RevokedReason),
	)
	return err
}

// UpdateCapabilityLease updates one capability lease row.
func (r *Repository) UpdateCapabilityLease(ctx context.Context, lease domain.CapabilityLease) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE capability_leases
		SET lease_token = ?, agent_name = ?, project_id = ?, scope_type = ?, scope_id = ?, role = ?, parent_instance_id = ?, allow_equal_scope_delegation = ?, issued_at = ?, expires_at = ?, heartbeat_at = ?, revoked_at = ?, revoke_reason = ?
		WHERE instance_id = ?
	`,
		strings.TrimSpace(lease.LeaseToken),
		strings.TrimSpace(lease.AgentName),
		strings.TrimSpace(lease.ProjectID),
		string(domain.NormalizeCapabilityScopeType(lease.ScopeType)),
		strings.TrimSpace(lease.ScopeID),
		string(domain.NormalizeCapabilityRole(lease.Role)),
		strings.TrimSpace(lease.ParentInstanceID),
		boolToInt(lease.AllowEqualScopeDelegation),
		ts(lease.IssuedAt),
		ts(lease.ExpiresAt),
		ts(lease.HeartbeatAt),
		nullableTS(lease.RevokedAt),
		strings.TrimSpace(lease.RevokedReason),
		strings.TrimSpace(lease.InstanceID),
	)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetCapabilityLease returns one capability lease row by instance id.
func (r *Repository) GetCapabilityLease(ctx context.Context, instanceID string) (domain.CapabilityLease, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		FROM capability_leases
		WHERE instance_id = ?
	`, strings.TrimSpace(instanceID))
	return scanCapabilityLease(row)
}

// ListCapabilityLeasesByScope lists scope-matching leases in deterministic order.
func (r *Repository) ListCapabilityLeasesByScope(ctx context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string) ([]domain.CapabilityLease, error) {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	query := `
		SELECT instance_id, lease_token, agent_name, project_id, scope_type, scope_id, role, parent_instance_id, allow_equal_scope_delegation, issued_at, expires_at, heartbeat_at, revoked_at, revoke_reason
		FROM capability_leases
		WHERE project_id = ? AND scope_type = ?
	`
	args := []any{projectID, string(scopeType)}
	if scopeID != "" {
		query += ` AND scope_id = ?`
		args = append(args, scopeID)
	}
	query += ` ORDER BY issued_at ASC, instance_id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.CapabilityLease, 0)
	for rows.Next() {
		lease, scanErr := scanCapabilityLease(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, lease)
	}
	return out, rows.Err()
}

// RevokeCapabilityLeasesByScope revokes all leases matching one scope filter.
func (r *Repository) RevokeCapabilityLeasesByScope(ctx context.Context, projectID string, scopeType domain.CapabilityScopeType, scopeID string, revokedAt time.Time, reason string) error {
	projectID = strings.TrimSpace(projectID)
	scopeType = domain.NormalizeCapabilityScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "revoked"
	}
	query := `
		UPDATE capability_leases
		SET revoked_at = ?, revoke_reason = ?
		WHERE project_id = ? AND scope_type = ?
	`
	args := []any{ts(revokedAt.UTC()), reason, projectID, string(scopeType)}
	if scopeID != "" {
		query += ` AND scope_id = ?`
		args = append(args, scopeID)
	}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// queryRower represents a query-only DB contract used by DB and Tx implementations.
type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// getTaskByID returns a task using the canonical work_items table.
func getTaskByID(ctx context.Context, q queryRower, id string) (domain.Task, error) {
	row := q.QueryRowContext(ctx, `
		SELECT
			id, project_id, parent_id, kind, scope, lifecycle_state, column_id, position, title, description, priority, due_at, labels_json,
			metadata_json, created_by_actor, updated_by_actor, updated_by_type, created_at, updated_at, started_at, completed_at, archived_at, canceled_at
		FROM work_items
		WHERE id = ?
	`, id)
	return scanTask(row)
}

// execerContext represents a write-only DB contract used by DB and Tx implementations.
type execerContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// insertTaskChangeEvent inserts a change-event ledger record.
func insertTaskChangeEvent(ctx context.Context, execer execerContext, event domain.ChangeEvent) error {
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode change event metadata: %w", err)
	}
	_, err = execer.ExecContext(ctx, `
		INSERT INTO change_events(project_id, work_item_id, operation, actor_id, actor_type, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		event.ProjectID,
		event.WorkItemID,
		string(event.Operation),
		chooseActorID(event.ActorID, "kan-user"),
		string(normalizeActorType(event.ActorType)),
		string(metadataJSON),
		ts(normalizeEventTS(event.OccurredAt)),
	)
	if err != nil {
		return fmt.Errorf("insert change event: %w", err)
	}
	return nil
}

// classifyTaskTransition derives the best operation category and metadata for a task update.
func classifyTaskTransition(prev, next domain.Task) (domain.ChangeOperation, map[string]string) {
	if prev.ArchivedAt == nil && next.ArchivedAt != nil {
		return domain.ChangeOperationArchive, map[string]string{
			"from_state": string(prev.LifecycleState),
			"to_state":   string(next.LifecycleState),
		}
	}
	if prev.ArchivedAt != nil && next.ArchivedAt == nil {
		return domain.ChangeOperationRestore, map[string]string{
			"from_state": string(prev.LifecycleState),
			"to_state":   string(next.LifecycleState),
		}
	}
	if prev.ColumnID != next.ColumnID || prev.Position != next.Position {
		return domain.ChangeOperationMove, map[string]string{
			"from_column_id": prev.ColumnID,
			"to_column_id":   next.ColumnID,
			"from_position":  strconv.Itoa(prev.Position),
			"to_position":    strconv.Itoa(next.Position),
		}
	}
	fields := changedTaskFields(prev, next)
	metadata := map[string]string{}
	if len(fields) > 0 {
		metadata["changed_fields"] = strings.Join(fields, ",")
	}
	return domain.ChangeOperationUpdate, metadata
}

// changedTaskFields identifies a deterministic set of meaningful changes for metadata.
func changedTaskFields(prev, next domain.Task) []string {
	changed := make([]string, 0)
	if prev.ParentID != next.ParentID {
		changed = append(changed, "parent_id")
	}
	if prev.Kind != next.Kind {
		changed = append(changed, "kind")
	}
	if prev.Scope != next.Scope {
		changed = append(changed, "scope")
	}
	if prev.LifecycleState != next.LifecycleState {
		changed = append(changed, "lifecycle_state")
	}
	if prev.Title != next.Title {
		changed = append(changed, "title")
	}
	if prev.Description != next.Description {
		changed = append(changed, "description")
	}
	if prev.Priority != next.Priority {
		changed = append(changed, "priority")
	}
	if !equalNullableTimes(prev.DueAt, next.DueAt) {
		changed = append(changed, "due_at")
	}
	if !equalStringSlices(prev.Labels, next.Labels) {
		changed = append(changed, "labels")
	}
	if !equalMetadata(prev.Metadata, next.Metadata) {
		changed = append(changed, "metadata")
	}
	if prev.UpdatedByActor != next.UpdatedByActor {
		changed = append(changed, "updated_by_actor")
	}
	if prev.UpdatedByType != next.UpdatedByType {
		changed = append(changed, "updated_by_type")
	}
	if !equalNullableTimes(prev.StartedAt, next.StartedAt) {
		changed = append(changed, "started_at")
	}
	if !equalNullableTimes(prev.CompletedAt, next.CompletedAt) {
		changed = append(changed, "completed_at")
	}
	if !equalNullableTimes(prev.CanceledAt, next.CanceledAt) {
		changed = append(changed, "canceled_at")
	}
	return changed
}

// equalStringSlices compares string slices by value and order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalNullableTimes compares nullable timestamps using UTC normalization.
func equalNullableTimes(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UTC().Equal(b.UTC())
}

// equalMetadata compares normalized JSON representations of task metadata.
func equalMetadata(a, b domain.TaskMetadata) bool {
	aJSON, aErr := json.Marshal(a)
	bJSON, bErr := json.Marshal(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// chooseActorID returns the first non-empty actor id or the default local actor.
func chooseActorID(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return "kan-user"
}

// normalizeActorType applies a default when actor type is unset or unsupported.
func normalizeActorType(actorType domain.ActorType) domain.ActorType {
	switch strings.TrimSpace(strings.ToLower(string(actorType))) {
	case string(domain.ActorTypeUser):
		return domain.ActorTypeUser
	case string(domain.ActorTypeAgent):
		return domain.ActorTypeAgent
	case string(domain.ActorTypeSystem):
		return domain.ActorTypeSystem
	default:
		return domain.ActorTypeUser
	}
}

// normalizeChangeOperation canonicalizes persisted operation values.
func normalizeChangeOperation(raw string) domain.ChangeOperation {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case string(domain.ChangeOperationCreate):
		return domain.ChangeOperationCreate
	case string(domain.ChangeOperationUpdate):
		return domain.ChangeOperationUpdate
	case string(domain.ChangeOperationMove):
		return domain.ChangeOperationMove
	case string(domain.ChangeOperationArchive):
		return domain.ChangeOperationArchive
	case string(domain.ChangeOperationRestore):
		return domain.ChangeOperationRestore
	case string(domain.ChangeOperationDelete):
		return domain.ChangeOperationDelete
	default:
		return domain.ChangeOperationUpdate
	}
}

// normalizeEventTS ensures event timestamps are always populated and UTC-normalized.
func normalizeEventTS(in time.Time) time.Time {
	if in.IsZero() {
		return time.Now().UTC()
	}
	return in.UTC()
}

// scanner represents scanner data used by this package.
type scanner interface {
	Scan(dest ...any) error
}

// scanProject handles scan project.
func scanProject(s scanner) (domain.Project, error) {
	var (
		p           domain.Project
		kindRaw     string
		metadataRaw string
		createdRaw  string
		updatedRaw  string
		archived    sql.NullString
	)
	if err := s.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &kindRaw, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, app.ErrNotFound
		}
		return domain.Project{}, err
	}
	p.Kind = domain.NormalizeKindID(domain.KindID(kindRaw))
	if p.Kind == "" {
		p.Kind = domain.DefaultProjectKind
	}
	if strings.TrimSpace(metadataRaw) == "" {
		metadataRaw = "{}"
	}
	if err := json.Unmarshal([]byte(metadataRaw), &p.Metadata); err != nil {
		return domain.Project{}, fmt.Errorf("decode project metadata_json: %w", err)
	}
	p.CreatedAt = parseTS(createdRaw)
	p.UpdatedAt = parseTS(updatedRaw)
	p.ArchivedAt = parseNullTS(archived)
	return p, nil
}

// scanTask handles scan task.
func scanTask(s scanner) (domain.Task, error) {
	var (
		t            domain.Task
		dueRaw       sql.NullString
		labelsRaw    string
		metadataRaw  string
		createdRaw   string
		updatedRaw   string
		startedRaw   sql.NullString
		completedRaw sql.NullString
		archivedRaw  sql.NullString
		canceledRaw  sql.NullString
		priority     string
		kind         string
		scopeRaw     string
		state        string
		updatedType  string
	)
	if err := s.Scan(
		&t.ID,
		&t.ProjectID,
		&t.ParentID,
		&kind,
		&scopeRaw,
		&state,
		&t.ColumnID,
		&t.Position,
		&t.Title,
		&t.Description,
		&priority,
		&dueRaw,
		&labelsRaw,
		&metadataRaw,
		&t.CreatedByActor,
		&t.UpdatedByActor,
		&updatedType,
		&createdRaw,
		&updatedRaw,
		&startedRaw,
		&completedRaw,
		&archivedRaw,
		&canceledRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, app.ErrNotFound
		}
		return domain.Task{}, err
	}
	t.Priority = domain.Priority(priority)
	t.Kind = domain.WorkKind(kind)
	t.Scope = domain.NormalizeKindAppliesTo(domain.KindAppliesTo(scopeRaw))
	t.LifecycleState = domain.LifecycleState(state)
	t.UpdatedByType = domain.ActorType(updatedType)
	t.CreatedAt = parseTS(createdRaw)
	t.UpdatedAt = parseTS(updatedRaw)
	t.StartedAt = parseNullTS(startedRaw)
	t.CompletedAt = parseNullTS(completedRaw)
	t.ArchivedAt = parseNullTS(archivedRaw)
	t.CanceledAt = parseNullTS(canceledRaw)
	t.DueAt = parseNullTS(dueRaw)
	if strings.TrimSpace(metadataRaw) == "" {
		metadataRaw = "{}"
	}
	if err := json.Unmarshal([]byte(metadataRaw), &t.Metadata); err != nil {
		return domain.Task{}, fmt.Errorf("decode metadata_json: %w", err)
	}
	if err := json.Unmarshal([]byte(labelsRaw), &t.Labels); err != nil {
		return domain.Task{}, fmt.Errorf("decode labels_json: %w", err)
	}
	if strings.TrimSpace(string(t.Kind)) == "" {
		t.Kind = domain.WorkKindTask
	}
	if t.Scope == "" {
		if strings.TrimSpace(t.ParentID) == "" {
			t.Scope = domain.KindAppliesToTask
		} else {
			t.Scope = domain.KindAppliesToSubtask
		}
	}
	if t.LifecycleState == "" {
		t.LifecycleState = domain.StateTodo
	}
	if strings.TrimSpace(t.CreatedByActor) == "" {
		t.CreatedByActor = "kan-user"
	}
	if strings.TrimSpace(t.UpdatedByActor) == "" {
		t.UpdatedByActor = t.CreatedByActor
	}
	if t.UpdatedByType == "" {
		t.UpdatedByType = domain.ActorTypeUser
	}
	return t, nil
}

// scanComment handles scan comment.
func scanComment(s scanner) (domain.Comment, error) {
	var (
		comment       domain.Comment
		targetTypeRaw string
		actorTypeRaw  string
		createdRaw    string
		updatedRaw    string
	)
	if err := s.Scan(
		&comment.ID,
		&comment.ProjectID,
		&targetTypeRaw,
		&comment.TargetID,
		&comment.BodyMarkdown,
		&actorTypeRaw,
		&comment.AuthorName,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Comment{}, app.ErrNotFound
		}
		return domain.Comment{}, err
	}
	comment.TargetType = domain.NormalizeCommentTargetType(domain.CommentTargetType(targetTypeRaw))
	if !domain.IsValidCommentTargetType(comment.TargetType) {
		return domain.Comment{}, fmt.Errorf("decode comment target_type %q: %w", targetTypeRaw, domain.ErrInvalidTargetType)
	}
	comment.ActorType = normalizeActorType(domain.ActorType(actorTypeRaw))
	comment.BodyMarkdown = strings.TrimSpace(comment.BodyMarkdown)
	comment.AuthorName = strings.TrimSpace(comment.AuthorName)
	if comment.AuthorName == "" {
		comment.AuthorName = "kan-user"
	}
	comment.CreatedAt = parseTS(createdRaw)
	comment.UpdatedAt = parseTS(updatedRaw)
	return comment, nil
}

// scanKindDefinition decodes one kind_catalog row.
func scanKindDefinition(s scanner) (domain.KindDefinition, error) {
	var (
		kind            domain.KindDefinition
		idRaw           string
		appliesRaw      string
		parentScopesRaw string
		templateRaw     string
		createdRaw      string
		updatedRaw      string
		archivedRaw     sql.NullString
	)
	if err := s.Scan(
		&idRaw,
		&kind.DisplayName,
		&kind.DescriptionMarkdown,
		&appliesRaw,
		&parentScopesRaw,
		&kind.PayloadSchemaJSON,
		&templateRaw,
		&createdRaw,
		&updatedRaw,
		&archivedRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.KindDefinition{}, app.ErrNotFound
		}
		return domain.KindDefinition{}, err
	}
	kind.ID = domain.NormalizeKindID(domain.KindID(idRaw))
	if kind.ID == "" {
		return domain.KindDefinition{}, domain.ErrInvalidKindID
	}
	if strings.TrimSpace(appliesRaw) == "" {
		appliesRaw = "[]"
	}
	if err := json.Unmarshal([]byte(appliesRaw), &kind.AppliesTo); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode applies_to_json: %w", err)
	}
	if strings.TrimSpace(parentScopesRaw) == "" {
		parentScopesRaw = "[]"
	}
	if err := json.Unmarshal([]byte(parentScopesRaw), &kind.AllowedParentScopes); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode allowed_parent_scopes_json: %w", err)
	}
	if strings.TrimSpace(templateRaw) == "" {
		templateRaw = "{}"
	}
	if err := json.Unmarshal([]byte(templateRaw), &kind.Template); err != nil {
		return domain.KindDefinition{}, fmt.Errorf("decode template_json: %w", err)
	}
	kind.CreatedAt = parseTS(createdRaw)
	kind.UpdatedAt = parseTS(updatedRaw)
	kind.ArchivedAt = parseNullTS(archivedRaw)
	return kind, nil
}

// scanCapabilityLease decodes one capability_leases row.
func scanCapabilityLease(s scanner) (domain.CapabilityLease, error) {
	var (
		lease         domain.CapabilityLease
		scopeTypeRaw  string
		roleRaw       string
		allowEqualRaw int
		issuedRaw     string
		expiresRaw    string
		heartbeatRaw  string
		revokedRaw    sql.NullString
	)
	if err := s.Scan(
		&lease.InstanceID,
		&lease.LeaseToken,
		&lease.AgentName,
		&lease.ProjectID,
		&scopeTypeRaw,
		&lease.ScopeID,
		&roleRaw,
		&lease.ParentInstanceID,
		&allowEqualRaw,
		&issuedRaw,
		&expiresRaw,
		&heartbeatRaw,
		&revokedRaw,
		&lease.RevokedReason,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CapabilityLease{}, app.ErrNotFound
		}
		return domain.CapabilityLease{}, err
	}
	lease.ScopeType = domain.NormalizeCapabilityScopeType(domain.CapabilityScopeType(scopeTypeRaw))
	lease.Role = domain.NormalizeCapabilityRole(domain.CapabilityRole(roleRaw))
	lease.AllowEqualScopeDelegation = allowEqualRaw != 0
	lease.IssuedAt = parseTS(issuedRaw)
	lease.ExpiresAt = parseTS(expiresRaw)
	lease.HeartbeatAt = parseTS(heartbeatRaw)
	lease.RevokedAt = parseNullTS(revokedRaw)
	return lease, nil
}

// translateNoRows handles translate no rows.
func translateNoRows(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return app.ErrNotFound
	}
	return nil
}

// boolToInt converts boolean values into sqlite-friendly numeric flags.
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// ts handles ts.
func ts(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// nullableTS handles nullable ts.
func nullableTS(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTS parses input into a normalized form.
func parseTS(v string) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return time.Time{}
	}
	return ts.UTC()
}

// parseNullTS parses input into a normalized form.
func parseNullTS(v sql.NullString) *time.Time {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	ts := parseTS(v.String)
	return &ts
}

// isDuplicateColumnErr reports whether the expected condition is satisfied.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}
