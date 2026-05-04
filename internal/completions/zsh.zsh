#compdef focus
#
# zsh completion for focus.
#
# Install: eval "$(focus completions zsh)"

_focus() {
    local -a subcommands
    subcommands=(
        'init:Create a .focus/ board here'
        'new:Create a new card'
        'show:Render card detail'
        'edit:Open INDEX.md in $EDITOR'
        'board:Show active + backlog'
        'list:Flat list with filters'
        'activate:backlog -> active'
        'park:active -> backlog'
        'done:active -> done'
        'kill:any -> archived'
        'revive:archived -> backlog'
        'reindex:Rebuild index.json'
        'epic:Epic operations'
        'mcp:JSON-RPC server'
        'tui:Open the interactive board'
        'completions:Print shell completion script'
        'version:Print version'
        'help:Show help'
    )

    local context state state_descr line
    typeset -A opt_args

    _arguments -C \
        '1: :->subcmd' \
        '*:: :->args'

    case $state in
        subcmd)
            _describe -t commands 'focus subcommand' subcommands
            ;;
        args)
            case $line[1] in
                show|edit|kill)
                    _focus_ids
                    ;;
                activate)
                    _focus_ids --status backlog
                    ;;
                park|done)
                    _focus_ids --status active
                    ;;
                revive)
                    _focus_ids --status archived
                    ;;
                list)
                    _arguments \
                        '1:status:(active backlog done archived)' \
                        '--project[filter by project]:project:' \
                        '--priority[filter by priority]:priority:(p0 p1 p2 p3)' \
                        '--epic[filter by epic id]:epic:' \
                        '--owner[filter by owner]:owner:' \
                        '--tag[filter by tag]:tag:' \
                        '--type[filter by type]:type:(card epic)'
                    ;;
                new)
                    _arguments \
                        '--project[project]:project:' \
                        '--priority[priority]:priority:(p0 p1 p2 p3)' \
                        '--epic[epic id]:epic:' \
                        '--type[type]:type:(card epic)' \
                        '--slug[slug]:slug:'
                    ;;
                epic)
                    if (( CURRENT == 2 )); then
                        local -a epic_subs
                        epic_subs=(list add)
                        _alternative \
                            'subcmds:epic subcommand:compadd -a epic_subs' \
                            'epics:epic id:_focus_ids --type epic'
                    fi
                    ;;
                completions)
                    _arguments '1:shell:(bash zsh fish)'
                    ;;
                mcp)
                    _arguments '1:subcommand:(serve)'
                    ;;
            esac
            ;;
    esac
}

_focus_ids() {
    local -a ids
    ids=(${(f)"$(focus _complete ids "$@" 2>/dev/null)"})
    compadd -a ids
}

compdef _focus focus
