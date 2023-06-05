package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

var mutex sync.Mutex

type completerProcess struct {
	stdin   *io.WriteCloser
	chanOut chan string
	mutex   sync.Mutex
}
type completerProcesses struct {
	processMap map[string]*completerProcess
}

var processes completerProcesses

func newCompleter(completeShell string) *completerProcess {
	if processes.processMap == nil {
		processes.processMap = make(map[string]*completerProcess)
	}
	completeShell = dedent(completeShell)
	if process, ok := processes.processMap[completeShell]; ok {
		return process
	} else {
		chanOut, stdin := startProcess(completeShell)
		process = &completerProcess{
			chanOut: chanOut,
			stdin:   stdin,
			mutex:   sync.Mutex{},
		}

		processes.processMap[completeShell] = process
		return process
	}
}

func (p *completerProcess) complete(cmdStr string) string {
	mutex.Lock()
	defer mutex.Unlock()

	_, _ = io.WriteString(*p.stdin, cmdStr+"\n")
	out := <-p.chanOut
	return strings.TrimRight(out, " \t\n")
}

func main() {
	completer := newCompleter(`
		very_simple () {
			COMPREPLY=("c1" "c2" "c4")
		}
		complete -F very_simple simple
	`)
	out := completer.complete("simple ")
	fmt.Printf("out: '%s'", out)
}

func startProcess(shellCode string) (chan string, *io.WriteCloser) {
	var err error
	var bashOutBuffer []byte
	var chanOut = make(chan string)

	shellCode = dedent(shellCode) + "\n" + dedent(`
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
	`)

	proc := exec.Command("bash", "-c", shellCode)
	stdin, err := proc.StdinPipe()
	check(err)
	stdout, err := proc.StdoutPipe()
	check(err)
	err = proc.Start()
	check(err)

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
					chanOut <- out
				} else {
					bashOutBuffer = append(bashOutBuffer, buff[i])
				}
			}
		}
	}()

	return chanOut, &stdin
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
