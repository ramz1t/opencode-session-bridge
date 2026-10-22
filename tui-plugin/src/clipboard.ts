import { spawn } from "node:child_process"

const commands: Array<[string, string[]]> = [
  ["pbcopy", []],
  ["wl-copy", []],
  ["xclip", ["-selection", "clipboard"]],
  ["clip", []],
]

// Tries the first clipboard tool that exists on this platform.
export async function copyToClipboard(text: string): Promise<boolean> {
  for (const [command, args] of commands) {
    if (await writeTo(command, args, text)) return true
  }
  return false
}

function writeTo(command: string, args: string[], input: string): Promise<boolean> {
  return new Promise((resolve) => {
    const child = spawn(command, args, { stdio: ["pipe", "ignore", "ignore"] })
    child.on("error", () => resolve(false))
    child.on("close", (code) => resolve(code === 0))
    child.stdin.end(input)
  })
}
