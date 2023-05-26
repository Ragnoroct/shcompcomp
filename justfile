set positional-arguments

watchfile file head=(""):
    #!/usr/bin/env bash

    if [[ "$2" == "head" || "$2" == "1" ]]; then
      head_out=1
    else
      head_out=0
    fi

    terminal_row_count="$(tput lines)"
    if [[ "$head_out" == 1 ]]; then
      clear && bat "$1" --paging=never --color=always | head -n "$terminal_row_count"
    else
      clear && bat "$1" --paging=never --color=always
    fi
    inotifywait -q -m -e close_write,modify,create,delete "$(dirname "$1")" --exclude ".*.log" | \
    while read -r file events dir_file; do
        if [[ "$(basename "$1")" == "$dir_file" \
            && "$(stat --printf="%s" "$1")" -gt 0 \
        ]]; then
            if [[ "$head_out" == 1 ]]; then
              terminal_row_count="$(tput lines)"
              clear && bat "$1" --paging=never --color=always | head -n "$terminal_row_count"
            else
              clear && bat "$1" --paging=never --color=always
            fi
        fi
    done

benchmark:
    TEST_BENCHMARK=1 ./tests/bctils-tests.sh

test:
    ./bctils-tests.sh

logs:
  tail -f ~/mybash.log

@build:
    mkdir -p "build"
    go build -o "build/bctils"
