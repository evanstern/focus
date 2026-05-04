# fish completion for focus.
#
# Install: focus completions fish > ~/.config/fish/completions/focus.fish

function __focus_no_subcommand
    set -l cmd (commandline -opc)
    if test (count $cmd) -lt 2
        return 0
    end
    return 1
end

function __focus_using
    set -l cmd (commandline -opc)
    if test (count $cmd) -ge 2
        if test "$cmd[2]" = "$argv[1]"
            return 0
        end
    end
    return 1
end

function __focus_ids
    focus _complete ids $argv 2>/dev/null
end

complete -c focus -f

complete -c focus -n __focus_no_subcommand -a init        -d 'Create a .focus/ board here'
complete -c focus -n __focus_no_subcommand -a new         -d 'Create a new card'
complete -c focus -n __focus_no_subcommand -a show        -d 'Render card detail'
complete -c focus -n __focus_no_subcommand -a edit        -d 'Open INDEX.md in $EDITOR'
complete -c focus -n __focus_no_subcommand -a board       -d 'Show active + backlog'
complete -c focus -n __focus_no_subcommand -a list        -d 'Flat list with filters'
complete -c focus -n __focus_no_subcommand -a activate    -d 'backlog -> active'
complete -c focus -n __focus_no_subcommand -a park        -d 'active -> backlog'
complete -c focus -n __focus_no_subcommand -a done        -d 'active -> done'
complete -c focus -n __focus_no_subcommand -a kill        -d 'any -> archived'
complete -c focus -n __focus_no_subcommand -a revive      -d 'archived -> backlog'
complete -c focus -n __focus_no_subcommand -a reindex     -d 'Rebuild index.json'
complete -c focus -n __focus_no_subcommand -a epic        -d 'Epic operations'
complete -c focus -n __focus_no_subcommand -a mcp         -d 'JSON-RPC server'
complete -c focus -n __focus_no_subcommand -a tui         -d 'Open the interactive board'
complete -c focus -n __focus_no_subcommand -a completions -d 'Print shell completion script'
complete -c focus -n __focus_no_subcommand -a version     -d 'Print version'
complete -c focus -n __focus_no_subcommand -a help        -d 'Show help'

complete -c focus -n '__focus_using show'     -a '(__focus_ids)'
complete -c focus -n '__focus_using edit'     -a '(__focus_ids)'
complete -c focus -n '__focus_using kill'     -a '(__focus_ids)'
complete -c focus -n '__focus_using activate' -a '(__focus_ids --status backlog)'
complete -c focus -n '__focus_using park'     -a '(__focus_ids --status active)'
complete -c focus -n '__focus_using done'     -a '(__focus_ids --status active)'
complete -c focus -n '__focus_using revive'   -a '(__focus_ids --status archived)'

complete -c focus -n '__focus_using list' -a 'active backlog done archived'

complete -c focus -n '__focus_using completions' -a 'bash zsh fish'
complete -c focus -n '__focus_using mcp' -a 'serve'

complete -c focus -n '__focus_using epic' -a 'list add'
complete -c focus -n '__focus_using epic' -a '(__focus_ids --type epic)'

complete -c focus -n '__focus_using new'  -l priority -xa 'p0 p1 p2 p3'
complete -c focus -n '__focus_using new'  -l type     -xa 'card epic'
complete -c focus -n '__focus_using new'  -l project  -x
complete -c focus -n '__focus_using new'  -l epic     -x
complete -c focus -n '__focus_using new'  -l slug     -x

complete -c focus -n '__focus_using list' -l priority -xa 'p0 p1 p2 p3'
complete -c focus -n '__focus_using list' -l type     -xa 'card epic'
complete -c focus -n '__focus_using list' -l project  -x
complete -c focus -n '__focus_using list' -l epic     -x
complete -c focus -n '__focus_using list' -l owner    -x
complete -c focus -n '__focus_using list' -l tag      -x
