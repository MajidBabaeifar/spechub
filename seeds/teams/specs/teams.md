# Teams

**Status:** Draft · **Prefix:** TEAM

## Purpose

Users group into teams with roles; apps attach their own resources and permissions to a team.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| TEAM-1 | Team: `id`, `name` (1–100), `slug` (`[a-z0-9-]`, 2–64, unique across teams and usernames), `logo?`, `bio?`, `private` (default false: member list visible on the public page). | slug equal to a username → refused |
| TEAM-2 | Membership `(user, team)` unique: `role`: `member` < `admin` < `owner`, `accepted` (bool). Unaccepted memberships grant nothing. | pending member reads team → 403 |
| TEAM-3 | Any verified user can create a team and becomes its `owner`. | create → creator is owner |
| TEAM-4 | Admins invite by email with a role ≤ their own; invite = single-use token valid 7 days; re-inviting replaces the token. | admin invites owner → refused |
| TEAM-5 | Accepting: a signed-in user with the invited email accepts; a new person signs up via the link (email counts as verified) and joins on signup. Wrong email → refused. | link used by other email → refused |
| TEAM-6 | Permissions: read team and members: member+; update team, invite, remove members (below own role): admin+; change roles, transfer ownership, delete team: owner. | member removes member → 403 |
| TEAM-7 | A team always has ≥ 1 owner: the last owner can't leave, be removed or be demoted. | sole owner leaves → refused |
| TEAM-8 | Members can leave; removal or leaving ends access immediately. | removed member's session → 403 on team |
| TEAM-9 | Deleting a team deletes its resources; deleting a user removes their memberships (refused while they're a sole owner). | sole owner deletes account → refused |
| TEAM-10 | Apps extend TEAM-6 with their own resource rows (resource, action, minimum role); the default for unlisted actions is admin+. | manual: requiring spec lists its rows |

## Out of scope

- Organizations with sub-teams and domain auto-join, custom roles, SCIM.

## Decisions

- Mostly designed here: the source (Cal.diy, enterprise code removed) kept only roles, membership and invite redemption.
- Slugs share one namespace with usernames so `/{slug}` URLs never clash.
