# Kan TUI Manual Test Worksheet (Pre-Phase 11)

Use this worksheet to run end-to-end manual verification of all implemented non-MCP features.
Fill in each response block as you go.

## 0) Test Setup

1. Run with an isolated DB so results are deterministic:
    - `KAN_DB_PATH=/tmp/kan-manual-test.db just run`
2. Keep terminal at least `140x45` if possible.
3. Keep this worksheet open side-by-side and record notes/screenshots.

### Setup Response

- DB path used:
- Terminal size:
- Start timestamp:
- End timestamp:

---

## 1) Startup + Baseline Navigation

### 1.1 App starts and board renders

Actions:

1. Start app.
2. Verify header, columns, summary/info, and bottom help line render.

Expected:

- No startup error.
- Board appears with `To Do`, `In Progress`, `Done`.

Response:

- Pass/Fail:
- Notes:

---

pass

### 1.2 Keyboard navigation

Actions:

1. Press `h/l` and arrow keys left/right for columns.
2. Press `j/k` and arrow keys up/down for task selection.

Expected:

- Selection moves correctly with both vim and arrows.

Response:

- Pass/Fail:
- Notes:

---

pass

## 2) Project Management

### 2.1 Create project

Actions:

1. Press `N`.
2. Fill project fields and save.

Expected:

- New project created and available in picker.

Response:

- Pass/Fail:
- Notes:

what is the icon text? I am not seeing it rendered, also, there is no "edit" project functionality. I also don't see a local path to the project. remember, we need to be able to put a local path the locally saved project and have a way for that to be turned into a relative path as needed when exported and shared. but there is no path logic to a dir. let's discuss fixes and options for this. explain the csv tags thing!

### 2.2 Edit project

Actions:

1. Switch to created project.
2. Press `M`, update metadata, save.

Expected:

- Project details update correctly.

Response:

- Pass/Fail:
- Notes:

now I see there is a way to edit a project, this needs to be more clear and evident, though I may have just been silly so who knows, let's discuss. it does work after knowing how to use it.

### 2.3 Project picker

Actions:

1. Press `p`/`P`.
2. Navigate and select project.

Expected:

- Picker works with keyboard and selection updates board.

Response:

- Pass/Fail:
- Notes:

---

this works, but want to know why "Inbox" is the default "project" also, do we really need to note that p and capital P open the picker? that takes space, and seems to lock off the option for different keys later. I don't mind reserving all "p's" for project picker, just want to discuss pros and cons of making the space taken to show both. I am leaning towards showing and keeping both, but let's discuss

## 3) Task Create/Edit + Due + Labels

### 3.1 Create task

Actions:

1. Press `n`.
2. Fill title, description, priority, due date/time, labels.
3. Save.

Expected:

- Task appears in selected column.

Response:

- Pass/Fail:
- Notes:

the task edit and create needs to show the expected date time format and a way to add time, we apparently don't have the time like we discussed in the plan.md file! explain the inherit labels. also, the helpful hints like hit ctrl + d for picker are obscured by the global stuff below it, it isn't obvious or intuitive. we need to fix that. ask me for a screen cap

### 3.2 Edit task

Actions:

1. Select task.
2. Press `e`.
3. Modify fields and save.

Expected:

- Updated fields persist.

Response:

- Pass/Fail:
- Notes:

I love the attach resource, but this needs to be a part of task creation as well, also, there needs to be a full ability to nav because resources could be outside the project. also, what is attached there? the file as it is or a pointer? I want it to be a pointer. also, the path nav stuff should be based on the project path info if given, not where the cli was launched because this is meant to possibly work with multiple projects at once, although, we could discuss the implications of that and maybe it is better to limit it to the dir it is called in. let's think and discuss. you will need to ask many clarifying questions and what not for this. make sure to ask me about the grand idea, for instance, I think of this as being a global tool, where we eventually may put boundaries on mcp calls, but the dev keeps it running in one window and can manage multiple projects with the agents through the mcp. but it could be a project by project local thing. but we also need to be set up to handle bare repos and worktrees and other common dev workflows, particularly workflows with agents, but also solo devs and so on. so research online and discuss with me and ask questions and so on and so forth!

### 3.3 Due datetime warnings

Actions:

1. Edit task due to a past datetime.
2. Observe warning.
3. Set valid future datetime and save.

Expected:

- Past-due warning shown before save.
- Future datetime accepted.

Response:

- Pass/Fail:
- Notes:

---

pass

## 4) Task Info + Resource Picker

### 4.1 Info modal

Actions:

1. Select task.
2. Press `i` and then `enter` (separately).

Expected:

- Both open task info modal.

Response:

- Pass/Fail:
- Notes:

---

pass

### 4.2 Resource attach flow

Actions:

1. In task info modal press `r` (or from edit mode use `ctrl+r`).
2. Navigate filesystem entries.
3. Attach file and attach directory.

Expected:

- Resource picker opens centered.
- Selected resource is attached to task metadata and reflected in info display.

Response:

- Pass/Fail:
- Notes:

---

I tried this but didn't see any resources attached. my bad, I did'nt see you needed to press 'a' not enter. enter should work or through a warning, it looked like it would have been attached by pressing enter again. also, see above about path and stuff!

## 5) Label Inheritance + Label Picker

### 5.1 Inherited label display

Actions:

1. Create/use a phase parent task with labels.
2. Create a child task under that phase.
3. Open child task info.

Expected:

- Inherited label sources display with global/project/phase context.

Response:

- Pass/Fail:
- Notes:

---

see above

### 5.2 Label picker in task form

Actions:

1. Edit/create task and focus labels field.
2. Press `ctrl+l`.
3. Select inherited label.

Expected:

- Label picker opens centered.
- Selected label appends without duplicates.

Response:

- Pass/Fail:
- Notes:

---

see above

## 6) Search Modal + Filtering

### 6.1 Search focus order and controls

Actions:

1. Press `/`.
2. Tab through: query -> states -> scope -> archived -> apply.
3. Toggle state selections.

Expected:

- Focus order is correct.
- No duplicate labels/fields.

Response:

- Pass/Fail:
- Notes:

---

needs fuzzy finder filtering that displays so the user gets immediate feedback

### 6.2 Clear query vs reset filters

Actions:

1. Apply search with non-default query/filters.
2. Use clear-query command.
3. Use reset-filters command.

Expected:

- Clear query clears text only.
- Reset returns query/states/scope/archived to defaults.

Response:

- Pass/Fail:
- Notes:

---

works, but see above about fuzzy finder and that stuff, let's discuss

## 7) Command Palette + Quick Actions

### 7.1 Command palette behavior

Actions:

1. Press `:`.
2. Type partial command terms.
3. Use `tab` autocomplete and `enter` execute.

Expected:

- Live filtering works.
- Enter runs highlighted command.

Response:

- Pass/Fail:
- Notes:

---

it filters which is good, but doesn't use fuzzy finding stuff so we can do ns for new-subtask and so on. we want it to be fuzzy finding. this goes for the file search too, we need to add file search for resource attachment as well. and we need errors for if someone puts in a path, say by strictly typing and the file doesn't exist.

### 7.2 Quick actions

Actions:

1. Press `.`.
2. Run several actions (edit/info/move/archive).

Expected:

- Quick actions execute on selected task.

Response:

- Pass/Fail:
- Notes:

---

this works, but we should find a way to simplify the menu and have a filter commands by fuzzy finding them!

## 8) Multi-Select + Bulk Actions

### 8.1 Select/unselect

Actions:

1. Select multiple tasks using `space`.
2. Unselect one.
3. Clear selection via `esc`.

Expected:

- Selection count/indicators update correctly.

Response:

- Pass/Fail:
- Notes:

---

appears to work

### 8.2 Bulk operations

Actions:

1. Multi-select tasks.
2. Execute bulk move left/right.
3. Execute bulk archive/delete via palette/quick actions.

Expected:

- All selected tasks update consistently.

Response:

- Pass/Fail:
- Notes:

---

pressing 't' says it is showing archived, but archived isn't on the screen and the screen viewport didn't update in any way other than saying showing archived. i imagine we would need a modal or some other way of showing the list of archived for that project. bulk select move right by using '[' doesn't work, it just moves the one that the cursor is focused on. also, the \* style of showing it is selected is ugly, let's change the color to the charmbracelet pink if it is selected by using space bar. actually, it seems that bulk actions just don't work, also, the quick action menu '.' needs to conditionally limit what is available if multiple items are selected, for instance, you can't edit multiple at once, but you can move or archive. we need logical bulk actions!

## 9) Undo/Redo + Activity Log

### 9.1 Undo/redo

Actions:

1. Perform mutating actions (single + bulk).
2. Press `z` undo.
3. Press `Z` redo.

Expected:

- Reversible actions undo/redo correctly.
- Non-undoable action shows clear status message.

Response:

- Pass/Fail:
- Notes:

---

works and make sense

### 9.2 Activity log modal

Actions:

1. Press `g`.
2. Review event list and timestamps.
3. Close with `esc`.

Expected:

- Activity log opens centered and shows recent events.

Response:

- Pass/Fail:
- Notes:

---

this should allow for moving back to old states or whatever, but remember, not just show the log, this should allow interaction and control. but remember this will eventually be used with an agent llm through an mcp and we will be updating the agent about any changes that happen, so the state log will need to track all actions even going back to a previous state through this menu and through z or Z. and the mcp will need to keep the agent aware of that anytime the agent makes a call, or whatever good methodology we should determine. this will be slightly determined by the use and clarifying questions above, so keep that in mind and we will discuss!

## 10) Subtree Projection + Breadcrumb

### 10.1 Focus subtree

Actions:

1. Select parent task.
2. Press `f` to focus subtree.
3. Navigate board.

Expected:

- Only root + descendants are shown.
- Breadcrumb/focus context visible.

Response:

- Pass/Fail:
- Notes:

---

i think this is user error, but I haven't seen how to. also, this focuses, but it doesn't logically change the screen where we would see the subtasks and all of that nested stuff. I think we really need to plan how this nesting should work and how it would look, so ascii art examples would be good, but also asking clarifying questions so you know what I have in mind and we can better plan would be good. but the state should change. right now, it just changes the view to only show one task, and doesn't change the view in a way that seems to appreciably allow for subtasks, also, task creation and edit should allow for markable subtask things. think about other task management systems like taskwarror, trello, clickup, and so on. we don't need full feature parity, but want similarity in func for stuff like this and need nesting and subtasks, but also nesting with phases that could be at the same level as tasks. we need to plan this out much better and you need to consider how other projects like this work and discuss with me what we are doing different keeping in mind that this will largely be beneficial with coding agents, but also devs or anyone wanting a tui task manager, but we need to plan out this important and base level functionality. how will it work, how will it look, and why, and so on!

### 10.2 Clear subtree focus

Actions:

1. Press `F`.

Expected:

- Full board is restored.

Response:

- Pass/Fail:
- Notes:

---

## 11) Dependency + Rollup Visualization

### 11.1 Project rollup line

Actions:

1. Ensure tasks have dependency metadata.
2. Observe board summary/overview.

Expected:

- Dependency rollup summary appears and updates.

Response:

- Pass/Fail:
- Notes:

---

how do I even do this? how do I add it, how do I check it what? how does this change with the things we are discussing and deciding above?

### 11.2 Task-level dependency hints

Actions:

1. Open task info modal for dependent/blocked task.

Expected:

- `depends_on`, `blocked_by`, and blocked reason hints render clearly.

Response:

- Pass/Fail:
- Notes:

---

explain how I do this and what it is for and how to control it and use it and how it changes with the above!

## 12) Destructive Action Confirmations

### 12.1 Confirm modals

Actions:

1. Trigger `d`, `a`, `D`, `u` actions on task(s).
2. Validate confirm/cancel behavior.

Expected:

- Confirmation modal appears according to config.
- Cancel does not mutate state.

Response:

- Pass/Fail:
- Notes:

---

'u' doesn't do anything!

## 13) Grouping + WIP Warnings (Config-Driven)

### 13.1 Group by priority

Actions:

1. Set `board.group_by = "priority"` in config.
2. Restart and observe board.

Expected:

- Task grouping/ordering reflects priority grouping.

Response:

- Pass/Fail:
- Notes:

---

how do I do this and what is it supposed to do and why? also, we don't have hot reloading of config and stuff like that. we also don't have a tui way of updating the config!

### 13.2 Group by state

Actions:

1. Set `board.group_by = "state"` and restart.

Expected:

- Grouping reflects lifecycle state.

Response:

- Pass/Fail:
- Notes:
  how do I do this and what is it supposed to do and why? also, we don't have hot reloading of config and stuff like that. we also don't have a tui way of updating the config!

### 13.3 WIP warning

Actions:

1. Set a low WIP limit on a column.
2. Move enough tasks into that column.

Expected:

- WIP warning appears when threshold exceeded.

Response:

- Pass/Fail:
- Notes:

---

how do I do this and what is it supposed to do and why? also, we don't have hot reloading of config and stuff like that. we also don't have a tui way of updating the config!

## 14) Help Modal + Discoverability

### 14.1 Help modal content

Actions:

1. Press `?`.
2. Verify new commands are documented (`space`, `g`, `z`, `Z`, `f`, `F`, search semantics).

Expected:

- Help content matches runtime behavior.

Response:

- Pass/Fail:
- Notes:

---

## 15) Final Regression Sweep

### 15.1 No regressions in core flow

Actions:

1. Create -> edit -> move -> archive -> restore -> search -> export/import (if desired).

Expected:

- No crashes, consistent status messages, and persisted state after restart.

Response:

- Pass/Fail:
- Notes:

### 15.2 Screenshots / artifact references

- Screenshot paths:
- GIF/video paths:
- Extra logs:

---

## Final Sign-Off

- Overall result: Pass / Pass with minor issues / Fail
- Critical bugs found:
- Non-critical UX issues:
- Suggested follow-ups:
- Tester name:
- Date:
