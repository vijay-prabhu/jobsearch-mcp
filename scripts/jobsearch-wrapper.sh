#!/bin/bash
# Smart wrapper that auto-rebuilds if source is newer than binary

SKILL_DIR="${JOBSEARCH_DIR:-$HOME/.claude/skills/jobsearch}"
BINARY="$SKILL_DIR/.bin/jobsearch"
GO_MOD="$SKILL_DIR/go.mod"

# Check if binary exists and is up-to-date
needs_rebuild() {
    # No binary exists
    [[ ! -f "$BINARY" ]] && return 0

    # Check if any .go file is newer than binary
    if find "$SKILL_DIR" -name "*.go" -newer "$BINARY" 2>/dev/null | grep -q .; then
        return 0
    fi

    return 1
}

# Rebuild if needed (silently, only show errors)
if needs_rebuild; then
    mkdir -p "$(dirname "$BINARY")"
    (cd "$SKILL_DIR" && go build -o "$BINARY" ./cmd/jobsearch) 2>&1 | grep -v "^$" >&2
fi

# Execute
exec "$BINARY" "$@"
