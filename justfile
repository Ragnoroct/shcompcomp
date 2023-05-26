set positional-arguments

watchfile file:
    #!/usr/bin/env bash

    terminal_row_count="$(tput lines)"
    clear && bat "$1" --paging=never --color=always | head -n "$terminal_row_count"
    inotifywait -q -m -e close_write,modify,create,delete "$(dirname "$1")" --exclude ".*.log" | \
    while read -r file events dir_file; do
        if [[ "$(basename "$1")" == "$dir_file" \
            && "$(stat --printf="%s" "$1")" -gt 0 \
        ]]; then
            terminal_row_count="$(tput lines)"
            clear && bat "$1" --paging=never --color=always | head -n "$terminal_row_count"
        fi
    done

benchmark:
    TEST_BENCHMARK=1 ./tests/tests.sh

test:
    ./tests/tests.sh

logs:
  tail -f ~/mybash.log

@build:
    mkdir -p "build"
    go build -o "build/bctils"
