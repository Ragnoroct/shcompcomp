package main

import (
	"bctils/pkg/lib"
	"bctils/pkg/testutil"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"os"
	"path"
	"testing"
)

var loggerCleanup func()

func TestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

type MainTestSuite struct {
	suite.Suite
	tmpdir string
}

func (suite *MainTestSuite) SetupSuite() {
	loggerCleanup = setupLogger()
	log.Printf("RUNNING TESTS")
}

func (suite *MainTestSuite) SetupTest() {
	suite.tmpdir = ""
}

func (suite *MainTestSuite) SetupSubTest() {
	suite.tmpdir = ""
}

func (suite *MainTestSuite) TearDownSuite() {
	defer loggerCleanup()
}

func (suite *MainTestSuite) AssertComplete(shell, cmdStr string, expected string) {
	suite.T().Helper()
	testutil.ExpectComplete(suite.T(), shell, cmdStr, expected)
}

func (suite *MainTestSuite) CreateFile(filename string, contents string) (filepath string) {
	if suite.tmpdir == "" {
		suite.tmpdir = suite.T().TempDir()
	}

	filepath = path.Join(suite.tmpdir, filename)
	contents = lib.Dedent(contents)
	err := os.WriteFile(filepath, []byte(contents), 0644)
	if err != nil {
		panic(err)
	}

	return filepath
}

func (suite *MainTestSuite) TestCases() {
	suite.Run("simple", func() {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=bobman
			opt "-h"
			opt "--help"
		`)
		suite.AssertComplete(shell, "bobman ", "-h --help")
		suite.AssertComplete(shell, "bobman --he", "--help")
		suite.AssertComplete(shell, "bobman -h", "")
	})

	suite.Run("positionals with choices", func() {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=testcli
			pos --choices="c1 c2 c3"
			pos --choices="c4 c5 c6"
		`)
		suite.AssertComplete(shell, "testcli ", "c1 c2 c3")
		suite.AssertComplete(shell, "testcli c", "c1 c2 c3")
		suite.AssertComplete(shell, "testcli d", "")
		suite.AssertComplete(shell, "testcli c2 ", "c4 c5 c6")
		suite.AssertComplete(shell, "testcli c2 d", "")
	})

	suite.Run("positionals with choices and optionals", func() {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=testcli
			pos --choices="c1 c2 c3"
			opt "-h"
			opt "--help"
		`)
		suite.AssertComplete(shell, "testcli ", "c1 c2 c3 -h --help")
		suite.AssertComplete(shell, "testcli c", "c1 c2 c3")
		suite.AssertComplete(shell, "testcli -", "-h --help")
		suite.AssertComplete(shell, "testcli c3 ", "-h --help")
		suite.AssertComplete(shell, "testcli --help ", "c1 c2 c3 -h")
		suite.AssertComplete(shell, "testcli -h ", "c1 c2 c3 --help")
	})

	suite.Run("simple subparsers", func() {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=testcli
			opt "-h"
			opt "--help"
			pos --parsers="sub-cmd"
			pos -p="sub-cmd" --choices="c1 c2 c3"
			opt -p="sub-cmd" "--awesome"
			opt -p="sub-cmd" "--print"
		`)
		suite.AssertComplete(shell, "testcli ", "-h --help")
		suite.AssertComplete(shell, "testcli sub-cmd ", "c1 c2 c3 --awesome --print")
	})

	suite.Run("nested subparsers", func() {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=testcli
			pos -p="sub-b" 				--help-b
			pos -p="sub-b.sub-c" 		--help-c
			pos -p="sub-b.sub-c.sub-d" 	--help-d
		`)
		suite.AssertComplete(shell, "testcli ", "sub-b")
		suite.AssertComplete(shell, "testcli sub-b ", "sub-c --help-b")
		suite.AssertComplete(shell, "testcli sub-b sub-c ", "sub-d --help-c")
		suite.AssertComplete(shell, "testcli sub-b sub-c sub-d ", "--help-d")
	})

	suite.Run("subparsers cmds are always the first positional and cannot clash", func() {})
	suite.Run("allow closure for positionals", func() {
		suite.CreateFile("closure.sh", `
			  __testcli_pos_1_completer() {
				mapfile -t COMPREPLY < <(compgen -W "c8 c9 c10" -- "$current_word")
				log "called"
			  }
		`)
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=testcli
			pos --closure="__examplecli5_pos_1_completer"
		`)
		suite.AssertComplete(shell, "testcli ", "")
		suite.AssertComplete(shell, "testcli sub-cmd ", "c1 c2 c3 --awesome --print")
	})
	suite.Run("error handling", func() {})
	suite.Run("compile to target file", func() {})
	suite.Run("sort results by pos -> --help option", func() {})
}

func (suite *MainTestSuite) TestMainToStdout() {
	stdout := mainWithStdout(
		`
		cfg cli_name=bobman
		cfg outfile=-
		opt "-h"
		opt "--help"
		`,
	)
	if len(stdout) == 0 {
		suite.T().Fatalf("stdout from calling main produced no output")
	}
}

func (suite *MainTestSuite) TestEndToEndAutoGenWithReload() {
	t := suite.T()
	tmpDir := t.TempDir()

	writeFile := func(filename string, value string) {
		value = lib.Dedent(value)
		err := os.WriteFile(path.Join(tmpDir, filename), []byte(value), 0644)
		lib.Check(err)
	}

	writeFile("cmd.py", `
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--awesome")
		`)

	writeFile("cmdlib.sh", fmt.Sprintf(`
			cmd_autogen_piper () {
				cat %s
			}
		`, path.Join(tmpDir, "cmd.py")))

	completeFile := path.Join(tmpDir, "cmd.bash")
	mainWithStdout(
		fmt.Sprintf(
			`
				cfg cli_name=bobman
				cfg autogen_lang=py
				cfg autogen_closure_func=cmd_autogen_piper
				cfg autogen_closure_source=%s
				cfg autogen_reload_trigger=%s
				cfg outfile=%s
				`,
			path.Join(tmpDir, "cmdlib.sh"),
			path.Join(tmpDir, "cmd.py"),
			completeFile,
		),
	)

	hashFile := func(filename string) string {
		filename = path.Join(tmpDir, filename)
		hasher := md5.New()
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				panic(err)
			}
		}(f)
		if _, err := io.Copy(hasher, f); err != nil {
			log.Fatal(err)
		}
		return hex.EncodeToString(hasher.Sum(nil))
	}

	// todo: test that it only reloads when changes are made to trigger files
	testutil.ExpectCompleteFile(t, completeFile, "bobman ", "--awesome")

	hashBeforeReload := hashFile("cmd.bash")

	writeFile("cmd.py", `
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--awesome-times-infinity")
		`)

	testutil.ExpectCompleteFile(t, completeFile, "bobman ", "--awesome-times-infinity")
	hashAfterReload := hashFile("cmd.bash")
	testutil.ExpectCompleteFile(t, completeFile, "bobman ", "--awesome-times-infinity")
	hashAfterNoReload := hashFile("cmd.bash")

	suite.NotEqual(hashBeforeReload, hashAfterReload)
	suite.Equal(hashAfterNoReload, hashAfterReload)
}

func mainWithStdout(stdin string) (stdout string) {
	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	var stdinReader bytes.Buffer
	stdinReader.WriteString(stdin)
	entry(&stdinReader, &stdoutWriter, &stderrWriter, Options{checkReload: false, args: []string{"-"}})
	return stdoutWriter.String()
}
