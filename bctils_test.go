package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

var chanExitCmd chan byte

func TestMain(m *testing.M) {
	defer setupLogger()()

	chanExitCmd = make(chan byte)
	defer close(chanExitCmd)

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	go func() {
		for {
			log.Printf("=== RUNNING TESTS")
			code := m.Run()
			if code == 1 {
				log.Printf("=== RESULTS FAIL : %s", time.Now().Format("3:4:5.000"))
			} else {
				log.Printf("=== RESULTS PASS : %s", time.Now().Format("3:4:5.000"))
			}
			os.Exit(code)
		}
	}()

	// cleanup
	sig := <-cancelChan
	fmt.Printf("signal handled: %v\n", sig.(syscall.Signal))
	chanExitCmd <- byte(0)
	//fmt.Printf("done\n")
	//err := syscall.Kill(syscall.Getpid(), sig.(syscall.Signal))
	//if err != nil {
	//	return
	//}
}

func TestSub(t *testing.T) {
	ctx := TestContext{t: t}
	ctx.Run("simple output", func(ctx TestContext) {
		var stdin bytes.Buffer
		stdin.Write([]byte(`
			cfg cli_name=bobman
			opt "-h"
			opt "--help"
		`))
		result := parseOperationsStdin(&stdin)
		ctx.ExpectEqualLines("hunter2", result, "")
	})
}

func TestEndToEnd(t *testing.T) {
	shellCode := dedent(`
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

		while read -r line; do
			echo "read input: $line"
		done
	`)

	cmd := exec.Command("/usr/bin/bash", "-c", shellCode)
	stdout, err := cmd.StdoutPipe()
	check(err)
	stdin, err := cmd.StdinPipe()
	check(err)
	err = cmd.Start()
	check(err)
	cmd.Process.Release()

	go func() {
		io.WriteString(stdin, "blab\n")
		io.WriteString(stdin, "blob\n")
		io.WriteString(stdin, "booo\n")
	}()

	output := make(chan string)
	defer close(output)
	go ReadOutput(output, stdout)
	stdout.Close()

	for o := range output {
		fmt.Printf(o)
	}

	go func() {
		<-chanExitCmd
		fmt.Printf("killing it\n")
		err := cmd.Process.Signal(syscall.SIGKILL)
		fmt.Printf("waiting\n")
		_, err = cmd.Process.Wait()
		close(output)
		if err != nil {
			return
		}
		fmt.Printf("killed\n")
		if err != nil {
			return
		}
	}()

	close(output)

	//_, err = stdin.Write([]byte("asdf"))
	//check(err)
	//scanner := bufio.NewScanner(stdout)
	//for scanner.Scan() {
	//	m := scanner.Text()
	//	fmt.Println(m)
	//}
	//err = cmd.Wait()
	//if err != nil {
	//	return
	//}
}

func ReadOutput(output chan string, rc io.ReadCloser) {
	r := bufio.NewReader(rc)
	for {
		x, _ := r.ReadString('\n')
		output <- string(x)
	}
}
