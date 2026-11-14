# Design

## Context

The single `assignee_id` (OpenSpec `task-assignment`) is an ordinary task field written through the same op log as every other field, with membership checked at write time (`400 not_a_member`) and cleared server-side when a member leaves. The sync endpoint validates each operation against the project's sharing scope (`internal/sync/scope.go`) before anything is applied, and notifications are plain server-side account data written after the op log batch commits. The family scenario (four members plus an assistant, `deploy/family`) is the driving use case: shared chores addressed to two children, re-addressing by the assistant, and a «Мои задачи» list per person.

## Goals / Non-Goals

**Goals:**
- One list field, applied atomically: the whole list is one operation, so a device's edit of the same field converges by last-writer-wins like any other.
- Refusals name the element that failed, so a client with several ids can tell which one to drop.
- Every assignee learns about the assignment; nobody gets a notification about themself.
- Assistant writes read exactly like human writes: same switches, limits, audit, `untrusted_text` shapes.

**Non-Goals:**
- Per-assignee completion or per-person due dates.
- More than the existing membership rule (no assignment of non-members on any surface).
- Keeping the old `assignee_id` as an alias: one name, one shape, a clean break on this branch.

## Decisions

### D1. The list travels as one op of a `kIDList` kind

The op log gains a `kIDList` op kind that carries the whole assignee list as its value. A write that sends three ids is one operation, not three per-element writes: the batch stays atomic (completely or not at all), an offline device that edited the list concurrently converges by last-writer-wins on the whole list, and the history entry reads "changed the assignees" once. Per-element ops would make two devices that each reordered the list merge into a mixed order no one chose.

- **Alternative: a `kID` op per element with add/remove deltas.** Rejected: it makes "replace the list" a multi-op batch whose parts can interleave with another device's parts, and it needs a tombstone rule for removals. The list is short (project members, at most a few dozen) and is always edited as a whole in the UI.

### D2. Scope validation checks every element (`scope.go`)

`internal/sync/scope.go` validates a `kIDList` op by resolving each id against the task's project membership, exactly once per op: duplicates are folded, an id that is not a current member (or, for a personal task, not the writer) fails the operation naming that id (`not_a_member`), and the rest of the batch is untouched. Validation happens in the same pass that already loads project membership, so the check adds no second read of the members table.

- **Alternative: validate after apply, on the stored row.** Rejected: the sync endpoint must reject per operation (`rejected: [{id, reason}]`) before applying anything, which is what scope validation is for.

### D3. Departures are a Go-side scan, not a SQL `json_each`

When a member leaves, is removed or is deactivated, the server clears their id from the assignee lists of the project in Go: one query selects the project's tasks, each stored list is decoded, filtered and written back with an operation of its own (device `server:<boot id>`), reusing the existing departure path of the single assignee. The scan runs inside the same transaction as the membership change.

- **Alternative: a SQLite `json_each` statement updating rows in place.** Rejected: it bypasses the op log (devices would not learn of the change), it duplicates the list shape in SQL, and it cannot write the departure operations the audit and history expect.

### D4. Notifications fan out per assignee

Applying an assignee write computes the diff against the previous list: every id added since gets one `assigned` notification (the same payload as today's single-assignee notification), the actor never gets one about themself, and nobody removed gets anything. In a two-children chore that is two notifications, each opening the same task. The notification routes, `turn: true` payload of rotations and the 90-day retention are unchanged.

### D5. Agents get `assignees` with name resolution

`create_tasks` and `update_tasks` accept `assignees`, whose elements are member ids or exact display names resolved against the project's members inside the same tool call (`-32602` when a name matches nobody or matches ambiguously, `not_a_member` for a resolved non-member). The agent REST routes accept the same field. The write is one batch through the tool registry, so it shares the counters (one REST write is one write call), the `agent.write` audit, the untrusted-text marking and the kept-away-project invisibility of every tool.

### D6. The «Мои задачи» view

A route view `assigned` selects open tasks whose `assignee_ids` contain the viewer, across their personal and shared projects, ordered by the same urgency order as Today. It is a plain view on stored data — no per-user index — because a family's task count is small and the same filter already exists for the MCP `list_tasks` shapes.

## Risks / Trade-offs

- **BREAKING rename.** Clients sending `assignee_id` fail with `400 invalid_input` naming the field; the docs and the changelog say so, and the error message points at `assignee_ids`.
- **Name resolution for agents** depends on display names being stable and exact; a rename between read and write fails loudly rather than assigning the wrong person, which is the safe direction.
- **Fan-out cost** is bounded by project membership; a write assigning every member of a large project creates that many notifications, which is what the writer asked for.
