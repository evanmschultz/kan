package tui

import "charm.land/bubbles/v2/key"

// keyMap represents key map data used by this package.
type keyMap struct {
	quit           key.Binding
	reload         key.Binding
	toggleHelp     key.Binding
	moveLeft       key.Binding
	moveRight      key.Binding
	moveUp         key.Binding
	moveDown       key.Binding
	addTask        key.Binding
	taskInfo       key.Binding
	editTask       key.Binding
	newProject     key.Binding
	editProject    key.Binding
	commandPalette key.Binding
	quickActions   key.Binding
	deleteTask     key.Binding
	archiveTask    key.Binding
	moveTaskLeft   key.Binding
	moveTaskRight  key.Binding
	hardDeleteTask key.Binding
	restoreTask    key.Binding
	search         key.Binding
	projects       key.Binding
	toggleArchived key.Binding
}

// newKeyMap constructs key map.
func newKeyMap() keyMap {
	return keyMap{
		quit:           key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		reload:         key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		toggleHelp:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		moveLeft:       key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "column left")),
		moveRight:      key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "column right")),
		moveUp:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "task up")),
		moveDown:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "task down")),
		addTask:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new task")),
		taskInfo:       key.NewBinding(key.WithKeys("i", "enter"), key.WithHelp("i/enter", "task info")),
		editTask:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit task")),
		newProject:     key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "new project")),
		editProject:    key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "edit project")),
		commandPalette: key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command palette")),
		quickActions:   key.NewBinding(key.WithKeys("."), key.WithHelp(".", "quick actions")),
		deleteTask:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete (default)")),
		archiveTask:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive task")),
		moveTaskLeft:   key.NewBinding(key.WithKeys("["), key.WithHelp("[", "move task left")),
		moveTaskRight:  key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "move task right")),
		hardDeleteTask: key.NewBinding(key.WithKeys("D", "shift+d"), key.WithHelp("D", "hard delete")),
		restoreTask:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "restore task")),
		search:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		projects:       key.NewBinding(key.WithKeys("p", "P"), key.WithHelp("p/P", "project picker")),
		toggleArchived: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "toggle archived")),
	}
}

// ShortHelp handles short help.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.addTask, k.taskInfo, k.editTask, k.newProject, k.commandPalette, k.quickActions, k.search, k.quit,
	}
}

// FullHelp handles full help.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.addTask, k.taskInfo, k.editTask, k.newProject, k.editProject, k.commandPalette, k.quickActions, k.search, k.projects, k.toggleArchived, k.toggleHelp, k.reload, k.quit},
		{k.moveLeft, k.moveRight, k.moveUp, k.moveDown, k.moveTaskLeft, k.moveTaskRight},
		{k.deleteTask, k.archiveTask, k.hardDeleteTask, k.restoreTask},
	}
}
