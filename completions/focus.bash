# bash completion for focus

_focus() {
  local cur prev commands statuses
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  commands="board new show activate park kill done edit intent wip list init setup completions version help"
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
      # Complete with card slugs (filenames without .md)
      local kanban_dir
      kanban_dir="${FOCUS_KANBAN_DIR:-}"
      if [[ -z "$kanban_dir" ]] && [[ -f "${FOCUS_HOME:-$HOME/.focus}/config" ]]; then
        kanban_dir=$(sed -n 's/^kanban_dir:[[:space:]]*//p' "${FOCUS_HOME:-$HOME/.focus}/config" | head -1)
      fi
      kanban_dir="${kanban_dir:-${FOCUS_HOME:-$HOME/.focus}/kanban}"
      if [[ -n "$kanban_dir" ]] && [[ -d "$kanban_dir" ]]; then
        local slugs
        slugs=$(find "$kanban_dir" -maxdepth 1 -name '*.md' -exec basename {} .md \;)
        COMPREPLY=( $(compgen -W "$slugs" -- "$cur") )
      fi
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
