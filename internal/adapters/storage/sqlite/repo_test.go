package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/evanschultz/kan/internal/app"
	"github.com/evanschultz/kan/internal/domain"
	_ "modernc.org/sqlite"
)

func TestRepository_ProjectColumnTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "kan.db")
	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Example", "desc", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	loadedProject, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject.Name != "Example" {
		t.Fatalf("unexpected project name %q", loadedProject.Name)
	}

	column, err := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	if err != nil {
		t.Fatalf("NewColumn() error = %v", err)
	}
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}

	due := now.Add(24 * time.Hour)
	task, err := domain.NewTask(domain.TaskInput{
		ID:          "t1",
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Position:    0,
		Title:       "Task title",
		Description: "Task details",
		Priority:    domain.PriorityHigh,
		DueAt:       &due,
		Labels:      []string{"a", "b"},
	}, now)
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	tasks, err := repo.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if len(tasks[0].Labels) != 2 {
		t.Fatalf("unexpected labels %#v", tasks[0].Labels)
	}

	task.Archive(now.Add(1 * time.Hour))
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	activeTasks, err := repo.ListTasks(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListTasks(active) error = %v", err)
	}
	if len(activeTasks) != 0 {
		t.Fatalf("expected 0 active tasks, got %d", len(activeTasks))
	}

	allTasks, err := repo.ListTasks(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListTasks(all) error = %v", err)
	}
	if len(allTasks) != 1 || allTasks[0].ArchivedAt == nil {
		t.Fatalf("expected archived task in full list, got %#v", allTasks)
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
	if _, err := repo.GetTask(ctx, task.ID); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound, got %v", err)
	}
}

func TestRepository_NotFoundCases(t *testing.T) {
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	ctx := context.Background()
	if _, err := repo.GetProject(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for project, got %v", err)
	}
	if _, err := repo.GetTask(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for task, got %v", err)
	}
	if err := repo.DeleteTask(ctx, "missing"); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for delete, got %v", err)
	}
}

func TestRepository_ProjectAndColumnUpdates(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "kan.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Alpha", "desc", now)
	project.Metadata = domain.ProjectMetadata{
		Owner: "owner-1",
		Tags:  []string{"kan"},
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if err := project.Rename("Beta", now.Add(time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}

	activeProjects, err := repo.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects(active) error = %v", err)
	}
	if len(activeProjects) != 1 || activeProjects[0].Name != "Beta" {
		t.Fatalf("unexpected active projects %#v", activeProjects)
	}
	if activeProjects[0].Metadata.Owner != "owner-1" || len(activeProjects[0].Metadata.Tags) != 1 {
		t.Fatalf("expected metadata persisted, got %#v", activeProjects[0].Metadata)
	}

	project.Archive(now.Add(2 * time.Minute))
	if err := repo.UpdateProject(ctx, project); err != nil {
		t.Fatalf("UpdateProject(archive) error = %v", err)
	}

	activeProjects, err = repo.ListProjects(ctx, false)
	if err != nil {
		t.Fatalf("ListProjects(active after archive) error = %v", err)
	}
	if len(activeProjects) != 0 {
		t.Fatalf("expected no active projects, got %#v", activeProjects)
	}

	allProjects, err := repo.ListProjects(ctx, true)
	if err != nil {
		t.Fatalf("ListProjects(all) error = %v", err)
	}
	if len(allProjects) != 1 || allProjects[0].ArchivedAt == nil {
		t.Fatalf("expected archived project in all list, got %#v", allProjects)
	}

	column, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 1, now)
	if err := repo.CreateColumn(ctx, column); err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	if err := column.Rename("Doing", now.Add(3*time.Minute)); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := column.SetPosition(2, now.Add(4*time.Minute)); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	if err := repo.UpdateColumn(ctx, column); err != nil {
		t.Fatalf("UpdateColumn() error = %v", err)
	}

	columns, err := repo.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns() error = %v", err)
	}
	if len(columns) != 1 || columns[0].Name != "Doing" {
		t.Fatalf("unexpected columns %#v", columns)
	}

	column.Archive(now.Add(5 * time.Minute))
	if err := repo.UpdateColumn(ctx, column); err != nil {
		t.Fatalf("UpdateColumn(archive) error = %v", err)
	}
	activeCols, err := repo.ListColumns(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("ListColumns(active) error = %v", err)
	}
	if len(activeCols) != 0 {
		t.Fatalf("expected no active columns, got %#v", activeCols)
	}
	allCols, err := repo.ListColumns(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("ListColumns(all) error = %v", err)
	}
	if len(allCols) != 1 || allCols[0].ArchivedAt == nil {
		t.Fatalf("expected archived column in all list, got %#v", allCols)
	}
}

func TestRepository_MigratesLegacyProjectsTable(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	_, err = db.ExecContext(ctx, `
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create legacy table error = %v", err)
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	project, _ := domain.NewProject("p1", "Legacy", "", time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC))
	project.Metadata = domain.ProjectMetadata{Owner: "evan"}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	loaded, err := repo.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loaded.Metadata.Owner != "evan" {
		t.Fatalf("expected metadata owner to persist after migration, got %#v", loaded.Metadata)
	}
}

func TestRepositoryOpenValidation(t *testing.T) {
	if _, err := Open("   "); err == nil {
		t.Fatal("expected error for empty sqlite path")
	}
}

func TestRepositoryUpdateNotFound(t *testing.T) {
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Now().UTC()
	p, _ := domain.NewProject("missing", "nope", "", now)
	if err := repo.UpdateProject(context.Background(), p); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateProject, got %v", err)
	}

	c, _ := domain.NewColumn("missing-col", "missing", "todo", 0, 0, now)
	if err := repo.UpdateColumn(context.Background(), c); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateColumn, got %v", err)
	}

	tk, _ := domain.NewTask(domain.TaskInput{
		ID:        "missing-task",
		ProjectID: "missing",
		ColumnID:  "missing-col",
		Position:  0,
		Title:     "x",
		Priority:  domain.PriorityLow,
	}, now)
	if err := repo.UpdateTask(context.Background(), tk); err != app.ErrNotFound {
		t.Fatalf("expected app.ErrNotFound for UpdateTask, got %v", err)
	}
}
