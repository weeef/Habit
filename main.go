// Command habit runs the Habit prototype. This copy at the repository root
// lets a plain `go build` work, which is what hosts like Render run by
// default; `go run ./cmd/habit` does the same thing.
package main

import "habit/internal/app"

func main() { app.Main() }
