package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"io"
	"os"
	"path"
	"shcomp2/pkg/lib"
	"shcomp2/pkg/testutil"
	"testing"
	"time"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	testutil.BaseSuite
}

func (suite *Suite) TestCases() {
	suite.Run("simple", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=bobman
			opt "-h"
			opt "--help"
		`)
		suite.RequireComplete(shell, "bobman ", "-h --help")
		suite.RequireComplete(shell, "bobman --he", "--help")
		suite.RequireComplete(shell, "bobman -h ", "--help")
	})

	suite.Run("positionals with choices", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="c1 c2 c3"
			pos --choices="c4 c5 c6"
		`)
		suite.RequireComplete(shell, "testcli ", "c1 c2 c3")
		suite.RequireComplete(shell, "testcli c", "c1 c2 c3")
		suite.RequireComplete(shell, "testcli d", "")
		suite.RequireComplete(shell, "testcli c2 ", "c4 c5 c6")
		suite.RequireComplete(shell, "testcli c2 d", "")
	})

	suite.Run("positionals with choices and optionals", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="c1 c2 c3"
			opt "-h"
			opt "--help"
		`)
		suite.RequireComplete(shell, "testcli --help ", "c1 c2 c3 -h")
		suite.RequireComplete(shell, "testcli ", "c1 c2 c3 -h --help")
		suite.RequireComplete(shell, "testcli c", "c1 c2 c3")
		suite.RequireComplete(shell, "testcli -", "-h --help")
		suite.RequireComplete(shell, "testcli c3 ", "-h --help")
		suite.RequireComplete(shell, "testcli -h ", "c1 c2 c3 --help")
	})

	suite.Run("simple subparsers", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt "-h"
			opt "--help"
			pos -p="sub-cmd" --choices="c1 c2 c3"
			opt -p="sub-cmd" "--awesome"
			opt -p="sub-cmd" "--print"
		`)
		suite.RequireComplete(shell, "testcli ", "sub-cmd -h --help")
		suite.RequireComplete(shell, "testcli sub-cmd ", "c1 c2 c3 --awesome --print")
	})

	suite.Run("nested subparsers", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt -p="sub-b" 				--help-b
			opt -p="sub-b.sub-c" 		--help-c
			opt -p="sub-b.sub-c.sub-d" 	--help-d
		`)
		suite.RequireComplete(shell, "testcli ", "sub-b")
		suite.RequireComplete(shell, "testcli sub-b ", "sub-c --help-b")
		suite.RequireComplete(shell, "testcli sub-b sub-c ", "sub-d --help-c")
		suite.RequireComplete(shell, "testcli sub-b sub-c sub-d ", "--help-d")
	})

	suite.Run("allow closure for positionals", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --closure="__testcli_pos_1_completer"
			opt --awesome
			opt --print
		`)
		shell += lib.Dedent(`
			__testcli_pos_1_completer() {
				mapfile -t COMPREPLY < <(compgen -W "c8 c9 c10" -- "$current_word")
			}
		`)
		suite.RequireComplete(shell, "testcli ", "c8 c9 c10 --awesome --print")
	})

	suite.Run("include other source files", func() {
		libpath := suite.CreateFile("lib.sh", `
			__testcli_pos_1_completer() {
				mapfile -t COMPREPLY < <(compgen -W "c8 c9 c10" -- "$current_word")
			}
		`)
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			cfg include_source="%s"
			pos --closure="__testcli_pos_1_completer"
			opt --awesome
			opt --print
		`, libpath)
		suite.RequireComplete(shell, "testcli ", "c8 c9 c10 --awesome --print")
	})

	suite.Run("simple options with arguments like --opt val", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt "--key" --choices="val1 val2"
			opt "--tree" --closure="__testcli_completer"
		`)
		shell += lib.Dedent(`
			__testcli_completer() {
				mapfile -t COMPREPLY < <(compgen -W "c8 c9 c10" -- "$current_word")
			}
		`)
		suite.RequireComplete(shell, "testcli ", "--key --tree")
		suite.RequireComplete(shell, "testcli --key ", "val1 val2")
		suite.RequireComplete(shell, "testcli --tree ", "c8 c9 c10")
	})

	suite.Run("order of operations is always the same", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt "--key"
			opt "--tree"
		`)
		shell2 := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt "--tree"
			opt "--key"
		`)
		suite.RequireComplete(shell, "testcli ", "--key --tree")
		suite.RequireComplete(shell2, "testcli ", "--tree --key")
	})

	// todo: figure out if min range is ever useful
	suite.Run("nargs with known number non-unique", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="one two three" --nargs=3
		`)
		suite.RequireComplete(shell, "testcli ", "one two three")
		suite.RequireComplete(shell, "testcli one ", "one two three")
		suite.RequireComplete(shell, "testcli one two ", "one two three")
	})
	suite.Run("nargs with range", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="one two" --nargs={1,2}
		`)
		suite.RequireComplete(shell, "testcli ", "one two")
		suite.RequireComplete(shell, "testcli one ", "one two")
	})
	suite.Run("nargs with unique choices", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="one two three" --nargs=3 --nargs-unique
		`)
		suite.RequireComplete(shell, "testcli ", "one two three")
		suite.RequireComplete(shell, "testcli one ", "two three")
		suite.RequireComplete(shell, "testcli one two ", "three")
	})
	suite.Run("nargs with zero to unlimited", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="many" --nargs={0,inf}
		`)
		suite.RequireComplete(shell, "testcli ", "many")
		suite.RequireComplete(shell, "testcli many ", "many")
		suite.RequireComplete(shell, "testcli many many asd a a f dsaf asdd asdf saf asdf ", "many")
	})
	suite.Run("nargs unlimited shorthand", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="many" --nargs=*
		`)
		suite.RequireComplete(shell, "testcli ", "many")
		suite.RequireComplete(shell, "testcli many ", "many")
		suite.RequireComplete(shell, "testcli many many asd a a f dsaf asdd asdf saf asdf ", "many")
	})
	suite.Run("nargs mixing unlimited with known", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			pos --choices="one"
			pos --choices="two"
			pos --choices="many" --nargs={0,inf}
		`)
		suite.RequireComplete(shell, "testcli ", "one")
		suite.RequireComplete(shell, "testcli many ", "two")
		suite.RequireComplete(shell, "testcli many many asd a a f dsaf asdd asdf saf asdf ", "many")
	})
	suite.Run("nargs opt simple range", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt -v --nargs=3
		`)
		suite.RequireComplete(shell, "testcli ", "-v")
		suite.RequireComplete(shell, "testcli -v ", "-v")
		suite.RequireComplete(shell, "testcli -v -v ", "-v")
		suite.RequireComplete(shell, "testcli -v -v -v ", "")
	})
	suite.Run("nargs opt many", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt -v --nargs=*
		`)
		suite.RequireComplete(shell, "testcli ", "-v")
		suite.RequireComplete(shell, "testcli -v ", "-v")
		suite.RequireComplete(shell, "testcli -v -v ", "-v")
		suite.RequireComplete(shell, "testcli -v -v -v asd asdas dasf asdas asdf s ", "-v")
	})
	suite.Run("nargs opt value simple range", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt --key --choices="value" --nargs=2
		`)
		suite.RequireComplete(shell, "testcli ", "--key")
		suite.RequireComplete(shell, "testcli --key ", "value")
		suite.RequireComplete(shell, "testcli --key value ", "--key")
		suite.RequireComplete(shell, "testcli --key value --key ", "value")
		suite.RequireComplete(shell, "testcli --key value --key value ", "")
	})
	suite.Run("nargs opt value many", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt --key --choices="value" --nargs=*
		`)
		suite.RequireComplete(shell, "testcli ", "--key")
		suite.RequireComplete(shell, "testcli --key ", "value")
		suite.RequireComplete(shell, "testcli --key value ", "--key")
		suite.RequireComplete(shell, "testcli --key value --key ", "value")
		suite.RequireComplete(shell, "testcli --key value --key value asdf asdf sdfdsfd asddf faa ", "--key")
	})
	suite.Run("option alternatives", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt --help|-help|-h
		`)
		suite.RequireComplete(shell, "testcli ", "--help -help -h")
		suite.RequireComplete(shell, "testcli -h ", "")
		suite.RequireComplete(shell, "testcli --help ", "")
		suite.RequireComplete(shell, "testcli -help ", "")
	})
	suite.Run("combining single opt flags -v -v -v into -vvv", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			cfg merge_single_opt=1
			opt -v --nargs=3
		`)
		suite.RequireComplete(shell, "testcli ", "-v")
		suite.RequireComplete(shell, "testcli -v", "-vv")
		suite.RequireComplete(shell, "testcli -vv", "-vvv")
		suite.RequireComplete(shell, "testcli -vvv", "")
		suite.RequireComplete(shell, "testcli -vvv ", "")
	})
	suite.Run("should only complete word if no other matches and no space", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			opt -v
			opt -verbose
		`)
		suite.RequireComplete(shell, "testcli ", "-v -verbose")
		suite.RequireComplete(shell, "testcli ", "-v -verbose")
		suite.RequireComplete(shell, "testcli -v", "-verbose")
	})
	suite.Run("merge -abc", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			cfg merge_single_opt=1
			opt -a
			opt -b
			opt -c
			opt -d
		`)
		suite.RequireComplete(shell, "testcli ", "-a -b -c -d")
		suite.RequireComplete(shell, "testcli -a", "-ab -ac -ad")
		suite.RequireComplete(shell, "testcli -ab", "-abc -abd")
		suite.RequireComplete(shell, "testcli -abc", "-abcd")
		suite.RequireComplete(shell, "testcli -abcd", "")
	})
	suite.Run("merge regular and nargs", func() {
		shell := testutil.ParseOperations(`
			cfg cli_name=testcli
			cfg merge_single_opt=1
			opt -a --nargs=3
			opt -b
		`)
		suite.RequireComplete(shell, "testcli ", "-a -b")
		suite.RequireComplete(shell, "testcli -a", "-aa -ab")
		suite.RequireComplete(shell, "testcli -aa", "-aaa -aab")
		suite.RequireComplete(shell, "testcli -aaa", "-aaab")
		suite.RequireComplete(shell, "testcli -ab", "-aba")
	})
	suite.Run("allow closures through comments", func() {})
}

//func (suite *Suite) TestCompleteWithExpectTcl() {
//	shell := testutil.ParseOperations(`
//		cfg cli_name=testcli
//		cfg merge_single_opt=1
//		opt -a --nargs=3
//		opt -b
//	`)
//	suite.RequireComplete(shell, "testcli -ab", "testcli -aba")
//}

func (suite *Suite) FutureTests() {
	suite.Run("include other source files error handling when missing include source", func() {})
	suite.Run("sort results by pos -> --help option", func() {})
	suite.Run("options with values but prefer equals sign", func() {})
	suite.Run("autgenpy follow imports to other files", func() {})
	suite.Run("subparsers cmds are always the first positional and cannot clash", func() {})
	suite.Run("custom log", func() {})
	suite.Run("source ~/.bashrc is FAST with MANY 'autogen calls'", func() {})
	suite.Run("cache all compiles", func() {})
	suite.Run("move tests into golang environment", func() {})
	suite.Run("adding to .bashrc and removing it still adds time and is accumalative", func() {})
	suite.Run("positionals without hints are recognized countwise", func() {})
	suite.Run("py_autogen detect disabling --help/-h", func() {})
	suite.Run("shcomp2_autogen specify out file", func() {})
	suite.Run("exclusive options --vanilla --chocolate", func() {})
	suite.Run("complete option value like --opt=value", func() {})
	suite.Run("add flag to auto add = if only one arg option left and it requires an argument", func() {})
	suite.Run("complete single -s type options like -f filepath", func() {})
	suite.Run("complete single -s type options like -ffilepath (if that makes sense)", func() {})
	suite.Run("arbitrary rules like -flags only before positionals", func() {})
	suite.Run("arbitrary rules like --options values only or --options=values only for long args (getopt bug)", func() {})
	suite.Run("simple options and arguments with nargs=*", func() {})
	suite.Run("more complex autocomplete in different parts of command", func() {})
	suite.Run("advanced subparsers with options + arguments at different levels", func() {})
	suite.Run("cache based on input argument streams", func() {})
	suite.Run("feature complete existing", func() {})
	suite.Run("benchmark testing compilation", func() {})
	suite.Run("benchmark testing compilation caching", func() {})
	suite.Run("benchmark testing autogeneration of python script", func() {})
	suite.Run("benchmark source shcomp2 lib", func() {})
	suite.Run("benchmark source compiled scripts", func() {})
	suite.Run("use -- in util scripts to separate arguments from options", func() {})
	suite.Run("allow single -longopt like golang", func() {})
	suite.Run("allow opt=val and opt val", func() {})
	suite.Run("tab complete opt -> opt=", func() {})
	suite.Run("choices for options with arguments", func() {})
	suite.Run("scan python script for auto generate", func() {})
	suite.Run("get compiled script version", func() {})
	suite.Run("multiple functions to single file", func() {})
	suite.Run("compiled scripts are slim and simplified", func() {})
	suite.Run("provide custom functions -F to autocomplete arguments and options", func() {})
	suite.Run("provide custom functions -F to autocomplete subparsers arguments and options", func() {})
	suite.Run("provide custom functions -F to autocomplete option values", func() {})
	suite.Run("provide custom functions -F to autocomplete subparsers option values", func() {})
	suite.Run("invalid usages of shcomp2 utility functions", func() {})
	suite.Run("stateless in environment after compilation. no leftover variables.", func() {})
	suite.Run("zero logging when in production mode", func() {})
	suite.Run("doesnt share variable state between different cli_name", func() {})
	suite.Run("can provide autocompletion custom git extensions", func() {})
	suite.Run("has full documentation", func() {})
	suite.Run("compatable with bash,sh,zsh,ksh,msys2,fish,cygwin,bashwin (docker emulation)", func() {})
	suite.Run("is fast with very large complete options like aws", func() {})
	suite.Run("minimal forks in completion", func() {})
	suite.Run("minimal memory usage in completion", func() {})
	suite.Run("easy to install", func() {})
	suite.Run("install with choice of plugins", func() {})
	suite.Run("git plugin", func() {})
	suite.Run("npm plugin", func() {})
	suite.Run("autogen_py plugin", func() {})
	suite.Run("autogen_node plugin", func() {})
	suite.Run("autogen_golang plugin", func() {})
	suite.Run("autogen_sh plugin", func() {})
	suite.Run("compiled scripts are actually readable", func() {})
	suite.Run("compiled scripts contain auto-generated comment and license", func() {})
	suite.Run("project is licensed", func() {})
	suite.Run("order of options added and argument choices is order shown", func() {})
	suite.Run("all error messages match current script name", func() {})
	suite.Run("error handling", func() {})
	suite.Run("rethink nargs complexity. is specifying a lower range really necessary", func() {})
}

func (suite *Suite) TestMainToStdout() {
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

func (suite *Suite) TestEndToEndAutoGenWithReload() {
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
			log.Fatal().Msg(err.Error())
		}
		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				panic(err)
			}
		}(f)
		if _, err := io.Copy(hasher, f); err != nil {
			log.Fatal().Msg(err.Error())
		}
		return hex.EncodeToString(hasher.Sum(nil))
	}

	// todo: test that it only reloads when changes are made to trigger files
	suite.RequireCompleteFile(completeFile, "bobman ", "--awesome")

	hashBeforeReload := hashFile("cmd.bash")

	writeFile("cmd.py", `
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--awesome-times-infinity")
		`)
	time.Sleep(time.Millisecond) // allow reload to pickup time change

	suite.RequireCompleteFile(completeFile, "bobman ", "--awesome-times-infinity")
	hashAfterReload := hashFile("cmd.bash")
	suite.RequireCompleteFile(completeFile, "bobman ", "--awesome-times-infinity")
	hashAfterNoReload := hashFile("cmd.bash")

	suite.NotEqual(hashBeforeReload, hashAfterReload)
	suite.Equal(hashAfterNoReload, hashAfterReload)
}

func (suite *Suite) TestMainHandlesError() {
	result := executeEntry(lib.Dedent(`
		cfg cli_name=testcli
		pos --nargs=*
		pos
	`))
	suite.Require().Equal(1, result.code)
	suite.Require().Equal("error: cannot have a positional come after a indeterminant narg positional\n", result.stderr)
}

func (suite *Suite) TestNargsErrorHandling() {
	var errIndeterm = "cannot have a positional come after a indeterminant narg positional"

	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			"pos after *",
			`
				cfg cli_name=testcli
				pos --nargs=*
				pos
				`,
			errIndeterm,
		},
		{
			"pos after range",
			`
				cfg cli_name=testcli
				pos --nargs={0,1}
				pos
				`,
			errIndeterm,
		},
		{
			"opt with narg after range okay",
			`
				cfg cli_name=testcli
				pos --nargs={0,1}
				opt -v --nargs=*
				`,
			"",
		},
		{
			"invalid syntax missing end }",
			`
				cfg cli_name=testcli
				pos --nargs={0,1
				`,
			"unable to parse nargs {0,1",
		},
		{
			"invalid syntax",
			`
				cfg cli_name=testcli
				pos --nargs=bob
				`,
			"unable to parse nargs bob",
		},
		{
			"negative numbers",
			`
				cfg cli_name=testcli
				pos --nargs=-1
				`,
			"cannot have negative values for nargs -1",
		},
		{
			"negative numbers",
			`
				cfg cli_name=testcli
				pos --nargs={1,-3}
				`,
			"cannot have negative values for nargs {1,-3}",
		},
		{
			"max zero",
			`
				cfg cli_name=testcli
				pos --nargs={0,0}
				`,
			"cannot use 0 for narg max {0,0}",
		},
		{
			"min greater than max",
			`
				cfg cli_name=testcli
				pos --nargs={2,1}
				`,
			"cannot have a min greater than max narg {2,1}",
		},
		{
			"min greater than max",
			`
				cfg cli_name=testcli
				pos --nargs={inf,1}
				`,
			"cannot have a min greater than max narg {inf,1}",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			_, err := testutil.ParseOperationsErr(tt.input)
			if tt.expect != "" || err != nil {
				suite.Assert().EqualError(err, tt.expect)
			} else {
				suite.Assert().Equal(nil, err)
			}
		})
	}
}

func mainWithStdout(stdin string) (stdout string) {
	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	var stdinReader bytes.Buffer
	stdinReader.WriteString(stdin)
	entry(&stdinReader, &stdoutWriter, &stderrWriter, Options{checkReload: false, args: []string{"-"}})
	return stdoutWriter.String()
}

type resultMain struct {
	code   int
	stdout string
	stderr string
}

func executeEntry(stdin string) resultMain {
	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	var stdinReader bytes.Buffer
	stdinReader.WriteString(stdin)
	exitCode := entry(&stdinReader, &stdoutWriter, &stderrWriter, Options{checkReload: false, args: []string{"-"}})
	return resultMain{
		code:   exitCode,
		stdout: stdoutWriter.String(),
		stderr: stderrWriter.String(),
	}
}
