import { describe, expect, test } from "bun:test"
import type { Plugin } from "@opencode/plugin/tui"
import plugin, { openBridge } from "./tui.ts"

type Option = { title: string; value: string }
type Toast = { variant?: string; message: string }

const projects = [{ id: "p1", name: "alpha", canonical: "/home/dev/src/alpha" }]

function session(id: string, parentID?: string) {
  return {
    id,
    projectID: "p1",
    parentID,
    title: `Title ${id}`,
    location: { directory: "/home/dev/src/alpha" },
    time: { updated: Date.now() },
  }
}

function fakeContext(sessions: ReturnType<typeof session>[], answers: Array<string | undefined>) {
  const selects: Array<{ title: string; options: Option[] }> = []
  const toasts: Toast[] = []
  const layers: Array<{ commands?: Array<Record<string, unknown>> }> = []
  const queue = [...answers]

  const context = {
    client: {
      project: { list: async () => projects },
      session: { list: async () => ({ data: sessions, cursor: { previous: null, next: null } }) },
    },
    ui: {
      dialog: {
        select: async (options: { title: string; options: Option[] }) => {
          selects.push(options)
          return queue.shift()
        },
      },
      toast: { show: (options: Toast) => toasts.push(options) },
    },
    keymap: { layer: (factory: () => { commands?: Array<Record<string, unknown>> }) => layers.push(factory()) },
  }

  return { context: context as unknown as Plugin.Context, selects, toasts, layers }
}

describe("openBridge", () => {
  test("offers only primary sessions and copies the chosen ID", async () => {
    const copied: string[] = []
    const { context, selects, toasts } = fakeContext(
      [session("ses_root"), session("ses_child", "ses_root")],
      ["p1", "ses_root"],
    )

    await openBridge(context, async (text) => {
      copied.push(text)
      return true
    })

    expect(selects[0].options.map((o) => o.value)).toEqual(["p1"])
    expect(selects[1].options.map((o) => o.value)).toEqual(["ses_root"])
    expect(copied).toEqual(["ses_root"])
    expect(toasts[0].variant).toBe("success")
    expect(toasts[0].message).toContain("ses_root")
  })

  test("shows the ID in the toast when no clipboard tool is available", async () => {
    const { context, toasts } = fakeContext([session("ses_root")], ["p1", "ses_root"])

    await openBridge(context, async () => false)

    expect(toasts[0].variant).toBe("warning")
    expect(toasts[0].message).toContain("ses_root")
  })

  test("copies nothing when the project dialog is cancelled", async () => {
    const copied: string[] = []
    const { context, selects, toasts } = fakeContext([session("ses_root")], [undefined])

    await openBridge(context, async (text) => {
      copied.push(text)
      return true
    })

    expect(selects).toHaveLength(1)
    expect(copied).toEqual([])
    expect(toasts).toEqual([])
  })

  test("warns when there are no primary sessions", async () => {
    const { context, selects, toasts } = fakeContext([session("ses_child", "ses_root")], [])

    await openBridge(context, async () => true)

    expect(selects).toHaveLength(0)
    expect(toasts[0].variant).toBe("warning")
  })
})

describe("plugin", () => {
  test("registers a palette command titled Bridge", () => {
    const { context, layers } = fakeContext([], [])

    plugin.setup(context)

    expect(plugin.id).toBe("session-bridge")
    expect(layers).toHaveLength(1)
    const [command] = layers[0].commands ?? []
    expect(command.title).toBe("Bridge")
    expect(command.description).toBeUndefined()
    expect(command.palette).toBe(true)
  })
})
