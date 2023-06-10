#!/usr/bin/env bash

# https://stackoverflow.com/a/9505024/2276637
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
proj_dir="$(realpath "$script_dir")"
curr_dir="$PWD"
CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
MAGENTA='\033[0;35m'
NC='\033[0m'

cd "$proj_dir" || exit
source "/usr/share/bash-completion/bash_completion"
source "./bctils-lib.sh" || {
  echo -e "${RED}ERROR: sourcing bctils-lib.sh FAIL$NC"
  exit 1
}
cd "$curr_dir" || exit

if [[ -z "$BCTILS_COMPILE_TIME" ]]; then
  export BCTILS_COMPILE_TIME=0
fi
test_case="$*"
export BCTILS_COMPILE_DIR="$proj_dir/compile"

# https://serverfault.com/a/570651
exec 9>&2
exec 8> >(
  while IFS='' read -r line || [ -n "$line" ]; do
    echo -e "\033[31m${line}\033[0m"
  done
)
function redirect() { exec 2>&8; }
redirect

# todo: add line number to pass/fail or just fail
run_tests() {
  if [[ "$TEST_BENCHMARK" == 1 ]]; then
    run_benchmarks
    return
  fi

  all_result="${GREEN}SUCCESS${NC}"
  fail_count_str=""
  fail_count=0
  time_start=$(($(date +%s%N) / 1000000))

  current_suite "py_autogen simple end to end test"
  mkdir -p "/tmp/bctils-testing-simple-parser"
  cat - > "/tmp/bctils-testing-simple-parser/examplecli8.py" <<EOF
from argparse import ArgumentParser
parser = ArgumentParser()
parser.add_argument("--some-way2")
subparsers = parser.add_subparsers()
parser_cmd = subparsers.add_parser("sub-cmd-name")
parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"])
EOF
  bctils_autogen "/tmp/bctils-testing-simple-parser/examplecli8.py" --lang=py --source
  expect_cmd "python script is runnable" python "/tmp/bctils-testing-simple-parser/examplecli8.py"
  expect_complete_compreply "examplecli8.py " "sub-cmd-name --some-way2"
  expect_complete_compreply "examplecli8.py --some" "--some-way2"
  expect_complete_compreply "examplecli8.py sub-cmd-name " "c1 c2 c3"

  current_suite "bctils_autogen can choose out file"
  mkdir -p "/tmp/bctils-testing-simple-parser"
  echo "print('test')" > "/tmp/bctils-testing-simple-parser/examplecli9.py"
  bctils_autogen "/tmp/bctils-testing-simple-parser/examplecli9.py" --lang=py --source \
    --outfile="/tmp/bctils-testing-simple-parser/examplecli8.py.bash"
  expect_cmd "can specify outfile for autogen" test -f "/tmp/bctils-testing-simple-parser/examplecli8.py.bash"

  current_suite "bctils_autogen reloads on changes"
  mkdir -p "/tmp/bctils-testing-simple-parser"
  cat - > "/tmp/bctils-testing-simple-parser/examplecli10.py" <<EOF
from argparse import ArgumentParser
parser = ArgumentParser()
parser.add_argument("--some-way2")
subparsers = parser.add_subparsers()
parser_cmd = subparsers.add_parser("sub-cmd-name")
parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"])
EOF
  bctils_autogen "/tmp/bctils-testing-simple-parser/examplecli10.py" --lang=py --source
  expect_cmd "python script is runnable" python "/tmp/bctils-testing-simple-parser/examplecli10.py"
  expect_complete_compreply "examplecli10.py " "sub-cmd-name --some-way2"
  expect_complete_compreply "examplecli10.py --some" "--some-way2"
  expect_complete_compreply "examplecli10.py sub-cmd-name " "c1 c2 c3"
  cat - > "/tmp/bctils-testing-simple-parser/examplecli10.py" <<EOF
from argparse import ArgumentParser
parser = ArgumentParser()
parser.add_argument("--some-way2")
subparsers = parser.add_subparsers()
parser_cmd = subparsers.add_parser("sub-cmd-name")
parser_cmd.add_argument("arg1", choices=["c4", "c5", "c6"])
EOF
  sleep 0.001 # 1ms
  expect_complete_compreply "examplecli10.py sub-cmd-name " "c4 c5 c6"
  expect_complete_compreply "examplecli10.py " "sub-cmd-name --some-way2"
  expect_complete_compreply "examplecli10.py --some" "--some-way2"

  current_suite "bctils_autogen caches on md5 in variable"
  current_suite "bctils_autogen caches on md5 in file"

  current_suite "bctils_autogen takes a closure function to generate the source to analyze"
  touch /tmp/bctils-first-watch1
  touch /tmp/bctils-second-watch1
  closure_pipe_func () {
      cat <<EOF
from argparse import ArgumentParser
parser = ArgumentParser()
parser.add_argument("--awesome")
EOF
  }
  bctils_autogen - --lang=py --cliname "examplecli11" --source --closurepipe closure_pipe_func \
  --watch-file "/tmp/bctils-first-watch1" --watch-file "/tmp/bctils-second-watch1"
#  expect_complete_compreply "examplecli11 " "--awesome"

  current_suite "new cli"
  cat - << EOF | bctils
    cfg cli_name=boman
    opt "-h"
    opt "--help"
EOF

  current_suite "custom log"

  current_suite "source ~/.bashrc is FAST with MANY 'autogen calls'"

  current_suite "cache all compiles"
  current_suite "move tests into golang environment"
  current_suite "adding to .bashrc and removing it still adds time and is accumalative"

  current_suite "positionals without hints are recognized countwise"

  current_suite "py_autogen detect disabling --help/-h"
  current_suite "bctils_autogen specify out file"

  current_suite "exclusive options --vanilla --chocolate"
  current_suite "complete option value like --opt=value"
  current_suite "add flag to auto add = if only one arg option left and it requires an argument"
  current_suite "complete single -s type options like -f filepath"
  current_suite "complete single -s type options like -ffilepath (if that makes sense)"
  current_suite "arbitrary rules like -flags only before positionals"
  current_suite "arbitrary rules like --options values only or --options=values only for long args (getopt bug)"

  current_suite "simple options and arguments with nargs=*"
  current_suite "more complex autocomplete in different parts of command"
  current_suite "advanced subparsers with options + arguments at different levels"
  current_suite "cache based on input argument streams"
  current_suite "feature complete existing"
  current_suite "benchmark testing compilation"
  current_suite "benchmark testing compilation caching"
  current_suite "benchmark testing autogeneration of python script"
  current_suite "benchmark source bctils lib"
  current_suite "benchmark source compiled scripts"
  current_suite "use -- in util scripts to separate arguments from options"
  current_suite "allow single -longopt like golang"
  current_suite "allow opt=val and opt val"
  current_suite "tab complete opt -> opt="
  current_suite "choices for options with arguments"
  current_suite "scan python script for auto generate"
  current_suite "get compiled script version"
  current_suite "multiple functions to single file"
  current_suite "compiled scripts are slim and simplified"
  current_suite "provide custom functions -F to autocomplete arguments and options"
  current_suite "provide custom functions -F to autocomplete subparsers arguments and options"
  current_suite "provide custom functions -F to autocomplete option values"
  current_suite "provide custom functions -F to autocomplete subparsers option values"
  current_suite "nargs with known number"
  current_suite "nargs with unknown number"
  current_suite "invalid usages of bctil utility functions"
  current_suite "stateless in environment after compilation. no leftover variables."
  current_suite "zero logging when in production mode"
  current_suite "doesnt share variable state between different cli_name"
  current_suite "can provide autocompletion custom git extensions"
  current_suite "has full documentation"
  current_suite "compatable with bash,sh,zsh,ksh,msys2,fish,cygwin,bashwin (docker emulation)"
  current_suite "is fast with very large complete options like aws"
  current_suite "minimal forks in completion"
  current_suite "minimal memory usage in completion"
  current_suite "easy to install"
  current_suite "install with choice of plugins"
  current_suite "git plugin"
  current_suite "npm plugin"
  current_suite "autogen_py plugin"
  current_suite "autogen_node plugin"
  current_suite "autogen_golang plugin"
  current_suite "autogen_sh plugin"
  current_suite "compiled scripts are actually readable"
  current_suite "compiled scripts contain auto-generated comment and license"
  current_suite "project is licensed"
  current_suite "order of options added and argument choices is order shown"
  current_suite "all error messages match current script name"

  echo -e "${MAGENTA}DONE${NC}: run=$((($(date +%s%N) / 1000000) - time_start))ms gobuild=${BCTILS_COMPILE_TIME}ms (${all_result}${fail_count_str}) $(date '+%T.%3N')"

  # complete -F bashcompletils_autocomplete "example_cli"
  # expect_complete_compreply "example_cli " "pio channel deploy release streamermap"

  # add_positional_argument "example_cli" "pio channel deploy release streamermap"
  # add_positional_argument "example_cli.channel" "deploy rm"
  # add_option_argument "example_cli.channel.rm" "--all"
  # add_option_argument "example_cli.channel.deploy" "--print-only"
  # add_option_argument "example_cli.channel.deploy" "-v"
  # add_positional_argument "example_cli.release" --closure=_example_cli_branch_autocomplete
  # add_positional_argument "example_cli.release" "qa dev rob" --nargs="*"
  # add_positional_argument "example_cli.streamermap" "push pull"
  # add_positional_argument "example_cli.streamermap.push" "qa dev rob" --nargs="*"
  # complete -F bashcompletils_autocomplete "example_cli"
  # expect_complete_compreply "example_cli " "pio channel deploy release streamermap"
  # expect_complete_compreply "example_cli channel " "deploy rm"
  # expect_complete_compreply "example_cli channel rm " "--all"
  # expect_complete_compreply "example_cli channel rm --all" "--all"
  # expect_complete_compreply "example_cli channel deploy " "--print-only -v"
  # expect_complete_compreply "example_cli release " "develop custom/func/response"
  # expect_complete_compreply "example_cli release qa " "qa dev rob"    # confusing. first arg is branch
  # expect_complete_compreply "example_cli release develop qa " "dev rob"
  # expect_complete_compreply "example_cli release develop qa dev ro" "rob"
  # expect_complete_compreply "example_cli release develop qa dev rob" ""
  # expect_complete_compreply "example_cli release develop " "qa dev rob"
  # expect_complete_compreply "example_cli stream" "streamermap"
  # expect_complete_compreply "example_cli streamermap " "push pull"
  # expect_complete_compreply "example_cli streamermap push " "qa dev rob"
  # expect_complete_compreply "example_cli streamermap pull " ""

  # register_python_auto_gen "$script_dir/py_autogen_fixture"
  # expect_complete_compreply "py_autogen_fixture " "channel deploy streamermap release release-notes pio"
  # expect_complete_compreply "py_autogen_fixture channel deploy --p" "--print-only"
  # expect_complete_compreply "py_autogen_fixture channel deploy -" "--print-only -n -v"
}

run_benchmarks() {
  __bctils_controlgroup_completer_1() {
    # shellcheck disable=SC2034
    local cword_index previous_word words current_word
    _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word
    mapfile -t COMPREPLY < <(compgen -W "c1 d2" -- "$current_word")
  }

  benchmark_results() {
    # benchmark_control_1
    local name="$1"
    local result="$2"
    local time_total="$3"
    local iterations="$4"
    printf "%s: '%s' average=%sms total=%sms (%s iterations)\n" \
      "$name" \
      "$result" \
      "$(bc <<<"scale=2; $time_total/$iterations")" \
      "$time_total" \
      "$iterations"
  }

  local iterations=${ITER:-1000}

  # setup testbenchmarkcontrol1
  complete -F __bctils_controlgroup_completer_1 "benchmark_control_1"
  current_suite "benchmark testing completions" && print_suite
  # setup testbenchmark1
  bctils_cli_register "benchmark_1"
  bctils_cli_add      "benchmark_1" opt "--key" --choices="val1 val2"
  bctils_cli_compile  "benchmark_1" --source

  # benchmark_control_1
  time_start=$(($(date +%s%N) / 1000000))
  i=-1
  while true; do
    i=$((i + 1))
    if [[ "$i" -gt "$iterations" ]]; then break; fi
    complete_cmd_str "benchmark_control_1 "
  done
  rslt_benchmark_control_1="$complete_cmd_str_result"
  time_benchmark_control_1=$((($(date +%s%N) / 1000000) - time_start))

  # benchmark_1
  time_start=$(($(date +%s%N) / 1000000))
  i=-1
  while true; do
    i=$((i + 1))
    if [[ "$i" -gt "$iterations" ]]; then break; fi
    complete_cmd_str "benchmark_1 --ke"
  done
  rslt_benchmark_1="$complete_cmd_str_result"
  time_benchmark_1=$((($(date +%s%N) / 1000000) - time_start))

  # benchmark compilation
  time_start=$(($(date +%s%N) / 1000000))
  local iter=-1 # todo: fix this. while i it's constantly reset to 0
  while true; do
    iter=$((iter + 1))
    if [[ "$iter" -gt "$iterations" ]]; then break; fi
    bctils_cli_register "benchmark_compile_1"
    bctils_cli_add "benchmark_compile_1" opt "--key" --choices="val1 val2"
    bctils_cli_compile "benchmark_compile_1" --source
  done
  time_benchmark_compile_1=$((($(date +%s%N) / 1000000) - time_start))

  # benchmark compilation register
  time_start=$(($(date +%s%N) / 1000000))
  local iter=-1 # todo: fix this. while i it's constantly reset to 0
  while true; do
    iter=$((iter + 1))
    if [[ "$iter" -gt "$iterations" ]]; then break; fi
    bctils_cli_register "benchmark_compilereg_1"
  done
  time_benchmark_compilereg_1=$((($(date +%s%N) / 1000000) - time_start))

  # benchmark compilation add argument
  bctils_cli_register "benchmark_compile_addarg_1"
  time_start=$(($(date +%s%N) / 1000000))
  local iter=-1 # todo: fix this. while i it's constantly reset to 0
  while true; do
    iter=$((iter + 1))
    if [[ "$iter" -gt "$iterations" ]]; then break; fi
    bctils_cli_add "benchmark_compile_addarg_1" opt "--key" --choices="val1 val2"
  done
  time_benchmark_compile_addarg_1=$((($(date +%s%N) / 1000000) - time_start))

  # benchmark compilation compile + source

  local time_total=0
  local iter=-1 # todo: fix this. while i it's constantly reset to 0
  while true; do
    iter=$((iter + 1))
    if [[ "$iter" -gt "$iterations" ]]; then break; fi
    bctils_cli_register "benchmark_compile_and_source_1"
    bctils_cli_add "benchmark_compile_and_source_1" opt "--key" --choices="val1 val2"
    time_start=$(($(date +%s%N) / 1000000))
    bctils_cli_compile "benchmark_compile_and_source_1" --source
    time_total=$((time_total + (($(date +%s%N) / 1000000) - time_start)))
  done
  time_benchmark_compile_and_source_1="$time_total"

  benchmark_results "benchmark_control_1" "$rslt_benchmark_control_1" "$time_benchmark_control_1" "$iterations"
  benchmark_results "benchmark_1" "$rslt_benchmark_1" "$time_benchmark_1" "$iterations"
  benchmark_results "benchmark_compile_1" "" "$time_benchmark_compile_1" "$iterations"
  benchmark_results "benchmark_compilereg_1" "" "$time_benchmark_compilereg_1" "$iterations"
  benchmark_results "benchmark_compile_addarg_1" "" "$time_benchmark_compile_addarg_1" "$iterations"
  benchmark_results "benchmark_compile_and_source_1" "" "$time_benchmark_compile_and_source_1" "$iterations"
}

expect_compile_success() {
  if [[ -n "$bctils_err" ]]; then
    fail_test "compile FAIL : $bctils_err"
  fi
}

expect_cmd() {
  print_suite
  local msg="$1"
  shift
  local cmd="$1"
  shift
  if "$cmd" "$@"; then
    pass_test "$msg"
  else
    fail_test "$msg"
  fi
}

current_suite_printed=0
current_suite_name=""
current_suite() {
  current_suite_name="${CYAN}====== $1${NC}"
  current_suite_printed=0
}

print_suite() {
  if [[ "$current_suite_printed" == 0 ]]; then
    echo -e "$current_suite_name"
    log "$current_suite_name"
    current_suite_printed=1
  fi
}

fail_test() {
  if [[ "$current_suite_printed" == 0 ]]; then
    echo -e "$current_suite_name"
    current_suite_printed=1
  fi
  echo -e "${RED}FAIL${NC}: '$1'"
  all_result="${RED}FAIL${NC}"
  fail_count=$((fail_count + 1))
  fail_count_str=" $fail_count"

  if [[ -n "$bctils_err" ]]; then
    echo "bctils_err: $bctils_err"
    echo "stopping tests..."
  fi
}

pass_test() {
  if [[ "$current_suite_printed" == 0 ]]; then
    echo -e "$current_suite_name"
    current_suite_printed=1
  fi
  echo -e "${GREEN}PASS${NC}: '$1'"

  if [[ -n "$bctils_err" ]]; then
    echo "bctils_err: $bctils_err" 1>&2
    echo "stopping tests..." 1>&2
    exit 1
  fi
}

complete_cmd_str() {
  local input_line="$1"
  declare -g complete_cmd_str_result

  # fixes: "compopt: not currently executing completion function"
  # allows compopt calls without giving the cmdname arg
  # compopt +o nospace instead of compopt +o nospace mycommand
  compopt () {
    builtin compopt "$@" "$__bctilstest_compopt_current_cmd"
  }

  IFS=', ' read -r -a comp_words <<<"$input_line"
  if [[ "$input_line" =~ " "$ ]]; then comp_words+=(""); fi

  cmd_name="${comp_words[0]}"
  COMP_LINE="$input_line"
  COMP_WORDS=("${comp_words[@]}")
  COMP_CWORD="$((${#comp_words[@]} - 1))"
  COMP_POINT="$(("${#input_line}" + 0))"

  complete_func="$(complete -p "$cmd_name" | awk '{print $(NF-1)}')"
  __bctilstest_compopt_current_cmd="$cmd_name"
  "$complete_func" &>/tmp/bashcompletils.out
  complete_cmd_str_result="${COMPREPLY[*]}"
  __bctilstest_compopt_current_cmd=""
  unset compopt
}

expect_complete_compreply() {
  test_name="$1"
  input_line="$1"
  expected_reply="$2"

  if [[ -n "$test_case" && "$test_case" != "$test_name" ]]; then
    return
  fi

  log "==== $test_name ===="
  complete_cmd_str "$input_line"
  actual_reply="${COMPREPLY[*]}"
  output=$(cat /tmp/bashcompletils.out)

  if [[ "$actual_reply" != "$expected_reply" ]]; then
    fail_test "$test_name"
    echo "actual   : '$actual_reply'"
    echo "expected : '$expected_reply'"
    if [[ "${#output}" -gt 0 ]]; then
      echo "$output"
    fi
    log "==== FAIL ===="
  else
    pass_test "$test_name"
    log "==== PASS ===="
  fi
}

_example_cli_branch_autocomplete() {
  # shellcheck disable=SC2034
  COMPREPLY=("develop" "custom/func/response")
  log "COMPREPLY: ${COMPREPLY[*]}"
}

if [[ "$TEST_RUN_MODE" == "RUN_TESTS_ONCE" ]]; then
  run_tests
else
  __bctilstest_inotify_loop() {
    local watch_file="$1"
    local events="$2"
    local dir_file="$3"

    BCTILS_COMPILE_TIME=0

    if [[ ! "$events" =~ .*"CLOSE_WRITE".* ]] || [[ "$dir_file" =~ ".lock"~?$ || "$dir_file" =~ "~"$ ]]; then
      return
    fi

    if [[ -n "$dir_file" ]]; then
      echo "file change: $dir_file $events"
    fi

    if [[ "$events" =~ .*"CLOSE_WRITE".* ]] && [[ "$dir_file" =~ ".go"$ || "$dir_file" == "complete-template.txt" ]]; then
      echo "rebuilding golang binary..."
      time_start=$(($(date +%s%N) / 1000000))
      if ! just build; then
        echo -e "${RED}ERROR: bctils FAIL to build${NC}"
        return
      fi
      BCTILS_COMPILE_TIME=$((($(date +%s%N) / 1000000) - time_start))
    fi

    __bctils_test_run_test_script
  }

  __bctils_test_run_test_script() {
    if go test -v "./pkg/generators"
    then
      BCTILS_COMPILE_TIME="$BCTILS_COMPILE_TIME" \
      TEST_RUN_MODE="RUN_TESTS_ONCE" \
      bash "$script_dir/bctils-tests.sh" || { echo -e "${RED}ERROR: $0 FAILED. exit_code=$? ${NC}"; }
    else
      echo -e "${RED}go tests failed${NC}"
    fi
    echo "waiting for changes..."
    echo "--------"
  }

  __bctilstest_kill_other_procs () {
    local pid gpid_matches script_name pid_i pid_killed script_name

    script_name="$(basename -- "${BASH_SOURCE[0]}")"
    current_group_id="$(ps -o pgid= "$$")"

    if [[ -z "$script_name" ]]; then
      echo -e "${RED}ERROR: script_name is empty. don't want to kill all processes${NC}"
      return 1
    fi

    # shellcheck disable=SC2009
    readarray -t gpid_matches < <(ps x -o "%r %a" | grep "$script_name" | grep -v "$current_group_id" | grep -v grep | awk '{$1=$1};1' | tr -s ' ' | cut -d ' ' -f1 | sort | uniq)
    if [[ "${#gpid_matches[@]}" -gt 0 ]]; then
      echo -e "${YELLOW}WARNING: found rogue $script_name processes ${NC}"
      echo -e "${YELLOW}killing pids ${gpid_matches[*]}${NC}"
      for pid in "${gpid_matches[@]}"; do
        kill -15 "-$pid"
      done
      while [[ "${#gpid_matches[@]}" -gt 0 ]]; do
        pid_killed=0
        for pid_i in "${!gpid_matches[@]}"; do
          if ! kill -0 "${gpid_matches[$pid_i]}" 2> /dev/null; then
            unset 'gpid_matches[$pid_i]'
            pid_killed=1
            break
          fi
        done
        if [[ "$pid_killed" == 0 ]]; then
          sleep 0.5
        fi
      done
    fi
  }

  __bctilstest_buildgolang () {
    time_start=$(($(date +%s%N) / 1000000))
    if just build; then
      BCTILS_COMPILE_TIME=$((($(date +%s%N) / 1000000) - time_start))
      return 0
    else
      echo -e "${RED}ERROR: bctils FAIL to build${NC}"
      return 1
    fi
  }

  export BCTILS_COMPILE_TIME=0
  __bctilstest_kill_other_procs
  __bctilstest_buildgolang && __bctils_test_run_test_script
  inotifywait -q -m -r -e close_write,create,delete "$proj_dir" \
    --exclude "$proj_dir/((compile|build|.git|.idea)/.*|.*\.log)" | \
    inotifywait_debounce 150 |
    while read -r watch_file events dir_file; do
      __bctilstest_inotify_loop "$watch_file" "$events" "$dir_file"
    done
fi
