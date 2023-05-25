set positional-arguments

watchfile file:
    #!/usr/bin/env bash

    clear && bat "$1"
    inotifywait -q -m -e close_write,modify,create,delete "$(dirname "$1")" | \
    while read -r file events dir_file; do
        if [[ "$(basename "$1")" == "$dir_file" \
            && "$(stat --printf="%s" "$1")" -gt 0 \
        ]]; then
            clear && bat "$1"
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
