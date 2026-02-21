package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
			column_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL,
			due_at TEXT,
			labels_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT,
			FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY(column_id) REFERENCES columns_v1(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_columns_project_position ON columns_v1(project_id, position);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_column_position ON tasks(project_id, column_id, position);`,
	}

	for _, stmt := range stmts {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	if _, err := r.db.ExecContext(ctx, `ALTER TABLE projects ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("migrate sqlite add projects.metadata_json: %w", err)
	}
	return nil
}

// CreateProject creates project.
func (r *Repository) CreateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO projects(id, slug, name, description, metadata_json, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Slug, p.Name, p.Description, string(metaJSON), ts(p.CreatedAt), ts(p.UpdatedAt), nullableTS(p.ArchivedAt))
	return err
}

// UpdateProject updates state for the requested operation.
func (r *Repository) UpdateProject(ctx context.Context, p domain.Project) error {
	metaJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("encode project metadata: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE projects
		SET slug = ?, name = ?, description = ?, metadata_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, p.Slug, p.Name, p.Description, string(metaJSON), ts(p.UpdatedAt), nullableTS(p.ArchivedAt), p.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetProject returns project.
func (r *Repository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, description, metadata_json, created_at, updated_at, archived_at
		FROM projects
		WHERE id = ?
	`, id)
	return scanProject(row)
}

// ListProjects lists projects.
func (r *Repository) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	query := `
		SELECT id, slug, name, description, metadata_json, created_at, updated_at, archived_at
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
			metadataRaw string
			createdRaw  string
			updatedRaw  string
			archived    sql.NullString
		)
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
			return nil, err
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
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO tasks(id, project_id, column_id, position, title, description, priority, due_at, labels_json, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ProjectID, t.ColumnID, t.Position, t.Title, t.Description, t.Priority, nullableTS(t.DueAt), string(labelsJSON), ts(t.CreatedAt), ts(t.UpdatedAt), nullableTS(t.ArchivedAt))
	return err
}

// UpdateTask updates state for the requested operation.
func (r *Repository) UpdateTask(ctx context.Context, t domain.Task) error {
	labelsJSON, err := json.Marshal(t.Labels)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET column_id = ?, position = ?, title = ?, description = ?, priority = ?, due_at = ?, labels_json = ?, updated_at = ?, archived_at = ?
		WHERE id = ?
	`, t.ColumnID, t.Position, t.Title, t.Description, t.Priority, nullableTS(t.DueAt), string(labelsJSON), ts(t.UpdatedAt), nullableTS(t.ArchivedAt), t.ID)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// GetTask returns task.
func (r *Repository) GetTask(ctx context.Context, id string) (domain.Task, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, column_id, position, title, description, priority, due_at, labels_json, created_at, updated_at, archived_at
		FROM tasks
		WHERE id = ?
	`, id)
	return scanTask(row)
}

// ListTasks lists tasks.
func (r *Repository) ListTasks(ctx context.Context, projectID string, includeArchived bool) ([]domain.Task, error) {
	query := `
		SELECT id, project_id, column_id, position, title, description, priority, due_at, labels_json, created_at, updated_at, archived_at
		FROM tasks
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
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return translateNoRows(res)
}

// scanner represents scanner data used by this package.
type scanner interface {
	Scan(dest ...any) error
}

// scanProject handles scan project.
func scanProject(s scanner) (domain.Project, error) {
	var (
		p           domain.Project
		metadataRaw string
		createdRaw  string
		updatedRaw  string
		archived    sql.NullString
	)
	if err := s.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &metadataRaw, &createdRaw, &updatedRaw, &archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, app.ErrNotFound
		}
		return domain.Project{}, err
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
		t          domain.Task
		dueRaw     sql.NullString
		labelsRaw  string
		createdRaw string
		updatedRaw string
		archived   sql.NullString
		priority   string
	)
	if err := s.Scan(&t.ID, &t.ProjectID, &t.ColumnID, &t.Position, &t.Title, &t.Description, &priority, &dueRaw, &labelsRaw, &createdRaw, &updatedRaw, &archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, app.ErrNotFound
		}
		return domain.Task{}, err
	}
	t.Priority = domain.Priority(priority)
	t.CreatedAt = parseTS(createdRaw)
	t.UpdatedAt = parseTS(updatedRaw)
	t.ArchivedAt = parseNullTS(archived)
	t.DueAt = parseNullTS(dueRaw)
	if err := json.Unmarshal([]byte(labelsRaw), &t.Labels); err != nil {
		return domain.Task{}, fmt.Errorf("decode labels_json: %w", err)
	}
	return t, nil
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
