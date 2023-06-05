#!/usr/bin/env bash

# shellcheck disable=SC2064
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

# todo: reload when this changes and py script changes (WHEN ANYTHING IN THIS DIR CHANGES)
# todo: reload when type "rs\n"
# ./tmux-watch.sh "pumpitcli " "$HOME/repos/pumpit"

script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
auto_complete_script=$(realpath -e "$script_dir/bashcompletils-lib.sh")
starting_command="$1"
cli_name="$(echo "$starting_command" | cut -d' ' -f1)"
cli_dir="$(dirname "$(which "$cli_name")")"
starting_dir="$(realpath "$2")"

log() { echo -e "$(date --iso-8601=ns) $*" >> ~/bashscript.log; }

script_watcher() {
    inotifywait -q -m -r -e modify,delete,create "$auto_complete_script" | \
    while read -r _directory _action _file; do
        initial_args="$(cat "/tmp/bashcompletils-$cli_name-initial-args")"
        log "initial args: $initial_args"
        tmux send-keys -t '=completer-session:=completer-window.0' 'C-c' Enter
        tmux send-keys -t '=completer-session:=completer-window.0' "source $auto_complete_script" Enter
        tmux send-keys -t '=completer-session:=completer-window.0' "cd $starting_dir && export PATH=\"\$PATH:$cli_dir\"" Enter
        tmux send-keys -t '=completer-session:=completer-window.0' "rm -rf ~/.cache/bashcompletils/ && reset_config $cli_name" Enter
        tmux send-keys -t '=completer-session:=completer-window.0' "register_python_auto_gen $initial_args" Enter
        tmux send-keys -t '=completer-session:=completer-window.0' 'clear && echo "reloaded script"' Enter
        tmux send-keys -t '=completer-session:=completer-window.0' "$starting_command" Tab
        log "reloaded $(basename "$auto_complete_script")"
    done
}

script_watcher &
pid_script_watcher=$!

# start session
tmux kill-session -t "completer-session" 2>/dev/null || true
tmux new-session -d -s "completer-session" -n "completer-window"

# pane: logs
tmux split-window -h -t '=completer-session:=completer-window'
tmux send-keys -t '=completer-session:=completer-window.1' 'tail -f ~/bashscript.log' Enter

# source from start
# todo: combine this
initial_args="$(cat "/tmp/bashcompletils-$cli_name-initial-args")"
tmux send-keys -t '=completer-session:=completer-window.0' 'C-c' Enter
tmux send-keys -t '=completer-session:=completer-window.0' "source $auto_complete_script" Enter
tmux send-keys -t '=completer-session:=completer-window.0' "rm -rf ~/.cache/bashcompletils/ && reset_config $cli_name" Enter
tmux send-keys -t '=completer-session:=completer-window.0' "cd $starting_dir && export PATH=\"\$PATH:$cli_dir\"" Enter
tmux send-keys -t '=completer-session:=completer-window.0' "register_python_auto_gen $initial_args" Enter
tmux send-keys -t '=completer-session:=completer-window.0' 'clear && echo "reloaded script"' Enter
tmux send-keys -t '=completer-session:=completer-window.0' "$starting_command" Tab

# attach to session
if [ -n "${TMUX:-}" ]; then tmux switch-client -t "=completer-session:0.0"; else tmux attach-session -t "=completer-session:0.0"; fi

kill $pid_script_watcher
