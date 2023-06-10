package testutil

import (
	"bctils/pkg/lib"
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var mutex sync.Mutex
var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

var loggerCleanup func()

type BaseSuite struct {
	suite.Suite
	tmpdir string
}

func (suite *BaseSuite) SetupSuite() {
	loggerCleanup = lib.SetupLogger()
	log.Printf("RUNNING TESTS")
}

func (suite *BaseSuite) SetupTest() {
	suite.tmpdir = ""
}

func (suite *BaseSuite) SetupSubTest() {
	suite.tmpdir = ""
}

func (suite *BaseSuite) TearDownSuite() {
	defer loggerCleanup()
}

func (suite *BaseSuite) RequireComplete(shell, cmdStr string, expected string) {
	suite.T().Helper()
	ExpectComplete(suite.T(), shell, cmdStr, expected)
}

func (suite *BaseSuite) RequireCompleteFile(file, cmdStr string, expected string) {
	suite.T().Helper()
	ExpectCompleteFile(suite.T(), file, cmdStr, expected)
}

func (suite *BaseSuite) CreateFile(filename string, contents string, rest ...any) (filepath string) {
	if suite.tmpdir == "" {
		suite.tmpdir = suite.T().TempDir()
	}

	permission := 0644

	if len(rest) > 0 {
		switch v := rest[0].(type) {
		case int:
			permission = v
		}
	}

	filepath = path.Join(suite.tmpdir, filename)
	contents = lib.Dedent(contents)
	err := os.WriteFile(filepath, []byte(contents), fs.FileMode(permission))
	if err != nil {
		panic(err)
	}

	return filepath
}

func (suite *BaseSuite) TempDir() (filepath string) {
	if suite.tmpdir == "" {
		suite.tmpdir = suite.T().TempDir()
	}
	return suite.tmpdir
}

type CompleterProcess struct {
	stdin      *io.WriteCloser
	chanOut    chan string
	chanStderr chan string
	mutex      sync.Mutex
}
type completerProcesses struct {
	processMap map[string]*CompleterProcess
}

var processes completerProcesses

func CompleterFile(shellFile string) *CompleterProcess {
	if processes.processMap == nil {
		processes.processMap = make(map[string]*CompleterProcess)
	}
	processKey := "file:" + shellFile
	if process, ok := processes.processMap[processKey]; ok {
		return process
	} else {
		chanOut, chanStderr, stdin := startProcess("", shellFile)
		process = &CompleterProcess{
			chanOut:    chanOut,
			chanStderr: chanStderr,
			stdin:      stdin,
			mutex:      sync.Mutex{},
		}

		processes.processMap[processKey] = process
		return process
	}
}

func Completer(completeShell string) *CompleterProcess {
	if processes.processMap == nil {
		processes.processMap = make(map[string]*CompleterProcess)
	}
	completeShell = lib.Dedent(completeShell)
	if process, ok := processes.processMap[completeShell]; ok {
		return process
	} else {
		chanOut, chanStderr, stdin := startProcess(completeShell, "")
		process = &CompleterProcess{
			chanOut:    chanOut,
			chanStderr: chanStderr,
			stdin:      stdin,
			mutex:      sync.Mutex{},
		}

		processes.processMap[completeShell] = process
		return process
	}
}

func (p *CompleterProcess) Complete(cmdStr string) string {
	mutex.Lock()
	defer mutex.Unlock()

	_, _ = io.WriteString(*p.stdin, cmdStr+"\n")
	out := <-p.chanOut
	outStderr := <-p.chanStderr
	if outStderr != "" {
		lines := strings.Split(outStderr, "\n")
		nonPlus := false
		for _, line := range lines {
			if line != "" && !strings.HasPrefix(line, "+") {
				nonPlus = true
			}
		}
		if nonPlus {
			panic("error : stderr : " + outStderr)
		}
	}
	return strings.TrimRight(out, " \t\n")
}

func ParseOperationsStdinHelper(operations string, values ...any) string {
	var stdin bytes.Buffer
	operations = fmt.Sprintf(operations, values...)
	operations = lib.Dedent(operations)
	stdin.Write([]byte(operations))
	return lib.ParseOperationsStdin(&stdin)
}

func ExpectCompleteFile(t *testing.T, shellFile string, cmdStr string, expected string) {
	t.Helper()
	completer := CompleterFile(shellFile)
	actual := strings.TrimRight(completer.Complete(cmdStr), "\n \t")
	if actual != expected {
		testname := t.Name()
		pwd, _ := os.Getwd()
		testname = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(testname, "_")
		testname = regexp.MustCompile(`_+`).ReplaceAllString(testname, "_")
		testname = strings.ToLower(testname)
		testname += ".bash"
		compilePath := path.Join(pwd, "compile/"+testname)
		content, _ := os.ReadFile(shellFile)
		_ = os.WriteFile(compilePath, content, 0644)

		t.Fatalf(
			"\n%s\n"+
				"     cmd: '%s'\n"+
				"  actual: '%s'\n"+
				"expected: '%s'\n",
			"./"+strings.TrimPrefix(compilePath, pwd+"/")+":0:",
			cmdStr,
			actual,
			expected,
		)
	}
}

func ExpectComplete(t require.TestingT, shell string, cmdStr string, expected string) {
	completer := Completer(shell)
	actual := strings.TrimRight(completer.Complete(cmdStr), "\n \t")
	if actual != expected {
		testname := "unknowntestname"
		if n, ok := t.(interface {
			Name() string
		}); ok {
			testname = n.Name()
		}

		testname = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(testname, "_")
		testname = regexp.MustCompile(`_+`).ReplaceAllString(testname, "_")
		testname = strings.ToLower(testname)
		testname += ".bash"
		compilePath := path.Join(basepath, "../../compile/"+testname)
		_ = os.WriteFile(compilePath, []byte(shell), 0644)

		if h, ok := t.(interface{ Helper() }); ok {
			h.Helper()
		}
		require.Equalf(t, expected, actual, "completion does not match\n"+compilePath+":0")
	}
}

type StdoutMock struct {
	pipeWriter *os.File
	pipeReader *os.File
}

func (stdout StdoutMock) GetString() string {
	_ = stdout.pipeWriter.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, stdout.pipeReader)
	_ = stdout.pipeReader.Close()
	return buf.String()
}

func startProcess(shellCode string, filename string) (chan string, chan string, *io.WriteCloser) {
	var err error
	var bashOutBuffer []byte
	var stderrBuffer []byte
	var chanOut = make(chan string)
	var chanStderr = make(chan string)

	var shellCodePrefix string
	if shellCode != "" {
		// inline completion code
		shellCodePrefix = lib.Dedent(shellCode)
	} else {
		// external completion code (for testing auto reload stuff)
		shellCodePrefix = lib.Dedent(fmt.Sprintf(`
			source "%s"
		`, filename))
	}

	cwd, _ := os.Getwd()
	buildPath := path.Join(cwd, "build")

	shellCode = shellCodePrefix + "\n" + lib.Dedent(`
		export PATH="$PATH:`+buildPath+`"
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

		if [ -f /usr/share/bash-completion/bash_completion ]; then
			source /usr/share/bash-completion/bash_completion
		elif [ -f /etc/bash_completion ]; then
			source /etc/bash_completion
		fi

		while IFS= read -r line; do
			complete_cmd_str "$line"
			printf '\0'
		done
	`)

	proc := exec.Command("bash", "-c", shellCode)
	stdin, err := proc.StdinPipe()
	check(err)
	stdout, err := proc.StdoutPipe()
	check(err)
	stderr, err := proc.StderrPipe()
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
					outstderr := string(stderrBuffer)
					bashOutBuffer = []byte{}
					stderrBuffer = []byte{}
					chanOut <- out
					chanStderr <- outstderr
				} else {
					bashOutBuffer = append(bashOutBuffer, buff[i])
				}
			}
		}
	}()

	go func() {
		var err error
		var n int
		buff := make([]byte, 256)
		for err == nil {
			n, err = stderr.Read(buff)
			for i := 0; i < n; i++ {
				stderrBuffer = append(stderrBuffer, buff[i])
			}
		}
	}()

	return chanOut, chanStderr, &stdin
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
