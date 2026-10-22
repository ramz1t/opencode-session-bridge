# opencode-session-bridge

An MCP server, written in Go, that lets one OpenCode session send messages to other OpenCode sessions. Ask things like:

> Create a new image version and push it to the registry in the last active session in my-app.

It talks to the local OpenCode service over its HTTP API, so it needs OpenCode to be running.

## Tools

- `create_session`: start a new session in a project directory, optionally with a first prompt
- `list_sessions`: discover sessions, filtered by project, title, or running state
- `find_session`: the most recently active session for a project (running sessions first)
- `send_message`: send a message, optionally waiting for and returning the reply
- `read_session`: read the latest messages of a session

The skill that explains how to use them is in [`skills/opencode-session-bridge/SKILL.md`](skills/opencode-session-bridge/SKILL.md).

## Build

```sh
go build -ldflags "-X main.version=0.1.0" -o bin/opencode-session-bridge .
go test ./...
```

## Register with OpenCode

Add it as a global local MCP server:

```sh
opencode mcp add session-bridge --global -- "$PWD/bin/opencode-session-bridge"
opencode mcp list
```

Or add it by hand to `~/.config/opencode/opencode.json`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "session-bridge": {
        "type": "local",
        "command": ["/absolute/path/to/opencode-session-bridge/bin/opencode-session-bridge"]
      }
    }
  }
}
```

To use the skill, copy `skills/opencode-session-bridge` into `~/.agents/skills/` or `~/.config/opencode/skills/`.

## How it finds OpenCode

1. `OPENCODE_BASE_URL`, if set. Otherwise `opencode service status` is run and the URL it prints is used.
2. `OPENCODE_SERVER_PASSWORD`, if set. Otherwise the `password` in `$XDG_CONFIG_HOME/opencode/service.json` (default `~/.config/opencode/service.json`).
3. `OPENCODE_SERVER_USERNAME`, defaulting to `opencode`.

Discovery runs on every tool call, so the bridge keeps working after OpenCode restarts on a different port.

## Creating sessions

`create_session` resolves its `project` argument in this order:

1. An absolute path that exists is used as is.
2. A name that matches the last directory of a recent session (case-insensitive).
3. A name that exists as a child of one of the directories in `SESSION_BRIDGE_PROJECT_ROOTS`. The variable is a colon-separated list, like `PATH`.

Set it in the server's `environment`, for example:

```jsonc
"session-bridge": {
  "type": "local",
  "command": ["/absolute/path/to/opencode-session-bridge/bin/opencode-session-bridge"],
  "environment": { "SESSION_BRIDGE_PROJECT_ROOTS": "/home/dev/src" }
}
```

## Session targeting

- Project matching checks path segments first, so `my-app` matches `/home/dev/src/my-app`. It then falls back to substring matches.
- Ranking: path-segment matches, then running sessions, then most recently updated.
- When OpenCode passes the calling session ID in MCP `_meta`, that session is excluded from discovery and cannot be the target of `send_message`.
- Discovery looks at the 200 most recent sessions.

## Layout

```
main.go                       MCP server wiring over stdio
internal/opencode/            OpenCode discovery and HTTP client
internal/bridge/              MCP tools and session ranking (with tests)
skills/opencode-session-bridge/SKILL.md
```

## Limits

- The OpenCode session and inbox endpoints are marked experimental in the V2 API, so this may need updating when they change.
- Only text from user and assistant messages is returned. Tool calls and reasoning are not.
- Reply detection looks for an assistant message created after the send, once the session has stopped running. A reply that is still streaming when the wait expires is returned as partial with `replied: false`.
