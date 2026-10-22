import { Plugin } from "@opencode/plugin/tui"
import { copyToClipboard } from "./src/clipboard.ts"
import { groupRootSessions, relativeTime, type ProjectGroup } from "./src/group.ts"

type Copy = (text: string) => Promise<boolean>

async function loadGroups(context: Plugin.Context): Promise<ProjectGroup[]> {
  const [projects, sessions] = await Promise.all([
    context.client.project.list(),
    context.client.session.list({ limit: 200, order: "desc", parentID: null }),
  ])
  return groupRootSessions(sessions.data, projects)
}

export async function openBridge(context: Plugin.Context, copy: Copy = copyToClipboard) {
  const groups = await loadGroups(context)
  if (groups.length === 0) {
    context.ui.toast.show({ variant: "warning", title: "Bridge", message: "No sessions found." })
    return
  }

  const now = Date.now()
  const projectID = await context.ui.dialog.select<string>({
    title: "Bridge: choose a project",
    placeholder: "Filter projects",
    options: groups.map((group) => ({
      title: group.name,
      value: group.id,
      description: group.directory,
      footer: `${group.sessions.length} sessions · ${relativeTime(group.lastActive, now)}`,
    })),
  })
  if (!projectID) return

  const group = groups.find((candidate) => candidate.id === projectID)
  if (!group) return

  const sessionID = await context.ui.dialog.select<string>({
    title: `Bridge: ${group.name}`,
    placeholder: "Filter sessions",
    options: group.sessions.map((session) => ({
      title: session.title || "Untitled session",
      value: session.id,
      description: session.id,
      footer: relativeTime(session.time.updated, now),
    })),
  })
  if (!sessionID) return

  const copied = await copy(sessionID)
  context.ui.toast.show({
    variant: copied ? "success" : "warning",
    title: "Bridge",
    message: copied
      ? `Copied ${sessionID}. Paste it into your prompt.`
      : `No clipboard tool found. Session ID: ${sessionID}`,
    duration: 8000,
  })
}

export default Plugin.define({
  id: "session-bridge",
  setup(context) {
    context.keymap.layer(() => ({
      mode: "global",
      commands: [
        {
          id: "session-bridge.open",
          title: "Bridge",
          description: "Pick a project and a session, then copy its ID",
          group: "Bridge",
          palette: true,
          bind: false,
          run: async () => {
            try {
              await openBridge(context)
            } catch (error) {
              context.ui.toast.show({
                variant: "error",
                title: "Bridge",
                message: error instanceof Error ? error.message : String(error),
              })
            }
          },
        },
      ],
    }))
  },
})
