benchmark:
  TEST_BENCHMARK=1 ./bctils-tests.sh

test:
  ./bctils-tests.sh

test-generators:
  #!/usr/bin/env bash
  proj_dir="$PWD"

  # $1: debounce_by time
  # $2: file_prefix for lockfiles
  @debounce () {
    debounce_by="$1"; shift
    file_prefix="$1"; shift
    _d_last_pid="$file_prefix-last-pid"
    _d_exec_pid="$file_prefix-exec-pid"
    _d_exec_que="$file_prefix-exec-queue"

    __debounce_pid_exit () {
      while true; do
        if [[ ! -f "$_d_exec_pid" && -z "$(cat "$_d_exec_que" 2>/dev/null)" ]]; then
          break
        fi
        sleep .1
      done
    }
    trap '__debounce_pid_exit' EXIT

    (
      echo "$BASHPID" > "$_d_last_pid"
      sleep "$debounce_by"
      if [[ "$BASHPID" == "$(cat "$_d_last_pid")" ]]; then
        echo "$BASHPID" > "$_d_exec_que"
        while true; do
          if [[ ! -f "$_d_exec_pid" && "$BASHPID" == "$(head -n 1 "$_d_exec_que")" ]]; then
            sed -i '1d' "$_d_exec_que"
            break
          fi
          sleep 0.1
        done

        echo "$BASHPID" > "$_d_exec_pid"
        "$@"
        rm -rf "$_d_exec_pid"
      fi
    ) &
  }

  MAGENTA='\033[0;35m'
  NC='\033[0m'
  run_tests () {
    go test -v "./generators"
    echo -e "${MAGENTA}DONE${NC}: $(date '+%T.%3N')"
  }

  run_tests
  inotifywait -q -m -r -e close_write,create,delete "$proj_dir" \
  --exclude "$proj_dir\/((compile|build|.git|.idea)\/?|.*(\.lock|~|\.log))$" |
  while read -r dir events dir_file; do
    @debounce "0.1" "/tmp/bctils-autogen-test" run_tests
  done

logs:
  tail -f ~/mybash.log

@build:
  mkdir -p "build"
  go build -o "build/bctils"
