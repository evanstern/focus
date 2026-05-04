# bash completion for focus.
#
# Install: eval "$(focus completions bash)"
#
# Subcommands and flags are statically known. Dynamic candidates
# (card ids, statuses, priorities, types) come from the binary
# itself via `focus _complete <kind>` so the script never drifts
# from the Go source.

_focus() {
    local cur prev words cword
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion || return
    else
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    local subcommands="init new show edit board list activate park done kill revive reindex epic mcp tui completions version help"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$subcommands" -- "$cur") )
        return 0
    fi

    local sub="${words[1]}"

    case "$prev" in
        --priority)
            COMPREPLY=( $(compgen -W "$(focus _complete priorities 2>/dev/null)" -- "$cur") )
            return 0
            ;;
        --type)
            COMPREPLY=( $(compgen -W "$(focus _complete types 2>/dev/null)" -- "$cur") )
            return 0
            ;;
        --project|--epic|--owner|--tag|--slug)
            return 0
            ;;
    esac

    if [[ "$cur" == --* ]]; then
        local flags=""
        case "$sub" in
            new)
                flags="--project --priority --epic --type --slug"
                ;;
            list)
                flags="--project --priority --epic --owner --tag --type"
                ;;
            activate|done)
                flags="--force"
                ;;
            epic)
                flags="--force"
                ;;
        esac
        COMPREPLY=( $(compgen -W "$flags" -- "$cur") )
        return 0
    fi

    case "$sub" in
        show|edit|kill)
            COMPREPLY=( $(compgen -W "$(focus _complete ids 2>/dev/null)" -- "$cur") )
            ;;
        activate)
            COMPREPLY=( $(compgen -W "$(focus _complete ids --status backlog 2>/dev/null)" -- "$cur") )
            ;;
        park|done)
            COMPREPLY=( $(compgen -W "$(focus _complete ids --status active 2>/dev/null)" -- "$cur") )
            ;;
        revive)
            COMPREPLY=( $(compgen -W "$(focus _complete ids --status archived 2>/dev/null)" -- "$cur") )
            ;;
        list)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(focus _complete statuses 2>/dev/null)" -- "$cur") )
            fi
            ;;
        epic)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "list add $(focus _complete ids --type epic 2>/dev/null)" -- "$cur") )
            fi
            ;;
        completions)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
            fi
            ;;
        mcp)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "serve" -- "$cur") )
            fi
            ;;
    esac
    return 0
}

complete -F _focus focus
