benchmark:
    TEST_BENCHMARK=1 ./tests/bctils-tests.sh

test:
    ./bctils-tests.sh

logs:
  tail -f ~/mybash.log

@build:
    mkdir -p "build"
    go build -o "build/bctils"
