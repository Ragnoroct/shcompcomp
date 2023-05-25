#!/usr/bin/env bash

# https://stackoverflow.com/a/9505024/2276637
script_dir=$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)
proj_dir="$(realpath "$script_dir/..")"
curr_dir="$PWD"
CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

cd "$proj_dir" || exit
source "/usr/share/bash-completion/bash_completion"
source "./bctils-lib.sh" || { echo -e "${RED}ERROR: sourcing bctils-lib.sh failed$NC"; exit 1; }
cd "$curr_dir" || exit

test_case="$*"
export BCTILS_COMPILE_DIR="$script_dir/../compile"

# todo: run test suites in parallel but still print in queued order
# todo: show SUCCESS OR FAIL if anything fails
# todo: re-run last failed tests until nothing then everything (maybe bad idea)
# todo: print failures at bottom of script so they are seen (somehow with suite)
# todo: add line number to pass/fail or just fail
run_tests () {
    if [[ "$TEST_BENCHMARK" == 1 ]]; then
        run_benchmarks
        return
    fi
    
    all_result="${GREEN}SUCCESS${NC}"
    fail_count_str=""
    fail_count=0
    time_start=$(($(date +%s%N)/1000000))

    current_suite "simple"
    bctils_cli_register "examplecli"
    bctils_cli_add "examplecli" opt "-h"
    bctils_cli_add "examplecli" opt "--help"
    __bctils_dump "examplecli"
#    bctils_v2_cli_compile       "examplecli" --source
#    expect_complete_compreply   "examplecli " "-h --help"
#
#    bctils_v2_cli_register "examplecli"
#    bctils_v2_cli_add_argument "examplecli" "-h"
#    bctils_v2_cli_compile "examplecli" "/tmp/bctils_complete_test.sh"
#    expect_cmd "compile to target file" test -f "/tmp/bctils_complete_test.sh"
#
#    current_suite "positionals with choices"
#    bctils_v2_cli_register      "examplecli2"
#    bctils_v2_cli_add_argument  "examplecli2" --choices="c1 c2 c3"
#    bctils_v2_cli_add_argument  "examplecli2" --choices="c4 c5 c6"
#    bctils_v2_cli_compile       "examplecli2" --source
#    expect_complete_compreply   "examplecli2 " "c1 c2 c3"
#    expect_complete_compreply   "examplecli2 c" "c1 c2 c3"
#    expect_complete_compreply   "examplecli2 d" ""
#    expect_complete_compreply   "examplecli2 c2 " "c4 c5 c6"
#    expect_complete_compreply   "examplecli2 c2 d" ""
#
#    current_suite "positionals with choices and optionals"
#    bctils_v2_cli_register      "examplecli3"
#    bctils_v2_cli_add_argument  "examplecli3" --choices="c1 c2 c3"
#    bctils_v2_cli_add_argument  "examplecli3" "-h"
#    bctils_v2_cli_add_argument  "examplecli3" "--help"
#    bctils_v2_cli_compile       "examplecli3" --source
#    expect_complete_compreply   "examplecli3 "          "c1 c2 c3 -h --help"
#    expect_complete_compreply   "examplecli3 c"         "c1 c2 c3"
#    expect_complete_compreply   "examplecli3 -"         "-h --help"
#    expect_complete_compreply   "examplecli3 c3 "       "-h --help"
#    expect_complete_compreply   "examplecli3 --help "   "c1 c2 c3 -h"
#    expect_complete_compreply   "examplecli3 -h "       "c1 c2 c3 --help"
#
#    current_suite "simple subparsers"
#    bctils_v2_cli_register      "examplecli4"
#    bctils_v2_cli_add_argument  "examplecli4" "-h"
#    bctils_v2_cli_add_argument  "examplecli4" "--help"
#    bctils_v2_cli_add_argument  "examplecli4" -p="sub-cmd" --choices="c1 c2 c3"
#    bctils_v2_cli_add_argument  "examplecli4" -p="sub-cmd" "--help"
#    bctils_v2_cli_add_argument  "examplecli4" -p="sub-cmd" "--print"
#    bctils_v2_cli_compile       "examplecli4" --source
#    expect_complete_compreply   "examplecli4 " "-h --help"
#    expect_complete_compreply   "examplecli4 sub-cmd " "c1 c2 c3 --help --print"
#
#    current_suite "simple options with arguments like --opt val"
#    bctils_v2_cli_register      "examplecli5"
#    bctils_v2_cli_add_argument  "examplecli5" "--key" --choices="val1 val2"
#    bctils_v2_cli_compile       "examplecli5" --source
#    expect_complete_compreply   "examplecli5 " "--key"
#    expect_complete_compreply   "examplecli5 --key " "val1 val2"

    current_suite "complete option value like --opt=value"
    current_suite "add flag to auto add = if only one arg option left and it requires an argument"
    current_suite "complete single -s type options like -f filepath"
    current_suite "complete single -s type options like -ffilepath (if that makes sense)"

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
    

    
    echo -e "done: $((($(date +%s%N)/1000000)-time_start))ms (${all_result}${fail_count_str}) $(date '+%T.%3N')"

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

run_benchmarks () {
    __bctils_controlgroup_completer_1 () {
        # shellcheck disable=SC2034
        local cword_index previous_word words current_word
        _get_comp_words_by_ref -n = -n @ -n : -w words -i cword_index -p previous_word -c current_word
        mapfile -t COMPREPLY < <(compgen -W "c1 d2" -- "$current_word")
    }

    benchmark_results () {
        # benchmark_control_1
        local name="$1"
        local result="$2"
        local time_total="$3"
        local iterations="$4"
        printf "%s: '%s' average=%sms total=%sms (%s iterations)\n" \
            "$name" \
            "$result" \
            "$(bc <<< "scale=2; $time_total/$iterations")" \
            "$time_total" \
            "$iterations"
    }

    local iterations=${ITER:-1000}

    # setup testbenchmarkcontrol1
    complete -F __bctils_controlgroup_completer_1 "benchmark_control_1"
    current_suite "benchmark testing completions" && print_suite
    # setup testbenchmark1
    bctils_v2_cli_register      "benchmark_1"
    bctils_v2_cli_add_argument  "benchmark_1" "--key" --choices="val1 val2"
    bctils_v2_cli_compile       "benchmark_1" --source
    
    # benchmark_control_1
    time_start=$(($(date +%s%N)/1000000))
    i=-1
    while true; do
        i=$((i+1))
        if [[ "$i" -gt "$iterations" ]]; then break; fi
        complete_cmd_str "benchmark_control_1 "
    done
    rslt_benchmark_control_1="$complete_cmd_str_result"
    time_benchmark_control_1=$((($(date +%s%N)/1000000)-time_start))

    # benchmark_1
    time_start=$(($(date +%s%N)/1000000))
    i=-1
    while true; do
        i=$((i+1))
        if [[ "$i" -gt "$iterations" ]]; then break; fi
        complete_cmd_str "benchmark_1 --ke"
    done
    rslt_benchmark_1="$complete_cmd_str_result"
    time_benchmark_1=$((($(date +%s%N)/1000000)-time_start))

    # benchmark compilation
    time_start=$(($(date +%s%N)/1000000))
    local iter=-1    # todo: fix this. while i it's constantly reset to 0
    while true; do
        iter=$((iter+1))
        if [[ "$iter" -gt "$iterations" ]]; then break; fi
        bctils_v2_cli_register      "benchmark_compile_1"
        bctils_v2_cli_add_argument  "benchmark_compile_1" "--key" --choices="val1 val2"
        bctils_v2_cli_compile       "benchmark_compile_1" --source
    done
    time_benchmark_compile_1=$((($(date +%s%N)/1000000)-time_start))

    # benchmark compilation register
    time_start=$(($(date +%s%N)/1000000))
    local iter=-1    # todo: fix this. while i it's constantly reset to 0
    while true; do
        iter=$((iter+1))
        if [[ "$iter" -gt "$iterations" ]]; then break; fi
        bctils_v2_cli_register      "benchmark_compilereg_1"
    done
    time_benchmark_compilereg_1=$((($(date +%s%N)/1000000)-time_start))

    # benchmark compilation add argument
    bctils_v2_cli_register      "benchmark_compile_addarg_1"
    time_start=$(($(date +%s%N)/1000000))
    local iter=-1    # todo: fix this. while i it's constantly reset to 0
    while true; do
        iter=$((iter+1))
        if [[ "$iter" -gt "$iterations" ]]; then break; fi
        bctils_v2_cli_add_argument  "benchmark_compile_addarg_1" "--key" --choices="val1 val2"
    done
    time_benchmark_compile_addarg_1=$((($(date +%s%N)/1000000)-time_start))

    # benchmark compilation compile + source

    local time_total=0
    local iter=-1    # todo: fix this. while i it's constantly reset to 0
    while true; do
        iter=$((iter+1))
        if [[ "$iter" -gt "$iterations" ]]; then break; fi
        bctils_v2_cli_register      "benchmark_compile_and_source_1"
        bctils_v2_cli_add_argument  "benchmark_compile_and_source_1" "--key" --choices="val1 val2"
        time_start=$(($(date +%s%N)/1000000))
        bctils_v2_cli_compile       "benchmark_compile_and_source_1" --source
        time_total=$((time_total+(($(date +%s%N)/1000000)-time_start)))
    done
    time_benchmark_compile_and_source_1="$time_total"

    benchmark_results "benchmark_control_1" "$rslt_benchmark_control_1" "$time_benchmark_control_1" "$iterations"
    benchmark_results "benchmark_1" "$rslt_benchmark_1" "$time_benchmark_1" "$iterations"
    benchmark_results "benchmark_compile_1" "" "$time_benchmark_compile_1" "$iterations"
    benchmark_results "benchmark_compilereg_1" "" "$time_benchmark_compilereg_1" "$iterations"
    benchmark_results "benchmark_compile_addarg_1" "" "$time_benchmark_compile_addarg_1" "$iterations"
    benchmark_results "benchmark_compile_and_source_1" "" "$time_benchmark_compile_and_source_1" "$iterations"
}

expect_cmd () {
    local msg="$1"; shift
    local cmd="$1"; shift
    if "$cmd" "$@"; then
        pass_test "$msg"
    else
        fail_test "$msg"
    fi
}

current_suite_printed=0
current_suite_name=""
current_suite () {
    current_suite_name="${CYAN}====== $1${NC}"
    current_suite_printed=0
}

print_suite () {
    if [[ "$current_suite_printed" == 0 ]]; then
        echo -e "$current_suite_name"
        current_suite_printed=1
    fi
}

fail_test () {
    if [[ "$current_suite_printed" == 0 ]]; then
        echo -e "$current_suite_name"
        current_suite_printed=1
    fi
    echo -e "${RED}FAILED${NC}: '$1'"
    all_result="${RED}FAILED${NC}"
    fail_count=$((fail_count+1))
    fail_count_str=" $fail_count"
}

pass_test () {
    if [[ "$current_suite_printed" == 0 ]]; then
        echo -e "$current_suite_name"
        current_suite_printed=1
    fi
    echo -e "${GREEN}PASSED${NC}: '$1'"
}

complete_cmd_str () {
    local input_line="$1"
    declare -g complete_cmd_str_result

    IFS=', ' read -r -a comp_words <<< "$input_line"
    if [[ "$input_line" =~ " "$ ]]; then comp_words+=(""); fi

    cmd_name="${comp_words[0]}"
    COMP_LINE="$input_line"
    COMP_WORDS=("${comp_words[@]}")
    COMP_CWORD="$((${#comp_words[@]} - 1))"
    COMP_POINT="$(("${#input_line}" + 0))"

    "$(complete -p "$cmd_name" | sed "s/.*-F \\([^ ]*\\) .*/\\1/")" &> /tmp/bashcompletils.out
    complete_cmd_str_result="${COMPREPLY[*]}"
}

expect_complete_compreply () {
    test_name="$1"
    input_line="$1"
    expected_reply="$2"

    if [[ -n "$test_case" && "$test_case" != "$test_name" ]]; then
        return
    fi

    log "==== $test_name ===="

    IFS=', ' read -r -a comp_words <<< "$input_line"
    if [[ "$input_line" =~ " "$ ]]; then comp_words+=(""); fi   # append empty space if ends in space
    comp_line="$input_line"
    comp_point=$(("${#input_line}" + 0))
    comp_cword=$((${#comp_words[@]} - 1))
    cmd_name="${comp_words[0]}"

    COMP_LINE="$comp_line"
    COMP_WORDS=("${comp_words[@]}")
    COMP_CWORD="$comp_cword"
    COMP_POINT="$comp_point"
    "$(complete -p "$cmd_name" | sed "s/.*-F \\([^ ]*\\) .*/\\1/")" &> /tmp/bashcompletils.out
    output=$(cat /tmp/bashcompletils.out)
    
    # todo: diff with vimdiff/diff/git diff -U0 --word-diff --no-index -- foo bar | grep -v ^@@
    actual_reply="${COMPREPLY[*]}"
    if [[ "$actual_reply" != "$expected_reply" ]]; then
        fail_test "$test_name"
        echo "actual   : '$actual_reply'"
        echo "expected : '$expected_reply'"
        if [[ "${#output}" -gt 0 ]]; then
            echo "$output"
        fi
        log "==== FAILED ===="
    else
        pass_test "$test_name"
        log "==== PASSED ===="
    fi
}

_example_cli_branch_autocomplete () {
  # shellcheck disable=SC2034
  COMPREPLY=("develop" "custom/func/response")
  log "COMPREPLY: ${COMPREPLY[*]}"
}

if [[ "$TEST_RUN_MODE" == "RUN_TESTS_ONCE" ]]; then
  run_tests
else
  inotify_loop () {
    local watch_file="$1"
    local events="$2"
    local dir_file="$3"

    if [[ ! "$events" =~ .*"CLOSE_WRITE".* ]] || [[ "$dir_file" == "index.lock" || "$dir_file" =~ "~"$ ]]; then
      return
    fi

    if [[ -n "$dir_file" ]]; then
      echo "file change: $dir_file $events"
    fi

    if [[ "$events" =~ .*"CLOSE_WRITE".* && "$dir_file" == "bctils-lib.go" ]]; then
      echo "rebuilding golang binary..."
      just build
    fi

    echo "waiting for changes..."
    TEST_RUN_MODE="RUN_TESTS_ONCE" bash "$script_dir/tests.sh"
  }

  just build
  TEST_RUN_MODE="RUN_TESTS_ONCE" bash "$script_dir/tests.sh"
  inotifywait -q -m -r -e close_write,create,delete "$proj_dir" \
  --exclude "$proj_dir/(compile|build)/*/**" | \
  while read -r watch_file events dir_file; do
    inotify_loop "$watch_file" "$events" "$dir_file"
  done
fi


