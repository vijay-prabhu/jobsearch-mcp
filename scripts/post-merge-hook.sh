#!/bin/bash
# Git post-merge hook - rebuilds binary after git pull
# Install: cp scripts/post-merge-hook.sh .git/hooks/post-merge && chmod +x .git/hooks/post-merge

echo "Rebuilding jobsearch binary..."
go build -o .bin/jobsearch ./cmd/jobsearch 2>&1

if [[ $? -eq 0 ]]; then
    echo "✓ jobsearch binary updated"
else
    echo "✗ Build failed" >&2
fi
