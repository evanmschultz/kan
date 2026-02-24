# PRE-MCP USER NOTES ADDITIONS

We need to add somethings to the PRE_MCP_EXECUTION_WAVES.md file.

Remember, Kan is meant to make it easy to work with AI agents and fix issues found with using MD files to track everything for project management, but could also be used as a traditional solo user task management system like trello. So keep that in mind when thinking about all the below and answering questions and so on!

## ONE - ADD TYPE FIELDS TO ALL MODELS

We need to add 'type' fields for everything, [ Project | Phase | Subphase (which is really just a phase) | Branch | (Subbranch which is just a branch) | Task | Subtask (which is really just a task) ]. The type fields will be definable through the in the tui, and when we make the mcp an Orchestrator agent could update them, and saved at the db root with all other db level stuff like `user_name`. When launching a project it will ask you if you want to include types, (it must be editable in the 'M' modal as well and programmatically from the orchestrator agent as well). When adding or editing allowed/expected types for a project we must use the same fuzzy finding stuff used in path and command pallette and other stuff, so hopefully we can easily reuse that, investigate our code and tell me how easily it will be to keep this kind of stuff DRY. It would list what available types from the db root (ones that are already defined) or you could add a new one. Adding a new one would add them to the db root types section, and to the project's `allowed_types` would have pointers to that that the tui and http (http based MCP if local) so the information about them could be seen in the tui or by the agent or remotely in another tui.

### REASON/PURPOSE

The purpose and use of types will be to programmatically add data, fields and things on node creation (what should we call these, what is the common name for them in things like trello and other task managers?).

For Projects:
It would include things like `allowed_types` list that would auto fill but would be editable at that project level. Other metadata, common practices/expectations like TDD, Hexagonal, go idioms and stuff like that that would make project creation and management easier and faster.

add autofill subtasks and shit like that and have the `owner` of the auto filled stuff be `system`, things like task and subtask creation for updating docs after changes, running tests and so on!

### QUESTIONS

1. Is there an issue with exporting SQLite db files from a mac and using them on other OS's? Should we consider doing something to fix this?
2. Is our code setup to be able to share all the necessary data, making a new db file when exporting specific projects instead of the whole db, because user's won't want to share ALL their personal projects. So, we need to make sure that exporting will get all of the data needed for a project, including all of the data for "allowed_types" that would be stored at the db level. Or would it be better to have the `allowed_types` be stored (with full data) at the project level but allow visibility from other projects to see them and copy them if they want to be reused instead of using `pointers`? This does seem problematic because we will want to gate keep project access, and the logic to copy the whole thing would be basically just as complicated as copying it on export and having migration logic when importing a project db file. So, I think we shouldn't do that but want to hear your thoughts!
3. Should we name them something other than `allowed_types`? Also, how do we, and how much do we need to separate them by node type (whatever we should call that, meaning project, branch, task, phase)? I would like to hear your thoughts!
4. what am I missing from this idea that would be helpful or needed? I know I am forgetting some details I thought of, and also likely didn't think of everything needed.

Remember the purpose of Kan!

## TWO - FAST AUTO FILL FILES IN A DIR/PROJECT

We would want things like the ability to find or add files in a dir, like AGENTS.md, CLAUDE.md and check if a section exists and if it has the exact language for a project type and ask if the user wants to add those sections or create the files and add those sections.

File creation for things like project, plan or other db files info, basically export db info to md file/s inside a dir but more general that what was just described.

### NOTES

This couldn't be done if the `project_path` is a file. So, we should prevent those actions and not show them in a tui if the project path is a file. Also, we need to add a confirm modal when a user tries to add a file as the project path.

### QUESTIONS

1. Should we make the ability to add multiple paths to a project, like we have now, impossible or just make it so user's would have to pick which path to add/export/edit files to? Discuss pros and cons.
2. what am I missing from this idea that would be helpful or needed? I know I am forgetting some details I thought of, and also likely didn't think of everything needed.

Remember the purpose of Kan!

## THREE - GATE KEEPING

We need the ability to gate keep projects and sections. What we want is the ability for an Orchestrator agent to be gate kept to a project. It would need some kind of token id (agent identifier), this is different than auth that would be at the system level, and something we are not worried about right now. In the db and in the tui, any changes made my an agent would say their `name` and id. For all subagents and things we would want to gate keep them to certain branches | phases and what not. This means we will need a unique id generation system and a way to tell the orchestrator agent (which could be gate kept to branches and phases as well, not just project, because there could be multiple orchestrators as the user could be working with multiple codex or claude code or whatever client instances at any given time and would probably be working on the branch level, but the user can decide how and what to limit the orchestrators to and the subagents for that matter).

This means that we need a system to call the mcp to get all the ids and what not to the agent without the user needing to generate them in the tui, but the user could generate them in the tui if desired. The orchestrator would need a way to call it first and then have an id generated (only one orchestrator agent id could exist at a level at one time). that call would give them their id that they would use as an arg anytime they call the db from then on. (they would need the level path as an arg for that too so they get properly gate kept after that, this assumes we have some trust that the orchestrator won't call it again without the user's approval so we need to note that in the readme and everything). the orchestrator then could make calls to generate and get new subagent ids that they would give to their subagents to use to make updates to the subagents gate kept levels, which should be lower than the orchestrator but doesn't need to be (the readme should say this and we would need all that information in the auto generatable/copyable Agents md claude md stuff). the all agent calls to the db aside from initial orchestrator (with db level path (what should that be called) would need an id token). When generating subagents the orchestrator would need to add a list of agent names so we can save that with the id so the human readable name would be saved on any edits for tracking purposes, and so the orchestrator could track it beyond the id they gave them as well.

each agent would have their own id and name and those would need to be passed as enum type safe args to call the mcp (need both the id and their name). we will need a way to invalidate old name/id pairs (clean-up and kill access token type stuff!)

### QUESTIONS

1. I feel like I missing a lot of ideas and things for this, please think and discuss all of this with me. ask any and all clarifying questions to get an understanding of goals (ask about goals first, before asking about specifics because the goals will help inform the specifics type questions!)
2. considering use, and the fact that there would be multiple orchestrators at a given level over time, think about starting a new codex session, should each orchestrator have their own unique name and id? I am thinking yes, but should we at least return a warning if two orchestrator id's are still valid at a given time to prevent double work at a level path? I would think it should just be a warning.
3. should names be repeatable? and just invalidate name/id pairs but allow a repeated name or should each name be repeated as well?
4. what am I missing from this idea that would be helpful or needed? I know I am forgetting some details I thought of, and also likely didn't think of everything needed.
5. just generally, how can this non-auth based gate keeping work?

Remember the purpose of Kan

### NOTES

1. Note that the user/owner of the whole db should never be gate kept. but eventually we will want ways to do this remotely and allow teams to work together so we will need a gate keeping system like this for users, so we need to build this where it is easy to extend it to users, but right now will just be for agents, unless it would be trivial to add the user gate keeping stuff just don't use it yet. what do you think? discuss after long consideration.

## THINGS MISSED

Return info when we build the http and http mcp server stuff.

1. We will need ways to limit the response size and amount, for instance the calls probably shouldn't return all the edits made to a task and comments or whatever as an example, it probably should just take the final state, but should be able to call and get more if desired. does this mean we should use a graphql shape or would rest be best still? think and explain.

What else am I missing from all of these.

Remember to ask clarifying questions, starting with general goals and general idea and stuff like that before asking about specifics as the specifics will be informed by the general stuff! ask a lot of good questions and present good ideas! Let's talk and build something great!
