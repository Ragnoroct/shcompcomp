package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

var bashOutBuffer []byte
var chanBashDone chan string
var stdin *io.WriteCloser
var mutex sync.Mutex

func main() {
	shellCode := getTestShellCode()
	chanBashDone = make(chan string)
	startProcess(shellCode)
	completeCmd("simple asdf ")
}

func completeCmd(cmdString string) {
	mutex.Lock()
	defer mutex.Unlock()

	_, _ = io.WriteString(*stdin, cmdString+"\n")
	out := <-chanBashDone
	fmt.Printf("output: %s", out)
}

func startProcess(shellCode string) {
	var err error
	subProcess := exec.Command("bash", "-c", shellCode)
	subProcessStdin, err := subProcess.StdinPipe()
	check(err)
	stdout, err := subProcess.StdoutPipe()
	check(err)

	err = subProcess.Start()
	check(err)

	stdin = &subProcessStdin

	go func() {
		var err error
		var n int
		buff := make([]byte, 256)
		for err == nil {
			n, err = stdout.Read(buff)
			for i := 0; i < n; i++ {
				if buff[i] == '\x00' {
					out := string(bashOutBuffer)
					bashOutBuffer = []byte{}
					chanBashDone <- out
				} else {
					bashOutBuffer = append(bashOutBuffer, buff[i])
				}
			}
		}
	}()
}

func getTestShellCode() string {
	return `
		very_simple () {
			COMPREPLY=("c1" "c2" "c3")
		}

		complete -F very_simple simple

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
			__bctilstest_compopt_current_cmd=""
			unset compopt

			printf '%s\n' "${COMPREPLY[*]}"
		}

		while read -r line; do
			complete_cmd_str "$line"
			printf '\0'
		done
	`
}

func dedent(str string) string {
	mixingSpacesAndTabs := false
	if str[0] == '\n' {
		str = str[1:]
	}
	lines := strings.Split(str, "\n")
	minIndent := -1
	for _, line := range lines {
		for i, c := range line {
			if c == ' ' {
				mixingSpacesAndTabs = true
				//panic("cannot handle mixing spaces with tab")
				continue
			} else if c != '\t' {
				if minIndent == -1 || i < minIndent {
					minIndent = i
				}
				break
			}
		}
	}

	if minIndent == 0 {
		return strings.TrimSpace(str) + "\n"
	} else if mixingSpacesAndTabs {
		panic("cannot handle mixing spaces with tab")
	}

	indentStr := strings.Repeat("\t", minIndent)
	for i := range lines {
		newLine, _ := strings.CutPrefix(lines[i], indentStr)
		lines[i] = newLine
	}

	if strings.Trim(lines[len(lines)-1], " \t\n") == "" {
		lines = lines[0 : len(lines)-1]
	}

	return strings.Join(lines, "\n") + "\n"
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
