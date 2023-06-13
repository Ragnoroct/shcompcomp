test test_name="":
  #!/usr/bin/env bash
  title="test"
  proj_dir="$PWD"

  echo -ne "\e[22;0t"; printf "\e]0;%s %s\007" "$title" "init"; trap 'echo -ne "\e[23;0t"' EXIT

  red="$(tput setaf 1)"
  green="$(tput setaf 2)"
  yellow="$(tput setaf 3)"
  magenta="$(tput setaf 5)"
  cyan="$(tput setaf 6)"
  reset="$(printf "%b" "\033[0m")"

  log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/bashscript.log; }
  inotify_action () {
    path_rel="$(realpath --relative-to="$proj_dir" "$1")"
    [[ "${path_rel: -1}" == "~" ]] && return
    [[ "$path_rel" =~ ^(.git/|.idea/|build/|scratch/|archive/|compile/) ]] && return
    [[ "$path_rel" =~ (.lock|.log)$ ]] && return

    test_call=(go test ./...)
    if [[ -n "{{test_name}}" ]]; then
      test_call+=(-run {{test_name}})
    fi
    echo "${test_call[*]}"


    profile_start_tests="$EPOCHREALTIME"
    "${test_call[@]}" 2>&1 \
      | clean_stack.pl \
      | tee /tmp/bashcompletils-test-results \
      | sed s/"FAIL"/"$red&$reset"/i \
      | sed s/"PASS\|ok"/"$green&$reset"/i \
      | sed s/"WARNING"/"$yellow&$reset"/i \
      ;
    status_test="${PIPESTATUS[0]}"
    status_build="${PIPESTATUS[1]}"
    profile_end_tests="$EPOCHREALTIME"
    profile_tests="$(bc <<< "(($profile_end_tests-$profile_start_tests)*10000)/10")"

    if [[ "$status_build" == 2 ]]; then
      log "[tests] compile error"
      printf "\e]0;%s %s %3dms\007" "$title" "cerr" "$profile_tests"
    elif [[ "$status_test" != 0 ]]; then
      log "[tests] fail tests ${profile_tests}ms\n$(cat "/tmp/bashcompletils-test-results" | grep '\--- FAIL:')"
      printf "\e]0;%s %s %3dms\007" "$title" "fail" "$profile_tests"
    else
      log "[tests] pass ${profile_tests}ms"
      printf "\e]0;%s %s %3dms\007" "$title" "pass" "$profile_tests"
    fi

    echo -e "${magenta}DONE${reset}: $(date '+%T.%3N')"
  }

  inotify_action "dummy.go"
  inotifywait -q -m -r -e close_write,create,delete "$proj_dir" | \
  while read -r dir events filename; do
    inotify_action "$dir$filename"
  done

logs:
  #!/usr/bin/env bash
  title="logs"
  log_file="$HOME/bashscript.log"

  echo -ne "\e[22;0t"; echo -ne "\e]0;$title\007"; trap 'echo -ne "\e[23;0t"' EXIT

  echo "tailing: $log_file"
  red="$(tput setaf 1)"
  green="$(tput setaf 2)"
  yellow="$(tput setaf 3)"
  blue="$(tput setaf 4)"
  magenta="$(tput setaf 5)"
  cyan="$(tput setaf 6)"
  reset="$(printf "%b" "\033[0m")"
  tail -f "$log_file" \
  | sed -u s/"go compilation error"/"$red&$reset"/i \
  | sed -u s/"RUNNING TESTS"/"$magenta&$reset"/i \
  | sed -u s/"RESULTS FAIL"/"$red&$reset"/i \
  | sed -u s/"\(FAIL\)\(.*\)\(test.*\)"/"$red\1$reset\2$cyan\3$reset"/i \
  | sed -u s/"FAIL"/"$red&$reset"/i \
  | sed -u s/"TEST:.*"/"$cyan&$reset"/i \
  | sed -u s/PASS/"$green&$reset"/i \
  | sed -u s/WARNING/"$yellow&$reset"/i \
  ;

build-watch:
    #!/usr/bin/env bash
    title="build"
    proj_dir="$PWD"

    echo -ne "\e[22;0t"; echo -ne "\e]0;$title\007"; trap 'echo -ne "\e[23;0t"' EXIT

    red="$(tput setaf 1)"
    green="$(tput setaf 2)"
    yellow="$(tput setaf 3)"
    magenta="$(tput setaf 5)"
    cyan="$(tput setaf 6)"
    reset="$(printf "%b" "\033[0m")"

    exec 3>&1

    log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/bashscript.log; }
    inotify_action () {
      path_rel="$(realpath --relative-to="$proj_dir" "$1")"
      [[ "${path_rel: -1}" == "~" ]] && return
      [[ "$path_rel" =~ ^(.git/|.idea/|build/|scratch/|archive/|compile/) ]] && return
      [[ "$path_rel" =~ (.lock|.log)$ ]] && return

      if [[ "$1" =~ ".go"$ && ! "$1" =~ "_test.go"$ && ! "$1" =~ "testutil.go"$ ]]; then
        profile_start="$EPOCHREALTIME"
        if ! build_out="$(go build -o "build/shcomp2" 2>&1 1>&3)"; then
          log "[build] compile error:\n$build_out"
        else
          log "[build] compile success"
        fi
        profile_end="$EPOCHREALTIME"
        compile_ms="$(bc <<< "scale=2;(($profile_end-$profile_start)*10000)/10")"
        printf "\e]0;%s %3dms\007" "$title" "${compile_ms%%.*}"
        echo "compiled: ${compile_ms}ms - $1"
      fi
    }

    inotify_action "dummy.go"
    inotifywait -q -m -r -e close_write,create,delete "$proj_dir" \
    --exclude "$proj_dir\/((compile|build|scratch.*|archive|\..*).*|.*(\.lock|~|\.log))$" \
    | inotifywait_debounce 100 | \
    while read -r dir events filename; do
      inotify_action "$dir$filename"
    done

@build:
  mkdir -p "build"
  go build -o "build/shcomp2"
