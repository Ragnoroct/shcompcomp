test test_name="":
  #!/usr/bin/env bash
  proj_dir="$PWD"

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
    "${test_call[@]}" \
    | sed s/FAIL/"$red&$reset"/i \
    | sed s/PASS/"$green&$reset"/i \
    | sed s/WARNING/"$yellow&$reset"/i \
    ;
    if [[ "${PIPESTATUS[0]}" == 2 ]]; then
      log "!!! go compilation error"
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
  log_file="$HOME/bashscript.log"
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
    proj_dir="$PWD"

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
        if ! build_out="$(go build -o "build/bctils" 2>&1 1>&3)"; then
          log "go compile FAIL:\n$build_out"
        else
          log "go compile PASS"
        fi
        profile_end="$EPOCHREALTIME"
        echo "compiled: $(bc <<< "scale=2;(($profile_end-$profile_start)*10000)/10")ms - $1"
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
  go build -o "build/bctils"
