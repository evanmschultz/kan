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

// TestRepository_ProjectColumnTaskLifecycle verifies behavior for the covered scenario.
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

// TestRepository_CreateAndListCommentsByTarget verifies behavior for the covered scenario.
func TestRepository_CreateAndListCommentsByTarget(t *testing.T) {
	ctx := context.Background()
	repo, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("p1", "Example", "", now)
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	comment2, err := domain.NewComment(domain.CommentInput{
		ID:           "c2",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "t1",
		BodyMarkdown: "second",
		ActorType:    domain.ActorType("AGENT"),
		AuthorName:   "agent-1",
	}, now)
	if err != nil {
		t.Fatalf("NewComment(c2) error = %v", err)
	}
	comment1, err := domain.NewComment(domain.CommentInput{
		ID:           "c1",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "t1",
		BodyMarkdown: "first",
		ActorType:    domain.ActorTypeUser,
		AuthorName:   "user-1",
	}, now)
	if err != nil {
		t.Fatalf("NewComment(c1) error = %v", err)
	}
	projectComment, err := domain.NewComment(domain.CommentInput{
		ID:           "c3",
		ProjectID:    project.ID,
		TargetType:   domain.CommentTargetTypeProject,
		TargetID:     project.ID,
		BodyMarkdown: "project note",
		ActorType:    domain.ActorTypeSystem,
		AuthorName:   "kan",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("NewComment(c3) error = %v", err)
	}

	if err := repo.CreateComment(ctx, comment2); err != nil {
		t.Fatalf("CreateComment(c2) error = %v", err)
	}
	if err := repo.CreateComment(ctx, comment1); err != nil {
		t.Fatalf("CreateComment(c1) error = %v", err)
	}
	if err := repo.CreateComment(ctx, projectComment); err != nil {
		t.Fatalf("CreateComment(c3) error = %v", err)
	}

	taskComments, err := repo.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeTask,
		TargetID:   "t1",
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget(task) error = %v", err)
	}
	if len(taskComments) != 2 {
		t.Fatalf("expected 2 task comments, got %d", len(taskComments))
	}
	if taskComments[0].ID != "c1" || taskComments[1].ID != "c2" {
		t.Fatalf("expected deterministic created_at/id ordering, got %#v", taskComments)
	}
	if taskComments[1].ActorType != domain.ActorTypeAgent {
		t.Fatalf("expected normalized actor type agent, got %q", taskComments[1].ActorType)
	}

	comments, err := repo.ListCommentsByTarget(ctx, domain.CommentTarget{
		ProjectID:  project.ID,
		TargetType: domain.CommentTargetTypeProject,
		TargetID:   project.ID,
	})
	if err != nil {
		t.Fatalf("ListCommentsByTarget(project) error = %v", err)
	}
	if len(comments) != 1 || comments[0].ID != "c3" {
		t.Fatalf("unexpected project comments %#v", comments)
	}
}

// TestRepository_NotFoundCases verifies behavior for the covered scenario.
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

// TestRepository_ProjectAndColumnUpdates verifies behavior for the covered scenario.
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

// TestRepository_MigratesLegacyProjectsTable verifies behavior for the covered scenario.
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

// TestRepository_MigratesLegacyTasksTable verifies behavior for the covered scenario.
func TestRepository_MigratesLegacyTasksTable(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-tasks.db")
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	legacySchema := []string{
		`CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE columns_v1 (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			wip_limit INTEGER NOT NULL DEFAULT 0,
			position INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			archived_at TEXT
		)`,
		`CREATE TABLE tasks (
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
			archived_at TEXT
		)`,
	}
	for _, stmt := range legacySchema {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("create legacy schema error = %v", err)
		}
	}
	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	for _, stmt := range []string{
		`INSERT INTO projects(id, slug, name, description, created_at, updated_at, archived_at)
		 VALUES ('p1', 'legacy', 'Legacy', '', '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
		`INSERT INTO columns_v1(id, project_id, name, wip_limit, position, created_at, updated_at, archived_at)
		 VALUES ('c1', 'p1', 'To Do', 0, 0, '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
		`INSERT INTO tasks(id, project_id, column_id, position, title, description, priority, due_at, labels_json, created_at, updated_at, archived_at)
		 VALUES ('t1', 'p1', 'c1', 0, 'Legacy task', 'desc', 'medium', NULL, '["legacy"]', '` + now.Format(time.RFC3339Nano) + `', '` + now.Format(time.RFC3339Nano) + `', NULL)`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed legacy rows error = %v", err)
		}
	}

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() on legacy task db error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	rows, err := repo.db.QueryContext(ctx, `PRAGMA table_info(tasks)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(tasks) error = %v", err)
	}
	t.Cleanup(func() {
		_ = rows.Close()
	})

	seenParentID := false
	for rows.Next() {
		var (
			cid        int
			name       string
			colType    string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		if name == "parent_id" {
			seenParentID = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}
	if !seenParentID {
		t.Fatal("expected parent_id column to be added during migration")
	}

	var workItemCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_items WHERE id = 't1'`).Scan(&workItemCount); err != nil {
		t.Fatalf("count work_items error = %v", err)
	}
	if workItemCount != 1 {
		t.Fatalf("expected migrated work_items row count 1, got %d", workItemCount)
	}
	loaded, err := repo.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask() migrated row error = %v", err)
	}
	if loaded.Title != "Legacy task" || loaded.ProjectID != "p1" {
		t.Fatalf("unexpected migrated task %#v", loaded)
	}
	if loaded.Kind != domain.WorkKindTask || loaded.LifecycleState != domain.StateTodo {
		t.Fatalf("expected default kind/state migration values, got kind=%q state=%q", loaded.Kind, loaded.LifecycleState)
	}

	var tableCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='change_events'`).Scan(&tableCount); err != nil {
		t.Fatalf("count change_events table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected change_events table to exist after migration, got %d", tableCount)
	}
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='comments'`).Scan(&tableCount); err != nil {
		t.Fatalf("count comments table error = %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected comments table to exist after migration, got %d", tableCount)
	}

	var indexCount int
	if err := repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_comments_project_target_created_at'`).Scan(&indexCount); err != nil {
		t.Fatalf("count comments index error = %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected comments target index to exist after migration, got %d", indexCount)
	}
}

// TestRepositoryOpenValidation verifies behavior for the covered scenario.
func TestRepositoryOpenValidation(t *testing.T) {
	if _, err := Open("   "); err == nil {
		t.Fatal("expected error for empty sqlite path")
	}
}

// TestRepositoryUpdateNotFound verifies behavior for the covered scenario.
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

// TestRepository_ListProjectChangeEventsLifecycle verifies behavior for the covered scenario.
func TestRepository_ListProjectChangeEventsLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	now := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("p1", "Events", "", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	todo, _ := domain.NewColumn("c1", project.ID, "To Do", 0, 0, now)
	done, _ := domain.NewColumn("c2", project.ID, "Done", 1, 0, now)
	if err := repo.CreateColumn(ctx, todo); err != nil {
		t.Fatalf("CreateColumn(todo) error = %v", err)
	}
	if err := repo.CreateColumn(ctx, done); err != nil {
		t.Fatalf("CreateColumn(done) error = %v", err)
	}

	task, _ := domain.NewTask(domain.TaskInput{
		ID:             "t1",
		ProjectID:      project.ID,
		ColumnID:       todo.ID,
		Position:       0,
		Title:          "Track me",
		Priority:       domain.PriorityMedium,
		CreatedByActor: "user-1",
		UpdatedByActor: "user-1",
		UpdatedByType:  domain.ActorTypeUser,
	}, now)
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if err := task.UpdateDetails("Track me v2", task.Description, task.Priority, task.DueAt, task.Labels, now.Add(time.Minute)); err != nil {
		t.Fatalf("UpdateDetails() error = %v", err)
	}
	task.UpdatedByActor = "agent-1"
	task.UpdatedByType = domain.ActorTypeAgent
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(update) error = %v", err)
	}

	if err := task.Move(done.ID, 1, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	task.UpdatedByActor = "user-2"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(move) error = %v", err)
	}

	task.Archive(now.Add(3 * time.Minute))
	task.UpdatedByActor = "user-3"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(archive) error = %v", err)
	}

	task.Restore(now.Add(4 * time.Minute))
	task.UpdatedByActor = "user-4"
	task.UpdatedByType = domain.ActorTypeUser
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("UpdateTask(restore) error = %v", err)
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}

	events, err := repo.ListProjectChangeEvents(ctx, project.ID, 10)
	if err != nil {
		t.Fatalf("ListProjectChangeEvents() error = %v", err)
	}
	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d (%#v)", len(events), events)
	}

	wantOps := []domain.ChangeOperation{
		domain.ChangeOperationDelete,
		domain.ChangeOperationRestore,
		domain.ChangeOperationArchive,
		domain.ChangeOperationMove,
		domain.ChangeOperationUpdate,
		domain.ChangeOperationCreate,
	}
	for i, want := range wantOps {
		if events[i].Operation != want {
			t.Fatalf("unexpected event operation at index %d: got %q want %q", i, events[i].Operation, want)
		}
	}

	if events[3].Metadata["from_column_id"] != todo.ID || events[3].Metadata["to_column_id"] != done.ID {
		t.Fatalf("expected move metadata to include column transition, got %#v", events[3].Metadata)
	}
	if events[5].ActorID != "user-1" {
		t.Fatalf("expected create actor user-1, got %q", events[5].ActorID)
	}
}
