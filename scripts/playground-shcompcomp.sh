#!/usr/bin/env bash

# shellcheck disable=SC2064
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

# todo: reload when this changes and py script changes (WHEN ANYTHING IN THIS DIR CHANGES)
# todo: reload when type "rs\n"
# ./tmux-watch.sh "pumpitcli " "$HOME/repos/pumpit"

script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
proj_dir=$(realpath "$script_dir/..")
input="$1"
#auto_complete_script=$(realpath -e "$script_dir/bashcompletils-lib.sh")
#starting_command="$1"
#cli_name="$(echo "$starting_command" | cut -d' ' -f1)"
#cli_dir="$(dirname "$(which "$cli_name")")" s
#starting_dir="$(realpath "$2")"sdf

log () {
  if [[ -z "$log_fd" ]]; then
    declare -g log_fd
    exec {log_fd}>>"$HOME/bashscript.log"
  fi
  local realtime="$EPOCHREALTIME"
  local seconds="${realtime%%.*}"
  local ms_str="${realtime##"$seconds".}"
  ms_str="${ms_str:0:3}"
  local -i ms=10#"$ms_str"
  printf '[%(%I:%M:%S)T.%03d] [%s] %s\n' "$seconds" "$ms" "${BASH_SOURCE[1]##*/}" "$*" 1>&"$log_fd"
}

session_name="shcompcomp-playground"
tmux kill-session -t "$session_name" 2>/dev/null
tmux new-session -d -s "$session_name" -n "window"

sendstrings () {
  tmux send-keys -t "=$session_name:=window.0" "$@"
}

inotify_action () {
  shcomp2 - < "$proj_dir/.tmuxinfile"
  sendstrings "C-c; source /tmp/tmuxfile.bash; clear" Enter
  sendstrings "$input"
}

(
  last_hash=$(md5sum "$script_dir/../build/shcomp2")
  inotify_action
  inotifywait -q -m -r -e "moved_to,create,modify,delete" "$proj_dir" | \
  while read -r dir _ filename; do
    file="$dir$filename"
    path_rel="$(realpath --relative-to="$proj_dir" "$file")"

    [[ "${path_rel: -1}" == "~" ]] && continue
    [[ "$path_rel" =~ ^(.git/|.idea/|scratch/|archive/|compile/) ]] && continue
    [[ "$path_rel" =~ (.lock|.log)$ ]] && continue
    if [[ "$path_rel" =~ \.tmuxinfile$ || "$path_rel" =~ shcomp2$ ]]; then
      current_hash=$(md5sum "$script_dir/../build/shcomp2")
      if [[ "$path_rel" =~ shcomp2$ && "$last_hash" != "$current_hash" ]]; then
        last_hash="$current_hash"
        log "reloading: binary change"
        inotify_action
      elif [[ "$path_rel" =~ \.tmuxinfile$ ]]; then
        log "reloading: config change"
        inotify_action
      fi
    fi
  done
) &

if [ -n "${TMUX:-}" ]; then tmux switch-client -t "=$session_name:0.0"; else tmux attach-session -t "=$session_name:0.0"; fi

#tmux send-keys -t "=$session_name:=window.0" "C-c" Enter
#tmux send-keys -t "=$session_name:=window.0" "source $auto_complete_script" Enter
#tmux send-keys -t "=$session_name:=window.0" "rm -rf ~/.cache/bashcompletils/ && reset_config $cli_name" Enter
#tmux send-keys -t "=$session_name:=window.0" "cd $starting_dir && export PATH=\"\$PATH:$cli_dir\"" Enter
#tmux send-keys -t "=$session_name:=window.0" "register_python_auto_gen $initial_args" Enter
#tmux send-keys -t "=$session_name:=window.0" 'clear && echo "reloaded script"' Enter
#tmux send-keys -t "=$session_name:=window.0" "$starting_command" Tab


#script_watcher() {
#    inotifywait -q -m -r -e modify,delete,create "$auto_complete_script" | \
#    while read -r _directory _action _file; do
#        initial_args="$(cat "/tmp/bashcompletils-$cli_name-initial-args")"
#        log "initial args: $initial_args"
#        tmux send-keys -t '=completer-session:=completer-window.0' 'C-c' Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' "source $auto_complete_script" Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' "cd $starting_dir && export PATH=\"\$PATH:$cli_dir\"" Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' "rm -rf ~/.cache/bashcompletils/ && reset_config $cli_name" Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' "register_python_auto_gen $initial_args" Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' 'clear && echo "reloaded script"' Enter
#        tmux send-keys -t '=completer-session:=completer-window.0' "$starting_command" Tab
#        log "reloaded $(basename "$auto_complete_script")"
#    done
#}

#script_watcher &
#pid_script_watcher=$!
#
## start session
#tmux kill-session -t "completer-session" 2>/dev/null || true
#tmux new-session -d -s "completer-session" -n "completer-window"
#
## pane: logs
#tmux split-window -h -t '=completer-session:=completer-window'
#tmux send-keys -t '=completer-session:=completer-window.1' 'tail -f ~/mybash.log' Enter
#
## source from start
## todo: combine this
#initial_args="$(cat "/tmp/bashcompletils-$cli_name-initial-args")"
#tmux send-keys -t '=completer-session:=completer-window.0' 'C-c' Enter
#tmux send-keys -t '=completer-session:=completer-window.0' "source $auto_complete_script" Enter
#tmux send-keys -t '=completer-session:=completer-window.0' "rm -rf ~/.cache/bashcompletils/ && reset_config $cli_name" Enter
#tmux send-keys -t '=completer-session:=completer-window.0' "cd $starting_dir && export PATH=\"\$PATH:$cli_dir\"" Enter
#tmux send-keys -t '=completer-session:=completer-window.0' "register_python_auto_gen $initial_args" Enter
#tmux send-keys -t '=completer-session:=completer-window.0' 'clear && echo "reloaded script"' Enter
#tmux send-keys -t '=completer-session:=completer-window.0' "$starting_command" Tab
#
## attach to session
#
#kill $pid_script_watcher
