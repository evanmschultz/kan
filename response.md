You’re making sense, and we’re aligned now.

What I meant by track-only vs execution-enabled:

- kan itself is always tracking/planning.
- “Execution-enabled” was only shorthand for future external policy in MCP/orchestrators, not kan running code.
- I’ll rename this in docs to avoid confusion.

On your path/capability concern (unlinked/not agent-eligible wording):

- Agreed, that wording is too hard.
- Better model:
    - planning_ready: always true (you can always use kan).
    - workspace_linked: true/false (has validated local root mapping).
    - automation_context_ready: true/false (linked root + required metadata for agent workflows).
- So unlinked projects are still usable; they just can’t provide file-context workflows reliably.

Import/export and path verification:

- I agree with your stricter direction.
- Import should require mapping each root_alias and then validating referenced relative paths.
- No absolute paths ever exported.
- If a referenced path is missing, import should fail by default, with an explicit user bypass flow.
- This is the right “git-like” trust check for “same repo/branch-ish context.”

Single DB vs multiple DB files:

- Recommendation: one local DB by default, project-level export/import for sharing.
- Why:
    - easier global search, blockers across projects, activity timeline, one config surface.
    - fewer failure modes than many DB files.
- Sharing unit should be per-project package/export, not “copy whole DB.”
- We can still support “one DB per project” later if needed, but default should be single DB.

You asked to resurface items 3 and 4:

- 3 Archived/hierarchy/dependency UX: full-screen archive view, tree-in-board hierarchy first, relation picker first.
- 4 Storage backend: SQLite now, relational schema for hierarchy/blockers/dependencies, roadmap path for remote service later (likely Postgres-capable backend), no graph DB needed now.

Recovered 5–7 from prior cutoff:

- 5 Undo: checkpoint-oriented undo/revert, not noisy per-keystroke history.
- 6 Agent updates: meaningful delta + current snapshot, not all intermediate actions.
- 7 Nesting UX: start with board continuity and focused subtree; consider fully re-rendered nested boards later if needed.

Your big-goal statement is clear now:

- Primary product goal: better human+LLM progress control than markdown files.
- Secondary: still usable as personal local tracker.
- Pre-Phase-11: build internal guardrails/checks now.
- MCP/tool integrations: roadmap only, no implementation now.

I agree with your hierarchy direction:

- Project -> Branch -> Phase (nestable subphases) -> Task -> Subtask
- Rules:
    - projects not nestable,
    - branches only under project,
    - phases nestable,
    - tasks own subtasks,
    - subtasks use complete/not-complete,
    - phases/tasks use todo | in_progress | done | archived.

Completion gates (pre-Phase-11, no MCP):

- move to done blocked unless children complete, blockers clear (or explicit override with reason), and completion criteria pass.
- move to in_progress can require criteria prompt/check if configured.
- this matches your “no fake completion” goal.

High-value clarifying questions (big-picture, not implementation trivia):

1. Should every project auto-create a default branch (for non-dev simplicity), or should branch be optional?
2. Should branch UI be hidden until enabled, so personal-todo users don’t see dev complexity?
3. Import default: hard-fail on missing referenced relative paths, with explicit bypass only, correct?
4. On bypass import, should project be automatically marked automation_context_ready=false until resolved?
5. Should completion criteria be required by default for phases/tasks, or opt-in per project/template?

If you confirm these, I’ll update PLAN.md, PRE_PHASE11_CLOSEOUT_DISCUSSION.md, and README.md in one pass with this exact consensus.

User response:
'On your path/capability concern (unlinked/not agent-eligible wording):

- Agreed, that wording is too hard.
- Better model:
    - planning_ready: always true (you can always use kan).
    - workspace_linked: true/false (has validated local root mapping).
    - automation_context_ready: true/false (linked root + required metadata for agent workflows).
- So unlinked projects are still usable; they just can’t provide file-context workflows reliably.':

so this would be a config.toml thing that would be updated through the tui? also, what is planning ready? that wording sounds weird. also automation_context_ready. shouldn't it just be dir linked or not? the llm could work with a dev regardless, we shouldn't complicate it. we just need to internally know if we can resolve file paths and share. so, maybe it shareable is the thing? no, I think it is just workspace linked or not. two things. we assume the llm agent is sandboxed. so, if someone wants to attach resources for now they need to use urls if they can't do it with relative paths after first linking the project to a dir on their system, and we still need to make sure that all other resource links for files or dirs/ are path resolved on import and we just fail for now. roadmap of roadmap meaning far down in the roadmap we plan a way to work with the user and llm or whatever to resolve paths if there is a git divergence or something like that. but if someone is importing a kan db that is linked to paths even just at the task level, right now we just fail if that can't be resolved with relative paths based on the linking at the first import/ingestion of the db time. we don't want people to import a kan db for a shared project thinking they are working on the same thing just to find out they are on the wrong branch or git commit or something and we will treat all things as if they were dev stuff. we will refine later so roadmap should mention future ideas and possibilities you come up with!

`Single DB vs multiple DB files:`
will your recommendation be easy to implement? will we be able to partially export db's and import a file and merge them keeping everything right without issue easily? I imagine, just want to make sure we are lessening work and effort for us and making things work best from the start. so think and discuss, but know I do generally agree and see what you are saying, just want confirmation.... that being said, don't give me confirmation if I am wrong in my interpretation. we want this to be good and you are not a sycophant!

`High-value clarifying questions (big-picture, not implementation trivia):`

1.  interesting, so you are saying we treat everything, even non-dev stuff like git would. force good development practices, or encourage them, even for non-devs. so, we have a project | (default branch is 'main') | potentially other branches, all under the the 'project'? did I understadn you? I do kind of like that. it does imply we should have some "merge | rebase" logic, but we can put that in the roadmap. I am definitely leaning towards auto branch created and it is main and the project view in the tui is always just main if that is the only branch. it does simplify the ui in a way too. we just need to make sure the tui shows that clearly, and we need to build the sql tables out right to work with a project table basically just having pointers to branches, but I like it. do you agree or should we do something else?
2.  it shouldn't be "hidden" they don't need to use it. let's just keep it and if they don't want it they won't use it. remember, this is a tui thing right now, so mainly only devs will use it, that is a concern for when we need to make a web version. so put that in the roadmap section as a possible thing to consider if users request it!
3.  for now yes, add that we need to discuss, research, think about resolutions in the future in the roadmap section of the @plan.md file!
4.  `automation_context_ready=false until resolved?` wtf? why this wording still? did you NOT understand what I said about workspace linked or not linked? and why automation? that wording implies we do things with llms. this is just a tool to make dev's and llms faster and better together. did I say anything that made you think this was the right wording and that this should be a config state at all anymore? if so, please tell me what I said that implied that, I want to know where I went wrong!
5.  it should be required. all subtasks, all tasks need to be finished to mark something as complete. you can't do it without that. roadmap for the mcp, we will send a message about that because it failed, but even if they are marked as complete we send a message to make the llm respond with a yes to confirm they finished the list of meta things that are put their for task types and phase levels and phase types and other configurable stuff to make sure that the llm agent always stays honest and runs all tests and what not before marking things complete! that is roadmap stuff. make sure the @plan.md file shows that we will need to discuss that in great detail before we start building the mcp/http!
