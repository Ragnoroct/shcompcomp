benchmark:
  TEST_BENCHMARK=1 ./bctils-tests.sh

test:
  ./bctils-tests.sh

test-generators test_name="":
  #!/usr/bin/env bash
  proj_dir="$PWD"

  red="$(tput setaf 1)"
  green="$(tput setaf 2)"
  yellow="$(tput setaf 3)"
  magenta="$(tput setaf 5)"
  cyan="$(tput setaf 6)"
  reset="$(printf "%b" "\033[0m")"

  log () { echo -e "[$(date '+%T.%3N')] $*" >> ~/mybash.log; }
  run_tests () {
    test_call=(go test -v "./generators")
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

  run_tests
  inotifywait -q -m -r -e close_write,create,delete "$proj_dir" \
  --exclude "$proj_dir\/((compile|build|.git|.idea)\/?|.*(\.lock|~|\.log))$" | \
  inotifywait_debounce 100 | \
  while read -r dir events dir_file; do
    run_tests
  done



  run_tests () {
    test_call=(go test -v "./generators")
    if [[ -n "{{test_name}}" ]]; then
      test_call+=(-run {{test_name}})
    fi
    echo "${test_call[*]}"
    "${test_call[@]}"
    echo -e "${magenta}DONE${reset}: $(date '+%T.%3N')"
  }
logs:
  #!/usr/bin/env bash
  red="$(tput setaf 1)"
  green="$(tput setaf 2)"
  yellow="$(tput setaf 3)"
  blue="$(tput setaf 4)"
  magenta="$(tput setaf 5)"
  cyan="$(tput setaf 6)"
  reset="$(printf "%b" "\033[0m")"
  tail -f ~/mybash.log \
  | sed -u s/"go compilation error"/"$red&$reset"/i \
  | sed -u s/"RUNNING TESTS"/"$magenta&$reset"/i \
  | sed -u s/"RESULTS FAIL"/"$red&$reset"/i \
  | sed -u s/"FAIL"/"$red&$reset"/i \
  | sed -u s/"=== TEST .*"/"$cyan&$reset"/i \
  | sed -u s/PASS/"$green&$reset"/i \
  | sed -u s/WARNING/"$yellow&$reset"/i \
  ;

@build:
  mkdir -p "build"
  go build -o "build/bctils"
