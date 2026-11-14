package agenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	opsync "github.com/OpenTodoApp/opentodo/internal/sync"
)

// Rotating assignees in the agent tools (OpenSpec change chore-rotation,
// design D6): the task view names the people who take turns, and
// update_tasks sets or clears them with the REST API's checks.

// rotationProp is update_tasks' rotation property.
var rotationProp = map[string]any{"type": []string{"array", "null"}, "maxItems": opsync.MaxRotation, "items": map[string]any{"type": "string"},
	"description": "Account ids of the members who take turns on a recurring task of a shared project, in order (see the task's assignees and rotation); completing the task assigns the next of them. An empty array or null stops taking turns."}

// assigneesProp is create_tasks' and update_tasks' assignees property
// (task-assignment): member ids or exact member names.
var assigneesProp = map[string]any{"type": []string{"array", "null"}, "maxItems": opsync.MaxBlockedBy, "items": map[string]any{"type": "string"},
	"description": "Who the task is assigned to, as account ids or exact member display names (see list_projects for a project's members). An empty array or null clears the assignment."}

// assigneePatch checks a write's assignees and puts the resolved assignee_ids
// into the patch (task-assignment D2): at most MaxBlockedBy distinct members of
// the task's project — an element is an account id or an exact member display
// name; a task without a shared project may only be assigned to the caller.
// Unknown names are a validation error, never a silent drop.
func (snap *snapshot) assigneePatch(p string, v json.RawMessage, pid, caller string, patch map[string]json.RawMessage) error {
	var elements []string
	if string(v) != "null" && json.Unmarshal(v, &elements) != nil {
		return argErr(p + "assignees: must be an array of member ids or names")
	}
	if len(elements) > opsync.MaxBlockedBy {
		return argErr(fmt.Sprintf("%sassignees: at most %d assignees", p, opsync.MaxBlockedBy))
	}
	members := map[string]string{} // member id → display name
	for id := range snap.members[pid] {
		members[id] = snap.people[id]
	}
	if len(members) == 0 {
		// A personal task: only the caller may hold it (sync canBeAssigned).
		members[caller] = snap.people[caller]
	}
	ids := make([]string, 0, len(elements))
	seen := map[string]bool{}
	for _, el := range elements {
		var id string
		if _, isID := members[el]; isID {
			id = el
		} else {
			// Not an id of a member: an exact member display name, then.
			for mid, name := range members {
				if name != "" && strings.EqualFold(name, el) {
					if id != "" {
						return argErr(fmt.Sprintf("%sassignees: several members are named %q; use their account id", p, el))
					}
					id = mid
				}
			}
		}
		if id == "" {
			return argErr(fmt.Sprintf("%sassignees: %q is not a member of the task's project or their exact name", p, el))
		}
		if seen[id] {
			return argErr(fmt.Sprintf("%sassignees: lists %q twice", p, el))
		}
		seen[id] = true
		ids = append(ids, id)
	}
	patch["assignee_ids"] = raw(ids)
	return nil
}

// rotationPatch checks an update's rotation against the task after the
// patch: at most MaxRotation distinct ids, and a non-empty list only on a
// top-level recurring task of a project with at least two members, every id
// one of them.
func (snap *snapshot) rotationPatch(p string, v json.RawMessage, t opsync.Entity, patch map[string]json.RawMessage) error {
	var list []string
	if string(v) != "null" && json.Unmarshal(v, &list) != nil {
		return argErr(p + "rotation: must be an array of account ids")
	}
	if list == nil {
		list = []string{}
	}
	if !opsync.ValidRotation(list) {
		return argErr(fmt.Sprintf("%srotation: at most %d different account ids", p, opsync.MaxRotation))
	}
	if len(list) > 0 {
		recur := t["recurrence"]
		if r, ok := patch["recurrence"]; ok {
			recur = r
		}
		pid := rawStr(t, "project_id")
		members := snap.members[pid]
		if opsync.StringOf(recur) == "" || rawStr(t, "parent_id") != "" || len(members) < 2 {
			return argErr(p + "rotation: only a top-level recurring task in a project shared with at least one other member takes turns")
		}
		for _, id := range list {
			if !members[id] {
				return argErr(fmt.Sprintf("%srotation: %s is not a member of the task's project", p, id))
			}
		}
	}
	patch["rotation"] = raw(list)
	return nil
}

// rotationView fills a task view's rotation: the ids in order and, as
// untrusted text, the people's names.
func (snap *snapshot) rotationView(t opsync.Entity, out *taskOut) {
	ids := stringList(t, "rotation")
	if len(ids) == 0 || rawStr(t, "recurrence") == "" {
		return
	}
	out.Rotation = ids
	for _, id := range ids {
		name, ok := snap.people[id]
		if !ok {
			name = "(left the project)"
		}
		out.Untrusted.Rotation = append(out.Untrusted.Rotation, name)
	}
}
