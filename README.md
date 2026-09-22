# Wajb

A tiny CLI + terminal UI for tracking tasks 'issus' as plain Markdown files, stored right alongside your code (in a `.tasks/` folder at the root of your git repo),
No database, no server — each task is just a folder with a `TASK.md` file you can read, grep, or edit by hand, and get traked by git.

## build
## Requirements
- Go 1.21+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) (fetched automatically via `go mod tidy`)

```bash
go build
```

Move the resulting `wajb` binary somewhere on your `$PATH` if you want to run it from anywhere.

## Usage

```bash
wajb new       # or: task n   — create a new task interactively
wajb list      # or: task l   — print all tasks, one line each
wajb           # no args      — open the interactive TUI
```

### Creating a task

```
$ wajb new
title: Fix login bug
tags (split by space): backend auth
description: Users get logged out after 5 minutes.
Investigate session TTL config.

priority (default: 50): 80
```

Leave a blank line to finish the description. Priority defaults to `50` if left empty.

### Listing tasks

```
$ wajb list
1758549213-a1b2c3d4    [OPEN]   80   Fix login bug                 backend,auth
1758549100-9f8e7d6c    [CLOSE]  50   Update docs                    docs
```

Sorted by priority, highest first.

## The TUI

Running `task` with no arguments opens an interactive list:

| Key         | Action                                  |
|-------------|------------------------------------------|
| `↑` / `k`   | Move up                                  |
| `↓` / `j`   | Move down                                |
| `enter`     | Open the selected task's full details    |
| `space`     | Toggle OPEN/CLOSE status                 |
| `/`         | Filter by title or description           |
| `@`         | Filter by tag                            |
| `e`         | Edit the task's `TASK.md` in `$EDITOR`   |
| `esc`       | Clear the active filter / go back        |
| `q`         | Quit                                     |

`$EDITOR` is used if set, otherwise it falls back to `nano`.

## Storage

Tasks live under `.tasks/` at the root of the current git repository (or a local `.task` folder if you're not in a git repo). Each task is its own directory named by a generated ID, containing a single `TASK.md`:

```
.tasks/
  1758549213-a1b2c3d4/
    TASK.md
```

```markdown
# Fix login bug

- STATUS: OPEN
- PRIORITY: 80
- TAGS: backend ,auth

~ Users get logged out after 5 minutes.
Investigate session TTL config.
```

Because it's just Markdown, tasks are easy to diff, commit, search with `grep`/`ripgrep`, or edit outside the tool entirely.

