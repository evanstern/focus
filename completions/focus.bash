# bash completion for focus

_focus_completion_kanban_dir() {
  local kanban_dir="${FOCUS_KANBAN_DIR:-}"
  if [[ -z "$kanban_dir" ]] && [[ -f "${FOCUS_HOME:-$HOME/.focus}/config" ]]; then
    kanban_dir=$(sed -n 's/^kanban_dir:[[:space:]]*//p' "${FOCUS_HOME:-$HOME/.focus}/config" | head -1)
  fi
  echo "${kanban_dir:-${FOCUS_HOME:-$HOME/.focus}/kanban}"
}

_focus_completion_read_field() {
  local file="$1" field="$2"
  sed -n '/^---$/,/^---$/p' "$file" | grep "^${field}:" | head -1 | sed "s/^${field}:[[:space:]]*//"
}

_focus_completion_truncate() {
  local text="$1" max="$2"
  if (( ${#text} <= max )); then
    printf '%s' "$text"
  elif (( max > 3 )); then
    printf '%s...' "${text:0:$((max - 3))}"
  else
    printf '%s' "${text:0:$max}"
  fi
}

_focus_complete_cards() {
  local cur="$1" cmd="$2"
  local kanban_dir
  kanban_dir="$(_focus_completion_kanban_dir)"
  [[ -n "$kanban_dir" ]] && [[ -d "$kanban_dir" ]] || return 0

  local ids=() descriptions=()
  local f id title project status priority
  for f in "$kanban_dir"/*.md; do
    [[ -f "$f" ]] || continue
    id="$(_focus_completion_read_field "$f" "id")"
    [[ -n "$id" ]] || continue
    status="$(_focus_completion_read_field "$f" "status")"

    case "$cmd" in
      activate) [[ "$status" == "backlog" ]] || continue ;;
      park)     [[ "$status" == "active" ]] || continue ;;
      done)     [[ "$status" == "active" ]] || continue ;;
      kill)     [[ "$status" == "backlog" || "$status" == "active" || "$status" == "parked" ]] || continue ;;
      show|edit) ;;
    esac

    title="$(_focus_completion_truncate "$(_focus_completion_read_field "$f" "title")" 40)"
    project="$(_focus_completion_truncate "$(_focus_completion_read_field "$f" "project")" 24)"
    priority="$(_focus_completion_read_field "$f" "priority")"
    ids+=("$id")
    descriptions+=("$(printf '%-4s  %-40s  %-24s  %-8s  %s' "$id" "$title" "${project:-}" "$status" "$priority")")
  done

  local sorted
  sorted=$(for i in "${!ids[@]}"; do printf '%s\t%s\n' "${ids[$i]}" "${descriptions[$i]}"; done | sort -t $'\t' -k1,1n)

  ids=() descriptions=()
  while IFS=$'\t' read -r id desc; do
    [[ -n "$id" ]] || continue
    ids+=("$id")
    descriptions+=("$desc")
  done <<< "$sorted"

  if [[ -n "$cur" ]]; then
    local id
    for id in "${ids[@]}"; do
      [[ "$id" == "$cur"* ]] && COMPREPLY+=("$id")
    done
  else
    COMPREPLY=("${ids[@]}")
  fi

  if (( ${#COMPREPLY[@]} > 1 )) || { (( ${#COMPREPLY[@]} == 1 )) && [[ -z "$cur" ]]; }; then
    printf '\n' >&2
    printf '%-4s  %-40s  %-24s  %-8s  %s\n' 'ID' 'Title' 'Project' 'Status' 'Priority' >&2
    local desc
    for desc in "${descriptions[@]}"; do
      printf '%s\n' "$desc" >&2
    done
    printf '\n' >&2
  fi
}

_focus() {
  local cur prev commands statuses
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  commands="board new milestone show activate park kill done edit intent wip list tui init setup completions version help"
  statuses="active backlog done parked killed"

  case "$prev" in
    focus)
      COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
      return 0
      ;;
    list)
      COMPREPLY=( $(compgen -W "$statuses" -- "$cur") )
      return 0
      ;;
    show|activate|park|kill|done|edit)
      _focus_complete_cards "$cur" "$prev"
      return 0
      ;;
    --project|--priority)
      return 0
      ;;
  esac

  if [[ "$cur" == -* ]]; then
    COMPREPLY=( $(compgen -W "--force --quiet --no-color --project --priority" -- "$cur") )
    return 0
  fi
}

complete -F _focus focus
