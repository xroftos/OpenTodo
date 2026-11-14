# Proposal

## Why

A family shares its tasks, and "who does it" is rarely one person: the four members of the family scenario (two adults, two children, plus an AI assistant that keeps the schedule) need one chore marked for two children, a shopping list addressed to both adults, a hand-over where the assistant re-addresses a task to whoever can actually do it. Today a task carries a single `assignee_id` (OpenSpec `task-assignment`), so a shared task is either duplicated once per person — and then completed, notified and tracked separately — or it sits unassigned and nobody owns it. The single field also forces the family's AI assistant to pick one member when the request plainly names two.

## What Changes

- **`assignee_ids: string[]` replaces `assignee_id` (BREAKING).** A task carries an ordered, de-duplicated list of the project members assigned to it. The old field is no longer accepted by `POST /api/v1/tasks`, `PATCH /api/v1/tasks/{id}`, the sync endpoint or the export; clients that sent `assignee_id` must send the list instead. A single-member list reads in the app exactly as the old single assignee did.
- **Validation per element.** Every id in the list must be a current member of the task's shared project (or the writer themself, for a personal task); a non-member refuses the write with `400 not_a_member`, naming the offending id. Viewers still cannot assign (`403 forbidden`).
- **Departures clear assignments.** When a member leaves, is removed or is deactivated, the server clears their ids from every task of that project with operations of its own, so every device converges on the tasks minus that person.
- **Notifications fan out.** Assigning several people creates one in-app notification per new assignee (the actor is not notified about themself), the same `assigned` payload as today, one per person.
- **Agent writes.** The MCP tools `create_tasks` and `update_tasks` accept `assignees` — member ids **or exact display names** — and the agent REST routes (`POST /api/v1/tasks`, `PATCH /api/v1/tasks/{id}` with an `agent` token) accept the same field, under the existing write switches, limits, `agent.write` audit and kept-away-project rules. A name that matches nobody is refused; a non-member stays `not_a_member`.
- **A «Мои задачи» list.** A unified route view `assigned` gathers every open task assigned to the viewer, across personal and shared projects, as one list the sidebar and the phone can show.

Not **BREAKING** for reading clients: the list is additive in every read shape for readers that ignore unknown fields; it is breaking only for writers of the old field name.

Out of scope: per-assignee completion states (the task is done for everyone at once), due-date splitting per person, rotation beyond the existing `rotation` field, and assignment of non-members through any surface.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tasks`: `assignee_ids` replaces `assignee_id`; per-element member validation; departure clearing; fan-out notifications.
- `sync`: the assignee list travels as one `kIDList` op and validates in scope.
- `mcp-server`: `assignees` (ids or exact names) on `create_tasks` and `update_tasks`.

## Impact

- Server: `internal/sync` (the `kIDList` op kind, scope validation per element, the departure scan), `internal/httpapi` (tasks routes, the `assigned` view), `internal/agenttools` (`assignees` on create and update, name resolution against project members), notifications fan-out, the v1 surface snapshot.
- Web: the task panel and row assignee display read the list; the «Мои задачи» view (route view `assigned`); sharing and member dialogs unchanged.
- Docs: `docs/agents-ru/AGENTS.md`, `docs/agents-ru/api-assignees.md` (new, Russian), `docs/api.md` (field reference; may trail this change while it is being edited).
- Changelog: `changelog.d/Changed/multi-assignee.md`, `changelog.d/Added/agent-assignees.md`, `changelog.d/Added/russian-translation.md`, `changelog.d/Added/family-theme.md` (the family-pack branch ships the theme and the translation on the same train).
