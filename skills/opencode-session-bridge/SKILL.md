---
name: opencode-session-bridge
description: Use when the user wants to send an instruction to another OpenCode session, hand work to the last active session of a project (for example "in my-app"), create a new session in a project directory, implement a feature that spans several directories, list or find OpenCode sessions, or read what a session replied. Uses the opencode-session-bridge MCP tools create_session, list_sessions, find_session, send_message and read_session.
---

# OpenCode Session Bridge

Lets this session start and talk to other OpenCode sessions through five MCP tools:

| Tool | Use it to |
|---|---|
| `create_session` | Start a new session in a project directory, optionally with a first prompt. |
| `list_sessions` | Discover sessions. Filter by `project`, `title`, or `active_only`. Newest first. |
| `find_session` | Get the most recently active session for a project. Running sessions are preferred. |
| `send_message` | Send text to a session. Set `wait_seconds` to get its reply back. |
| `read_session` | Read the latest messages of a session and check whether it is still running. |

The calling session is never returned by discovery and cannot be a `send_message` target.

## Single-project requests

1. **Resolve the target.**
   - The user names a project ("the last active session in my-app"): call `find_session` with `project` set to that name.
   - The user names a topic or title: call `list_sessions` with `project` and/or `title`, then pick the right ID.
   - The user gives a session ID: use it directly.
   - If nothing matches and the user wants a fresh session, call `create_session` with `project`.
   - If nothing matches and the project is unclear, list the candidates and ask which one to use.
2. **Send the message.** Pass the full instruction in `text`. Make it self-contained: the target session does not see this conversation.
   - Set `wait_seconds` when you need the result in this session (for example 600 for a build and push). The tool returns `reply` once the target has finished, or `replied: false` if the wait expired.
   - Leave `wait_seconds` at 0 for fire-and-forget work. Check on it later with `read_session`.
3. **Report back.** Summarise the reply, and name the session ID and directory that handled it.

### Choosing delivery

- `queue` (default): the message runs after the target's current work.
- `steer`: the message interrupts the target's current work. Use it only when the user wants the target to change direction now.

## Multi-directory features

Use this when a feature touches several repositories, for example a service, its client library, and a UI.

1. **Plan the split.** Break the feature into one slice per directory. Write down the contract between slices: endpoint paths, payload shapes, names, and the order they must land in. Show the plan to the user and get approval before creating sessions.
2. **Create one session per directory.** Call `create_session` for each directory with a clear `title` such as `"feature-x: client"`. Do not reuse a directory for two sessions, because they would edit the same files.
3. **Dispatch in parallel.** Send each session its slice with `send_message` and `wait_seconds: 0`, so all of them run at once. Every message should:
   - state the overall goal and this session's slice,
   - include the agreed contract,
   - tell the session to finish with a summary of changes and any interface details the others depend on.
4. **Monitor.** Poll `read_session` for each session. A session is finished when `running` is false and its last assistant message contains the summary. Do not use `wait_seconds` for several sessions in a row, because each wait blocks this session.
5. **Integrate.** When one session reports a change that another depends on, forward that detail to the dependent session with `send_message`. Keep the contract in sync rather than letting sessions guess.
6. **Report.** Summarise per directory: session ID, what changed, and anything left open. Ask before any external step such as a push, deploy, or publish.

### Example: a feature across a service and a client

```
create_session  project: "service-api"     title: "feature-x: service"
create_session  project: "client-sdk"      title: "feature-x: client"
send_message    session_id: <service>  text: "Add GET /v1/widgets/{id}. Contract: ... Reply with a summary when done."  wait_seconds: 0
send_message    session_id: <client>   text: "Add widgets.get(id) for GET /v1/widgets/{id}. Contract: ... Reply with a summary when done."  wait_seconds: 0
read_session    session_id: <service>   (repeat until finished)
send_message    session_id: <client>   text: "Service summary: ... Adjust your method name if needed."  wait_seconds: 0
```

## Rules

- Confirm with the user before sending a message or creating a session that starts external work (pushes, deploys, deletes), unless they already gave a clear instruction for that exact target.
- Never send to the calling session. The tool refuses, so pick another target.
- Use `find_session` only when the project is unambiguous. If several projects could match, list them and ask.
- Treat `replied: false` as unfinished, not failed. Check `read_session` before retrying, so the work is not duplicated.
- One session per directory. Parallel sessions in the same directory will conflict.

## Troubleshooting

- **"no running OpenCode service found"**: OpenCode is not running. Start it, or set `OPENCODE_BASE_URL` in the MCP server's environment.
- **"cannot read .../service.json"** or authentication errors: set `OPENCODE_SERVER_PASSWORD`, or start OpenCode so the service file is written.
- **"no session found for project"**: no recent session matches. Use `create_session` if a new session is wanted, or run `list_sessions` to see the existing directories.
- **"could not resolve project"** from `create_session`: pass an absolute path, or add the parent directory to `SESSION_BRIDGE_PROJECT_ROOTS` in the MCP server's environment.
- **"session ... was created but the first prompt failed"**: the session exists. Send the prompt with `send_message` instead of creating another session.
