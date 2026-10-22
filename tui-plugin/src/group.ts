export type RootSession = {
  id: string
  title?: string
  parentID?: string | null
  projectID: string
  location: { directory: string }
  time: { updated: number }
}

export type ProjectInfo = {
  id: string
  name?: string
  canonical: string
}

export type ProjectGroup = {
  id: string
  name: string
  directory: string
  lastActive: number
  sessions: RootSession[]
}

// Groups primary sessions (no parentID) by project, most recently active first.
// Subagent sessions have a parentID and are skipped.
export function groupRootSessions(sessions: RootSession[], projects: ProjectInfo[]): ProjectGroup[] {
  const projectsByID = new Map(projects.map((project) => [project.id, project]))
  const groups = new Map<string, ProjectGroup>()

  for (const session of sessions) {
    if (session.parentID) continue

    let group = groups.get(session.projectID)
    if (!group) {
      const project = projectsByID.get(session.projectID)
      const directory = project?.canonical ?? session.location.directory
      group = {
        id: session.projectID,
        name: project?.name || lastSegment(directory),
        directory,
        lastActive: 0,
        sessions: [],
      }
      groups.set(session.projectID, group)
    }

    group.sessions.push(session)
    group.lastActive = Math.max(group.lastActive, session.time.updated)
  }

  for (const group of groups.values()) {
    group.sessions.sort((a, b) => b.time.updated - a.time.updated)
  }
  return [...groups.values()].sort((a, b) => b.lastActive - a.lastActive)
}

export function relativeTime(then: number, now: number): string {
  const minutes = Math.max(0, Math.round((now - then) / 60_000))
  if (minutes < 1) return "just now"
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}

function lastSegment(path: string): string {
  return path.split("/").filter(Boolean).pop() ?? path
}
