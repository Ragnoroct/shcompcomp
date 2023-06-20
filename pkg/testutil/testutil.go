package testutil

import (
	"bytes"
	_ "embed"
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
	"shcomp2/pkg/lib"
	"strings"
	"sync"
	"time"
)

//go:embed bash-bridge.sh
var bashBridgeShell string
var check = lib.Check

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

func (suite *BaseSuite) RequireCompleteFile(file, cmdStr string, expected string) {
	suite.T().Helper()
	shell := "source " + file
	suite.RequireComplete(shell, cmdStr, expected)
	suite.requireCompleteHelper(completeRequest{
		shell:      "source " + file,
		file:       file,
		testMethod: "bashfunc",
		cmdStr:     cmdStr,
	}, expected)
}

// RequireCompleteWithExpectTcl
// Use to test compopt logic where you need to know more than just what the function will return
// Can ensure that output code works correctly and sanely with readline completions
func (suite *BaseSuite) RequireCompleteWithExpectTcl(shell, cmdStr string, expected string) {
	suite.T().Helper()
	file := suite.CreateFile("expecttclsource.sh", shell)
	suite.requireCompleteHelper(completeRequest{
		shell:      "source " + file,
		file:       file,
		testMethod: "expecttcl",
		cmdStr:     cmdStr,
	}, expected)
}

func (suite *BaseSuite) RequireComplete(shell, cmdStr string, expected string) {
	suite.requireCompleteHelper(completeRequest{
		shell:      shell,
		testMethod: "bashfunc",
		cmdStr:     cmdStr,
	}, expected)
}

func (suite *BaseSuite) requireCompleteHelper(request completeRequest, expected string) {
	suite.T().Helper()
	t := suite.T()
	shell := request.shell
	writeCompiled := func() string {
		content := []byte(shell)
		if request.file != "" {
			content, _ = os.ReadFile(request.file)
		}

		testname := suite.T().Name()
		testname = regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(testname, "_")
		testname = regexp.MustCompile(`_+`).ReplaceAllString(testname, "_")
		testname = strings.ToLower(testname)
		testname += ".bash"
		compilePath := path.Join(basepath, "../../compile/"+testname)
		_ = os.WriteFile(compilePath, content, 0644)
		return compilePath
	}

	defer func() {
		if r := recover(); r != nil {
			writeCompiled()
			panic(r)
		}
	}()

	completer := completer(shell)
	response := completer.complete(request)
	if response.error != nil {
		require.FailNow(
			suite.T(),
			fmt.Sprintf(
				"completion failed : %v\n"+
					"stderr: %s\n"+
					"stdout: %s", response.error, response.stderr, response.stdout,
			))
	}
	actual := strings.TrimRight(response.stdout, "\n\t ")
	if actual != expected {
		compilePath := writeCompiled()
		home, _ := os.UserHomeDir()
		compilePath = strings.Replace(compilePath, home, "~", 1)

		pathPrefix := os.Getenv("CUST_PROTOCOL_PREFIX")
		if pathPrefix != "" {
			compilePath = pathPrefix + "/" + strings.TrimLeft(compilePath, "/")
		}

		t.Helper()
		require.Equalf(t, expected, actual, "completion does not match\n"+compilePath)
	}
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

func ParseOperations(operations string, values ...any) string {
	var stdin bytes.Buffer
	operations = fmt.Sprintf(operations, values...)
	operations = lib.Dedent(operations)
	stdin.Write([]byte(operations))
	shell, err := lib.ParseOperationsStdin(&stdin)
	check(err)
	return shell
}

func ParseOperationsErr(operations string, values ...any) (string, error) {
	var stdin bytes.Buffer
	operations = fmt.Sprintf(operations, values...)
	operations = lib.Dedent(operations)
	stdin.Write([]byte(operations))
	shell, err := lib.ParseOperationsStdin(&stdin)
	if err != nil {
		return "", err
	}
	return shell, nil
}

type CompleterProcess struct {
	mutex        sync.Mutex
	chanRequest  chan completeRequest
	chanResponse chan completeResponse
}

type completeRequest struct {
	testMethod string
	cmdStr     string
	file       string
	shell      string
}

func (c completeRequest) serialize() string {
	if c.testMethod == "bashfunc" {
		return strings.Join([]string{c.testMethod, c.cmdStr}, ":") + "\n"
	} else if c.testMethod == "expecttcl" {
		return strings.Join([]string{c.testMethod, c.file, c.cmdStr}, ":") + "\n"
	} else {
		panic("unknown testmethod: " + c.testMethod)
	}
}

type completeResponse struct {
	stdout string
	stderr string
	error  error
}

var processes map[string]*CompleterProcess

func (p *CompleterProcess) complete(request completeRequest) completeResponse {
	p.chanRequest <- request
	response := <-p.chanResponse
	return response
}

func completer(shell string) *CompleterProcess {
	if processes == nil {
		processes = make(map[string]*CompleterProcess)
	}
	shell = lib.Dedent(shell)

	if process, ok := processes[shell]; ok {
		return process
	} else {
		var err error
		var stdoutBuffer []byte
		var stderrBuffer []byte
		var mutex = sync.Mutex{}
		var chanFlush = make(chan bool)
		var chanResponse = make(chan completeResponse)

		process = &CompleterProcess{
			chanRequest:  make(chan completeRequest),
			chanResponse: make(chan completeResponse),
		}

		cwd, _ := os.Getwd()
		buildBinaryPath := path.Join(cwd, "build")

		envCopy := os.Environ()
		for i, keyVal := range envCopy {
			split := strings.Split(keyVal, "=")
			if split[0] == "PATH" {
				envCopy[i] = split[0] + "=" + buildBinaryPath + ":" + split[1]
			}
		}

		bashCommandStr := shell + "\n" + bashBridgeShell
		proc := exec.Command("bash", "-c", bashCommandStr)
		proc.Env = envCopy
		stdin, err := proc.StdinPipe()
		check(err)
		stdout, err := proc.StdoutPipe()
		check(err)
		stderr, err := proc.StderrPipe()
		check(err)
		err = proc.Start()
		check(err)

		// process requests
		go func() {
			for {
				request := <-process.chanRequest
				process.chanResponse <- func() completeResponse {
					mutex.Lock()
					defer mutex.Unlock()

					var chanTimeout = make(chan bool)
					var response completeResponse

					go func() {
						time.Sleep(time.Second)
						chanTimeout <- true
					}()

					_, _ = io.WriteString(stdin, request.serialize())

					select {
					case response = <-chanResponse:
					case <-chanTimeout:
						chanFlush <- true
						response = <-chanResponse
						response.error = fmt.Errorf("timeout waiting for completion call")
					}

					if response.stderr != "" && response.error == nil {
						lines := strings.Split(response.stderr, "\n")
						nonSetXDebug := false
						for _, line := range lines {
							if line != "" && !strings.HasPrefix(line, "+") {
								nonSetXDebug = true
							}
						}
						if nonSetXDebug {
							response.error = fmt.Errorf("unexpected stderr in completion call")
						}
					}

					return response
				}()
			}
		}()

		// read stdout
		go func() {
			var err error
			var n int
			buff := make([]byte, 256)
			for err == nil {
				n, err = stdout.Read(buff)
				for i := 0; i < n; i++ {
					if buff[i] == '\x00' {
						stdoutCopy := string(stdoutBuffer)
						stderrCopy := string(stderrBuffer)
						stdoutBuffer = []byte{}
						stderrBuffer = []byte{}
						chanResponse <- completeResponse{
							stdout: stdoutCopy,
							stderr: stderrCopy,
						}
					} else {
						stdoutBuffer = append(stdoutBuffer, buff[i])
					}
				}
			}
		}()

		// read stderr
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

		// check for flush (handy for timeouts due to errors)
		go func() {
			for {
				<-chanFlush
				stdoutCopy := string(stdoutBuffer)
				stderrCopy := string(stderrBuffer)
				stdoutBuffer = []byte{}
				stderrBuffer = []byte{}
				chanResponse <- completeResponse{
					stdout: stdoutCopy,
					stderr: stderrCopy,
				}
			}
		}()

		processes[shell] = process
		return process
	}
}
