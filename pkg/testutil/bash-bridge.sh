cat - <<'EOF' >/tmp/complete-withexpect-init.sh
bind 'set bell-style none'
if [ -f /usr/share/bash-completion/bash_completion ]; then
  source /usr/share/bash-completion/bash_completion
elif [ -f /etc/bash_completion ]; then
  source /etc/bash_completion
fi
EOF

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

complete_str_with_expect() {
  input="$1"
  complete_script="$2"
  out_file=$(mktemp /tmp/shcompcomp-expect-outfile.XXXXXX)

  # todo: probably not the most efficient. not really familiar with expect/tcl
  expect - "$input" "$out_file" "$complete_script" <<'EOF' >/dev/null
    set input [lindex $argv 0]
    set out_file [lindex $argv 1]
    set complete_script [lindex $argv 2]

    spawn env -i PS1=>>> bash --noprofile --norc
    expect -re {>>>}

    send "source /tmp/complete-withexpect-init.sh\r"
    expect -re {>>>}

    send "source $complete_script\r"
    expect -re {>>>}

    send "$input\t\r"
    expect -re {(.*)>>>}

    set out_fd [open $out_file "w"]
    puts -nonewline $out_fd "$expect_out(1,string)"
    close $out_fd

    send "exit\r"
EOF

  sed -i 's/[^[:print:]]//g' "$out_file"
  mapfile -tn 0 completed_lines <"$out_file"
  complete_line=${completed_lines[0]}
  rm "$out_file" 2>/dev/null || true
  printf "%s\n" "$complete_line"
}

complete_str() {
  local input_line="$1"

  # fixes: "compopt: not currently executing completion function"
  # allows compopt calls without giving the cmdname arg
  # compopt +o nospace instead of compopt +o nospace mycommand
  compopt() {
    builtin compopt "$@" "$__complete_str_compopt_current_cmd"
  }

  IFS=', ' read -r -a comp_words <<<"$input_line"
  if [[ "$input_line" =~ " "$ ]]; then comp_words+=(""); fi

  cmd_name="${comp_words[0]}"
  COMP_LINE="$input_line"
  COMP_WORDS=("${comp_words[@]}")
  COMP_CWORD="$((${#comp_words[@]} - 1))"
  COMP_POINT="$(("${#input_line}" + 0))"

  complete_func="$(complete -p "$cmd_name" | awk '{print $(NF-1)}')"
  __complete_str_compopt_current_cmd="$cmd_name"
  "$complete_func"
  __complete_str_compopt_current_cmd=""
  unset compopt

  printf '%s\n' "${COMPREPLY[*]}"
}

if [ -f /usr/share/bash-completion/bash_completion ]; then
  source /usr/share/bash-completion/bash_completion
elif [ -f /etc/bash_completion ]; then
  source /etc/bash_completion
fi

while IFS= read -r line; do
  IFS=$'\n' read -d "" -ra split <<< "${line//:/$'\n'}"
  test_method="${split[0]}"
  if [[ $test_method == bashfunc ]]; then
    test_line="${line##*:}"
    complete_str "$test_line"
  elif [[ $test_method == expecttcl ]]; then
    test_method="${split[0]}"
    test_file="${split[1]}"
    test_line="${line##*:*:}"
    complete_str_with_expect "$test_line" "$test_file"
  else
    >&2 echo "unknown test method: $test_method"
  fi
  printf '\0'
done
