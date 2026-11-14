# Tasks

## 1. Server: the list field

- [x] 1.1 Replace `assignee_id` with `assignee_ids: string[]` on tasks (de-duplicated, ordered; `invalid_input` naming the field for the old name); verify `go test ./internal/...` and the v1 surface snapshot
- [x] 1.2 Add the `kIDList` op kind carrying the whole list as one operation (D1); verify convergence tests with a device op on the same field (`go test ./internal/sync`)

## 2. Validation and departures

- [x] 2.1 Validate every element against project membership in `internal/sync/scope.go`, failing the op with `not_a_member` naming the id (D2); verify unit tests for member, non-member, personal-task and viewer cases
- [x] 2.2 Clear a departing member's ids with the Go-side scan and departure operations (D3); verify the departure tests for leave, removal and deactivation
- [x] 2.3 Fan out one `assigned` notification per added assignee, never to the actor (D4); verify notification tests

## 3. Surfaces

- [x] 3.1 Accept `assignee_ids` on `POST /api/v1/tasks` and `PATCH /api/v1/tasks/{id}`; refuse non-members with `400 not_a_member`; verify `go test ./internal/httpapi`
- [x] 3.2 Accept `assignees` (member ids or exact display names) on `create_tasks` and `update_tasks` and on the agent REST routes, through the write switches, limits and `agent.write` audit (D5); verify `go test ./internal/agenttools` and the MCP surface snapshot
- [x] 3.3 Add the route view `assigned` («Мои задачи») selecting the viewer's open assigned tasks in urgency order (D6); verify `go test ./internal/httpapi`
- [x] 3.4 Read the list in the web task panel and rows, and show the «Мои задачи» view; verify `npm test` in `web/`

## 4. Docs and release

- [x] 4.1 Write `docs/agents-ru/AGENTS.md` and `docs/agents-ru/api-assignees.md` with the assignees cookbook, and `docs/agents-ru/opencode.json` as the example config
- [x] 4.2 Add `changelog.d/Changed/multi-assignee.md` and `changelog.d/Added/agent-assignees.md`
- [x] 4.3 Verify the family-pack deliverables on the branch: theme catalogue test (`cd web && npx vitest run src/lib/theme/catalogue.test.ts`), `deploy/family` compose, and the Russian UI/theme fragments (`changelog.d/Added/russian-translation.md`, `changelog.d/Added/family-theme.md`)
