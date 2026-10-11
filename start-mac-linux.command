#!/bin/sh
# Double-click (Mac) or run ./start-mac-linux.command to update Habit from
# GitHub and start it.
cd "$(dirname "$0")" || exit 1
echo "Getting the latest version from GitHub..."
git pull --ff-only || echo "Could not update (offline, or local changes). Starting the version you have."
echo
echo "Starting Habit. Your browser will open http://localhost:8080 in a few seconds."
echo "Close this window or press Ctrl+C to stop it."
(sleep 6; open http://localhost:8080 2>/dev/null || xdg-open http://localhost:8080 2>/dev/null) &
exec go run ./cmd/habit
