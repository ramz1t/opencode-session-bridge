#!/bin/sh
# Copies the plugin into OpenCode's global plugin directory, where it is discovered automatically.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/opencode"
target="$config_dir/plugins/session-bridge"

for file in "$here/tui.ts" "$here/package.json" "$here"/src/*.ts; do
  case "$file" in
    *.test.ts) continue ;;
  esac
  relative=${file#"$here"/}
  mkdir -p "$target/$(dirname "$relative")"
  cp "$file" "$target/$relative"
done

echo "Installed session-bridge plugin to $target"
echo "Restart OpenCode, then press Ctrl+P and choose \"Bridge\"."
