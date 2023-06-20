#!/usr/bin/env bash

trap "exit_cleanup" SIGINT SIGTERM EXIT

script_name=$(basename -- "${BASH_SOURCE[0]}")
script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
compile_dir=$(realpath "$script_dir/../compile")
proj_dir=$(realpath "$script_dir/..")
input="$1"
session_name="shcompcomp-playground"
proc_id=$$

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

exit_cleanup () {
  trap - SIGTERM && kill -- -"$$"
}

sendstrings () {
  tmux send-keys -t "=$session_name:=window.0" "$@"
}

inotify_action () {
  if [[ $1 != first ]]; then
    sendstrings C-c
  fi
  prompt="$(tmux capture-pane -b temp-capture-buffer -E - -p)"
  prompt="${prompt##*"$ "}"
  prompt="${prompt%%^C}"
  shcomp2 - < "$proj_dir/.tmuxinfile"
  sendstrings "C-c; source $compile_dir/tmuxfile.bash; clear" Enter
  if [[ -n "$prompt" ]]; then
    sendstrings "$prompt"
  else
    sendstrings "bind '\"\C-d\": \"\C-u\C-d\"'; clear; echo press ctrl-d to exit..." Enter # todo: do this in init file
    sendstrings "$input"
  fi
}

tmux kill-session -t "$session_name" 2>/dev/null
tmux new-session -d -s "$session_name" -n "window"

(
  last_hash=$(md5sum "$script_dir/../build/shcomp2")
  inotify_action first
  inotifywait -q -m -r -e "moved_to,close_write" "$proj_dir" | \
  while read -r dir events filename; do
    file="$dir$filename"
    path_rel="$(realpath --relative-to="$proj_dir" "$file")"

    if [[ $path_rel =~ \.tmuxinfile$ && $events =~ CLOSE_WRITE ]]; then
      log "reloading: config change"
      inotify_action
    elif [[ $path_rel =~ shcomp2$ && $events =~ MOVED_TO ]]; then
      current_hash=$(md5sum "$script_dir/../build/shcomp2")
      if [[ $last_hash != "$current_hash" ]]; then
        log "reloading: binary change"
        last_hash="$current_hash"
        inotify_action
      fi
    elif [[ $file =~ "$script_name"$ && $events =~ CLOSE_WRITE ]]; then
      tmux kill-session -t "$session_name" 2>/dev/null
      echo "exiting on change to: $script_name"
      pkill -P "$proc_id"
    fi
  done
) &

if [ -n "${TMUX:-}" ]; then tmux switch-client -t "=$session_name:0.0"; else tmux attach-session -t "=$session_name:0.0"; fi
