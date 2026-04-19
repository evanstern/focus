#!/usr/bin/env bats

setup() {
  export FOCUS_HOME="$BATS_TEST_TMPDIR/.focus"
  export FOCUS_KANBAN_DIR="$FOCUS_HOME/kanban"
  export FOCUS_INTENT_DIR="$FOCUS_HOME/intents"
  export NO_COLOR=1

  mkdir -p "$FOCUS_KANBAN_DIR"

  FIXTURES="$BATS_TEST_DIRNAME/fixtures"
  FOCUS="$BATS_TEST_DIRNAME/../bin/focus"
}

load_fixture() {
  cp "$FIXTURES/$1" "$FOCUS_KANBAN_DIR/"
}

@test "version prints version string" {
  run "$FOCUS" version
  [ "$status" -eq 0 ]
  [[ "$output" =~ ^focus\ [0-9]+\.[0-9]+\.[0-9]+$ ]]
}

@test "help exits successfully" {
  run "$FOCUS" help
  [ "$status" -eq 0 ]
  [[ "$output" =~ "kanban card manager" ]]
}

@test "board shows empty board" {
  run "$FOCUS" board
  [ "$status" -eq 0 ]
  [[ "$output" =~ "WIP: 0/3" ]]
}

@test "new creates a backlog card" {
  run "$FOCUS" new "Test card" "my-project"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Created: #1" ]]
  [ -f "$FOCUS_KANBAN_DIR/test-card.md" ]

  run grep 'status: backlog' "$FOCUS_KANBAN_DIR/test-card.md"
  [ "$status" -eq 0 ]

  run grep 'project: my-project' "$FOCUS_KANBAN_DIR/test-card.md"
  [ "$status" -eq 0 ]
}

@test "new without title fails" {
  run "$FOCUS" new
  [ "$status" -eq 1 ]
  [[ "$output" =~ "usage:" ]]
}

@test "new rejects duplicate slugs" {
  run "$FOCUS" new "Test card"
  [ "$status" -eq 0 ]
  run "$FOCUS" new "Test card"
  [ "$status" -eq 1 ]
  [[ "$output" =~ "already exists" ]]
}

@test "show displays card details" {
  load_fixture sample-backlog.md
  run "$FOCUS" show 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Sample backlog task" ]]
  [[ "$output" =~ "test-project" ]]
  [[ "$output" =~ "backlog" ]]
}

@test "show by slug works" {
  load_fixture sample-backlog.md
  run "$FOCUS" show sample-backlog
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Sample backlog task" ]]
}

@test "show with invalid ref fails" {
  run "$FOCUS" show 999
  [ "$status" -eq 1 ]
  [[ "$output" =~ "not found" ]]
}

@test "activate moves card to active" {
  load_fixture sample-backlog.md
  run "$FOCUS" activate 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Activated: 1" ]]

  run grep 'status: active' "$FOCUS_KANBAN_DIR/sample-backlog.md"
  [ "$status" -eq 0 ]
}

@test "activate respects WIP limit" {
  export FOCUS_WIP_LIMIT=1
  load_fixture sample-active.md
  load_fixture sample-backlog.md
  run "$FOCUS" activate 1
  [ "$status" -eq 1 ]
  [[ "$output" =~ "WIP limit reached" ]]
}

@test "activate --force bypasses WIP limit" {
  export FOCUS_WIP_LIMIT=1
  load_fixture sample-active.md
  load_fixture sample-backlog.md
  run "$FOCUS" --force activate 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Activated" ]]
}

@test "park moves card to parked" {
  load_fixture sample-active.md
  run "$FOCUS" park 2
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Parked: 2" ]]

  run grep 'status: parked' "$FOCUS_KANBAN_DIR/sample-active.md"
  [ "$status" -eq 0 ]
}

@test "kill moves card to killed" {
  load_fixture sample-backlog.md
  run "$FOCUS" kill 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Killed: 1" ]]

  run grep 'status: killed' "$FOCUS_KANBAN_DIR/sample-backlog.md"
  [ "$status" -eq 0 ]
}

@test "done --force skips contract check" {
  load_fixture sample-with-contract.md
  run "$FOCUS" --force done 3
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Done: 3" ]]

  run grep 'status: done' "$FOCUS_KANBAN_DIR/sample-with-contract.md"
  [ "$status" -eq 0 ]
}

@test "list shows all cards" {
  load_fixture sample-backlog.md
  load_fixture sample-active.md
  run "$FOCUS" list
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Sample backlog task" ]]
  [[ "$output" =~ "Sample active task" ]]
}

@test "list filters by status" {
  load_fixture sample-backlog.md
  load_fixture sample-active.md
  run "$FOCUS" list backlog
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Sample backlog task" ]]
  [[ ! "$output" =~ "Sample active task" ]]
}

@test "wip shows active count" {
  load_fixture sample-active.md
  run "$FOCUS" wip
  [ "$status" -eq 0 ]
  [[ "$output" =~ "WIP: 1/3" ]]
}

@test "intent sets and reads intent" {
  unset TMUX
  run "$FOCUS" intent "work on focus"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Intent set" ]]

  run "$FOCUS" intent
  [ "$status" -eq 0 ]
  [[ "$output" =~ "work on focus" ]]
}

@test "new auto-increments IDs" {
  load_fixture sample-active.md
  run "$FOCUS" new "Third task"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "#3" ]]
}

@test "--quiet suppresses output" {
  run "$FOCUS" --quiet new "Silent card"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ -f "$FOCUS_KANBAN_DIR/silent-card.md" ]
}

@test "--project filters board output" {
  load_fixture sample-backlog.md
  "$FOCUS" new "Other project task" "other-project"
  run "$FOCUS" --project test-project list
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Sample backlog task" ]]
  [[ ! "$output" =~ "Other project task" ]]
}

@test "unknown command fails" {
  run "$FOCUS" nonsense
  [ "$status" -eq 1 ]
  [[ "$output" =~ "Unknown command" ]]
}

@test "init creates new kanban board" {
  local init_dir="$BATS_TEST_TMPDIR/new-board"
  run "$FOCUS" init "$init_dir"
  [ "$status" -eq 0 ]
  [ -d "$init_dir" ]
  [ -f "$init_dir/getting-started.md" ]
  [[ "$output" =~ "Initialized kanban board" ]]
}

@test "init writes yaml config" {
  local init_dir="$BATS_TEST_TMPDIR/yaml-board"
  run "$FOCUS" init "$init_dir"
  [ "$status" -eq 0 ]
  [ -f "$FOCUS_HOME/config" ]
  run grep "^kanban_dir: $init_dir" "$FOCUS_HOME/config"
  [ "$status" -eq 0 ]
}

@test "setup writes yaml config" {
  local dir="$BATS_TEST_TMPDIR/existing-board"
  mkdir -p "$dir"
  run "$FOCUS" setup "$dir"
  [ "$status" -eq 0 ]
  [ -f "$FOCUS_HOME/config" ]
  run grep "^kanban_dir: $dir" "$FOCUS_HOME/config"
  [ "$status" -eq 0 ]
}

@test "setup without args fails" {
  run "$FOCUS" setup
  [ "$status" -eq 1 ]
  [[ "$output" =~ "usage:" ]]
}

@test "config kanban_dir is used when env not set" {
  unset FOCUS_KANBAN_DIR
  local custom_dir="$BATS_TEST_TMPDIR/custom-kanban"
  mkdir -p "$custom_dir"
  mkdir -p "$FOCUS_HOME"
  printf 'kanban_dir: %s\n' "$custom_dir" > "$FOCUS_HOME/config"

  run "$FOCUS" board
  [ "$status" -eq 0 ]
  [[ "$output" =~ "WIP: 0/" ]]
}

@test "config wip_limit is respected" {
  unset FOCUS_WIP_LIMIT
  mkdir -p "$FOCUS_HOME"
  printf 'wip_limit: 5\nkanban_dir: %s\n' "$FOCUS_KANBAN_DIR" > "$FOCUS_HOME/config"
  load_fixture sample-active.md

  run "$FOCUS" wip
  [ "$status" -eq 0 ]
  [[ "$output" =~ "WIP: 1/5" ]]
}

@test "completions bash prints completion script" {
  run "$FOCUS" completions bash
  [ "$status" -eq 0 ]
  [[ "$output" =~ "complete -F _focus focus" ]]
}

@test "completions with bad shell fails" {
  run "$FOCUS" completions fish
  [ "$status" -eq 1 ]
  [[ "$output" =~ "unsupported shell" ]]
}

# ── TUI tests ────────────────────────────────────────────────

@test "tui without terminal errors gracefully" {
  run "$FOCUS" tui < /dev/null
  [ "$status" -eq 1 ]
  [[ "$output" =~ "interactive terminal" ]]
}

@test "tui is a recognized command" {
  # Piped stdin makes it fail, but NOT as 'Unknown command'
  run "$FOCUS" tui < /dev/null
  [[ ! "$output" =~ "Unknown command" ]]
}

@test "help includes tui command" {
  run "$FOCUS" help
  [ "$status" -eq 0 ]
  [[ "$output" =~ "tui" ]]
}

@test "bash completions include tui" {
  run "$FOCUS" completions bash
  [ "$status" -eq 0 ]
  [[ "$output" =~ "tui" ]]
}

@test "zsh completions include tui" {
  run "$FOCUS" completions zsh
  [ "$status" -eq 0 ]
  [[ "$output" =~ "tui" ]]
}

@test "edit in non-TTY prints card path and exits 0" {
  load_fixture sample-backlog.md
  local updated_before
  updated_before="$(grep '^updated:' "$FOCUS_KANBAN_DIR/sample-backlog.md")"

  run "$FOCUS" edit 1
  [ "$status" -eq 0 ]
  [[ "$output" == "$FOCUS_KANBAN_DIR/sample-backlog.md" ]]

  local updated_after
  updated_after="$(grep '^updated:' "$FOCUS_KANBAN_DIR/sample-backlog.md")"
  [ "$updated_after" = "$updated_before" ]
}

@test "edit in non-TTY prints valid file path" {
  load_fixture sample-backlog.md
  local updated_before
  updated_before="$(grep '^updated:' "$FOCUS_KANBAN_DIR/sample-backlog.md")"

  run "$FOCUS" edit sample-backlog
  [ "$status" -eq 0 ]
  [ -f "$output" ]

  local updated_after
  updated_after="$(grep '^updated:' "$FOCUS_KANBAN_DIR/sample-backlog.md")"
  [ "$updated_after" = "$updated_before" ]
}

# ── Milestone tests ──────────────────────────────────────────

@test "new --type milestone sets type field" {
  run "$FOCUS" new "Big ship" "web" --type milestone
  [ "$status" -eq 0 ]
  [[ "$output" =~ "milestone" ]]
  run grep '^type: milestone' "$FOCUS_KANBAN_DIR/big-ship.md"
  [ "$status" -eq 0 ]
  run grep '^status: backlog' "$FOCUS_KANBAN_DIR/big-ship.md"
  [ "$status" -eq 0 ]
}

@test "new --type with unknown value fails" {
  run "$FOCUS" new "Thing" --type epic
  [ "$status" -eq 1 ]
  [[ "$output" =~ "unknown type" ]]
}

@test "milestone new is shortcut for --type milestone" {
  run "$FOCUS" milestone new "Release 1" "web"
  [ "$status" -eq 0 ]
  run grep '^type: milestone' "$FOCUS_KANBAN_DIR/release-1.md"
  [ "$status" -eq 0 ]
  run grep '^project: web' "$FOCUS_KANBAN_DIR/release-1.md"
  [ "$status" -eq 0 ]
}

@test "milestone new without title fails" {
  run "$FOCUS" milestone new
  [ "$status" -eq 1 ]
  [[ "$output" =~ "usage:" ]]
}

@test "milestone add links card to milestone" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Ship it" "web"
  run "$FOCUS" milestone add 1 2
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Linked" ]]
  run grep '^milestone: 1' "$FOCUS_KANBAN_DIR/ship-it.md"
  [ "$status" -eq 0 ]
}

@test "milestone add rejects non-milestone parent" {
  "$FOCUS" new "Regular" "web"
  "$FOCUS" new "Child" "web"
  run "$FOCUS" milestone add 1 2
  [ "$status" -eq 1 ]
  [[ "$output" =~ "not a milestone" ]]
}

@test "milestone add rejects nesting milestones" {
  "$FOCUS" milestone new "Outer" "web"
  "$FOCUS" milestone new "Inner" "web"
  run "$FOCUS" milestone add 1 2
  [ "$status" -eq 1 ]
  [[ "$output" =~ "nest" ]]
}

@test "milestone <id> shows progress" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Task A" "web"
  "$FOCUS" new "Task B" "web"
  "$FOCUS" milestone add 1 2
  "$FOCUS" milestone add 1 3
  "$FOCUS" --force done 2
  run "$FOCUS" milestone 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Milestone" ]]
  [[ "$output" =~ "Progress:" ]]
  [[ "$output" =~ "1/2" ]]
  [[ "$output" =~ "Children:" ]]
  [[ "$output" =~ "Task A" ]]
  [[ "$output" =~ "Task B" ]]
}

@test "milestone <id> rejects non-milestone" {
  "$FOCUS" new "Regular" "web"
  run "$FOCUS" milestone 1
  [ "$status" -eq 1 ]
  [[ "$output" =~ "not a milestone" ]]
}

@test "milestone list shows all milestones with progress" {
  "$FOCUS" milestone new "Alpha" "web"
  "$FOCUS" milestone new "Beta" "web"
  "$FOCUS" new "Task" "web"
  "$FOCUS" milestone add 1 3
  run "$FOCUS" milestone list
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Alpha" ]]
  [[ "$output" =~ "Beta" ]]
  [[ "$output" =~ "0/1" ]]
  [[ "$output" =~ "0/0" ]]
}

@test "milestone list empty shows none message" {
  run "$FOCUS" milestone list
  [ "$status" -eq 0 ]
  [[ "$output" =~ "no milestones" ]]
}

@test "done on milestone with unfinished children blocks" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child" "web"
  "$FOCUS" milestone add 1 2
  run "$FOCUS" done 1
  [ "$status" -eq 1 ]
  [[ "$output" =~ "unfinished child" ]]
}

@test "done --force on milestone bypasses child check" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child" "web"
  "$FOCUS" milestone add 1 2
  run "$FOCUS" --force done 1
  [ "$status" -eq 0 ]
  run grep '^status: done' "$FOCUS_KANBAN_DIR/launch.md"
  [ "$status" -eq 0 ]
}

@test "done on milestone with all children done succeeds" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child" "web"
  "$FOCUS" milestone add 1 2
  "$FOCUS" --force done 2
  run "$FOCUS" done 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Done: 1" ]]
}

@test "done on empty milestone succeeds" {
  "$FOCUS" milestone new "Launch" "web"
  run "$FOCUS" done 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Done: 1" ]]
}

@test "show on child card displays milestone context" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child task" "web"
  "$FOCUS" milestone add 1 2
  run "$FOCUS" show 2
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Milestone: #1 Launch" ]]
}

@test "show on milestone displays progress and children" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child" "web"
  "$FOCUS" milestone add 1 2
  run "$FOCUS" show 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Milestone" ]]
  [[ "$output" =~ "Progress:" ]]
  [[ "$output" =~ "Children:" ]]
  [[ "$output" =~ "Child" ]]
}

@test "cross-project milestone groups children with different projects" {
  "$FOCUS" milestone new "Big initiative" "cross-project"
  "$FOCUS" new "Backend task" "api"
  "$FOCUS" new "Frontend task" "web"
  "$FOCUS" milestone add 1 2
  "$FOCUS" milestone add 1 3
  run "$FOCUS" milestone 1
  [ "$status" -eq 0 ]
  [[ "$output" =~ "2/2" ]] || [[ "$output" =~ "0/2" ]]
  [[ "$output" =~ "Backend task" ]]
  [[ "$output" =~ "Frontend task" ]]
}

@test "board groups active milestones" {
  "$FOCUS" milestone new "Launch" "web"
  "$FOCUS" new "Child" "web"
  "$FOCUS" milestone add 1 2
  run "$FOCUS" board
  [ "$status" -eq 0 ]
  [[ "$output" =~ "MILESTONES" ]]
  [[ "$output" =~ "Launch" ]]
}
