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
  run "$FOCUS" edit 1
  [ "$status" -eq 0 ]
  [[ "$output" == "$FOCUS_KANBAN_DIR/sample-backlog.md" ]]
}

@test "edit in non-TTY prints valid file path" {
  load_fixture sample-backlog.md
  run "$FOCUS" edit sample-backlog
  [ "$status" -eq 0 ]
  [ -f "$output" ]
}
