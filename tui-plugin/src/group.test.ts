import { describe, expect, test } from "bun:test"
import { groupRootSessions, relativeTime, type RootSession } from "./group.ts"

const MINUTE = 60_000

function session(overrides: Partial<RootSession> & { id: string; projectID: string }): RootSession {
  return {
    title: `Session ${overrides.id}`,
    location: { directory: "/home/dev/src/app" },
    time: { updated: 0 },
    ...overrides,
  }
}

describe("groupRootSessions", () => {
  test("skips subagent sessions", () => {
    const groups = groupRootSessions(
      [
        session({ id: "ses_root", projectID: "p1", time: { updated: 10 } }),
        session({ id: "ses_child", projectID: "p1", parentID: "ses_root", time: { updated: 99 } }),
      ],
      [],
    )

    expect(groups).toHaveLength(1)
    expect(groups[0].sessions.map((s) => s.id)).toEqual(["ses_root"])
  })

  test("groups by project and sorts projects and sessions by last activity", () => {
    const groups = groupRootSessions(
      [
        session({ id: "a_old", projectID: "p1", time: { updated: 100 } }),
        session({ id: "b_new", projectID: "p2", time: { updated: 300 } }),
        session({ id: "a_new", projectID: "p1", time: { updated: 200 } }),
      ],
      [
        { id: "p1", name: "alpha", canonical: "/home/dev/src/alpha" },
        { id: "p2", name: "beta", canonical: "/home/dev/src/beta" },
      ],
    )

    expect(groups.map((g) => g.name)).toEqual(["beta", "alpha"])
    expect(groups[1].sessions.map((s) => s.id)).toEqual(["a_new", "a_old"])
    expect(groups[1].lastActive).toBe(200)
  })

  test("falls back to the session directory when the project is unknown", () => {
    const [group] = groupRootSessions(
      [session({ id: "ses_1", projectID: "unknown", location: { directory: "/work/my-service" } })],
      [],
    )

    expect(group.name).toBe("my-service")
    expect(group.directory).toBe("/work/my-service")
  })

  test("returns nothing when there are only subagent sessions", () => {
    expect(groupRootSessions([session({ id: "ses_child", projectID: "p1", parentID: "ses_root" })], [])).toEqual([])
  })
})

describe("relativeTime", () => {
  const now = Date.UTC(2026, 9, 22, 12, 0, 0)

  test("formats recent times", () => {
    expect(relativeTime(now - 10_000, now)).toBe("just now")
    expect(relativeTime(now - 5 * MINUTE, now)).toBe("5m ago")
    expect(relativeTime(now - 3 * 60 * MINUTE, now)).toBe("3h ago")
    expect(relativeTime(now - 2 * 24 * 60 * MINUTE, now)).toBe("2d ago")
  })

  test("never reports a negative age", () => {
    expect(relativeTime(now + MINUTE, now)).toBe("just now")
  })
})
