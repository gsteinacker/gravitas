#!/bin/sh
# PostToolUse hook: gofmt every Go file Claude edits or writes.
f=$(jq -r '.tool_input.file_path // empty')
case "$f" in
  *.go) [ -f "$f" ] && gofmt -w "$f" ;;
esac
exit 0
