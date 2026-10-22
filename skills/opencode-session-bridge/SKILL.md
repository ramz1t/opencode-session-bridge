---
name: opencode-session-bridge
description: Use when the user wants to send an instruction to another OpenCode session, hand work to the last active session of a project (for example "in my-app"), list or find OpenCode sessions, or read what a session replied. Uses the opencode-session-bridge MCP tools list_sessions, find_session, send_message and read_session.
---

# OpenCode Session Bridge

Lets this session talk to other OpenCode sessions through four MCP tools:

| Tool | Use it to |
|---|---|
| `list_sessions` | Discover sessions. Filter by `project`, `title`, or `active_only`. Newest first. |
| `find_session` | Get the most recently active session for a project. Running sessions are preferred. |
| `send_message` | Send text to a session. Set `wait_seconds` to get its reply back. |
| `read_session` | Read the latest messages of a session and check whether it is still running. |

The calling session is never returned by discovery and cannot be a `send_message` target.

## Workflow

1. **Resolve the target.**
   - The user names a project ("the last active session in my-app"): call `find_session` with `project` set to that name.
   - The user names a topic or title: call `list_sessions` with `project` and/or `title`, then pick the right ID.
   - The user gives a session ID: use it directly.
   - If nothing matches, tell the user which sessions exist (`list_sessions` with no filter) and ask which one to use.
2. **Send the message.** Pass the full instruction in `text`. Make it self-contained: the target session does not see this conversation.
   - Set `wait_seconds` when you need the result in this session (for example 600 for a build and push). The tool returns `reply` once the target has finished, or `replied: false` with a partial state if the wait expired.
   - Leave `wait_seconds` at 0 for fire-and-forget work. Then check on it later with `read_session`.
3. **Report back.** Summarise the reply for the user and say which session handled it, including its ID and directory.

### Choosing delivery

- `queue` (default for normal work): the message runs after the target's current work.
- `steer`: the message is injected into the target's current work. Use it only when the user wants the target to change direction now.

## Examples

- "Create a new image version and push it to the registry in the last active session in my-app":
  1. `find_session` with `project: "my-app"`.
  2. `send_message` with that `session_id`, `text: "Create a new image version and push it to the registry. Report the image tag and digest."`, `wait_seconds: 900`.
  3. Report the reply.
- "What is ses_abc doing?": `read_session` with `session_id: "ses_abc"`.
- "Which sessions are running right now?": `list_sessions` with `active_only: true`.

## Rules

- Confirm with the user before sending a message that starts external work (pushes, deploys, deletes), unless they already gave a clear instruction for that exact target.
- Never send to the calling session. The tool refuses, so pick another target.
- Use `find_session` only when the project is unambiguous. If several projects could match, list them and ask.
- Treat `replied: false` as unfinished, not failed. Check `read_session` before retrying, so the work is not duplicated.

## Troubleshooting

- **"no running OpenCode service found"**: OpenCode is not running. Start it, or set `OPENCODE_BASE_URL` in the MCP server's environment.
- **"cannot read .../service.json"** or authentication errors: set `OPENCODE_SERVER_PASSWORD`, or start OpenCode so the service file is written.
- **"no session found for project"**: the name did not match any recent session directory. Run `list_sessions` without a filter to see the directories that exist.
