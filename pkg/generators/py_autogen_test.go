package generators

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/suite"
	"path"
	"shcomp2/pkg/lib"
	"shcomp2/pkg/testutil"
	"testing"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

type Suite struct {
	testutil.BaseSuite
}

func (suite *Suite) AutogenParse(src string) string {
	filename := suite.CreateFile("file.py", src)
	cli := lib.ParseOperations(fmt.Sprintf(`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_file=%s
		cfg outfile=-
	`, filename))
	cli = GeneratePythonOperations2(cli)
	shell, err := lib.CompileCli(cli)
	if err != nil {
		panic(err)
	}
	return shell
}

func (suite *Suite) AutogenParseCfg(cfg string, values ...any) string {
	var nullbuffer bytes.Buffer
	cfg = fmt.Sprintf(cfg, values...)
	cli := lib.ParseOperations(cfg)
	cli = GeneratePythonOperations2(cli)
	shell, err := lib.CompileCli(cli)
	if err != nil {
		panic(err)
	}
	err = lib.CommitCli(cli, shell, &nullbuffer)
	if err != nil {
		panic(err)
	}
	return shell
}

func (suite *Suite) TestAutoGen() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--some-way")
		subparsers = parser.add_subparsers()
		parser_cmd = subparsers.add_parser("sub-cmd-name")
		parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"])
	`)
	suite.RequireComplete(shell, "testcli ", "sub-cmd-name --some-way")
	suite.RequireComplete(shell, "testcli sub-cmd-name ", "c1 c2 c3")
}

func (suite *Suite) TestTrueFalseArgs() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--some-way")
		subparsers = parser.add_subparsers(help="sub-command help", dest="command", required=True)
		parser_cmd = subparsers.add_parser("sub-cmd-name")
		parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"], required=False)
	`)
	suite.RequireComplete(shell, "testcli ", "sub-cmd-name --some-way")
	suite.RequireComplete(shell, "testcli sub-cmd-name ", "c1 c2 c3")
}

func (suite *Suite) TestSimpleOption() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
	`)
	suite.RequireComplete(shell, "testcli ", "--help")
}

func (suite *Suite) TestSimplePos() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("arg1", choices=["c1", "c2", "c3"])
		parser.add_argument("arg2", choices=["c4", "c5", "c6"])
	`)
	suite.RequireComplete(shell, "testcli ", "c1 c2 c3")
	suite.RequireComplete(shell, "testcli c2 ", "c4 c5 c6")
}

func (suite *Suite) TestSimpleIgnoresOtherMethods() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("arg1", choices=["c1", "c2", "c3"])
		parser.add_argument("arg2", choices=["c4", "c5", "c6"])
		parser.add_argument_ignored("arg3", choices=["c7", "c8", "c9"])
	`)
	suite.RequireComplete(shell, "testcli ", "c1 c2 c3")
	suite.RequireComplete(shell, "testcli c2 ", "c4 c5 c6")
	suite.RequireComplete(shell, "testcli c2 c5 ", "")
}

func (suite *Suite) TestSubparserWithoutLeftOperand() {
	shell := suite.AutogenParse(`
		parser = ArgumentParser()
		subparsers = parser.add_subparsers(help="sub-command help", dest="command", required=True)
		subparsers.add_parser("standalone")
		parser_nested = subparsers.add_parser("nested")
		parser_nested.add_argument("arg1", choices=["c1"])
		parser_nested.add_argument("--test-1")
	`)
	suite.RequireComplete(shell, "testcli ", "standalone nested")
	suite.RequireComplete(shell, "testcli standalone ", "")
	suite.RequireComplete(shell, "testcli nested ", "c1 --test-1")
	suite.RequireComplete(shell, "testcli nested c1 ", "--test-1")
}

func (suite *Suite) TestTripleLayerSubparser() {
	shell := suite.AutogenParse(`
		from argparse import ArgumentParser
		parser_a = ArgumentParser()
		parser_a.add_argument("--help-a")
		subparsers_a = parser_a.add_subparsers()

		parser_b = subparsers_a.add_parser("parser-b")
		parser_b.add_argument("--help-b")
		subparsers_b = parser_b.add_subparsers()

		parser_c = subparsers_b.add_parser("parser-c")
		parser_c.add_argument("--help-c")

		parser.parse_args()
	`)
	suite.RequireComplete(shell, "testcli ", "parser-b --help-a")
	suite.RequireComplete(shell, "testcli parser-b ", "parser-c --help-b")
	suite.RequireComplete(shell, "testcli parser-b parser-c ", "--help-c")
}

func (suite *Suite) TestChooseOutfile() {
	file := suite.CreateFile("file.py", `
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
	`)

	outfile := path.Join(suite.TempDir(), "outfile.bash")

	suite.AutogenParseCfg(
		`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_file=%s
		cfg outfile=%s
		`,
		file,
		outfile,
	)
	suite.RequireCompleteFile(outfile, "testcli ", "--help")
}

func (suite *Suite) TestSrcFromFile() {
	file := suite.CreateFile("file.py", `
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
	`)

	shell := suite.AutogenParseCfg(
		`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_file=%s
		cfg outfile=-
		`,
		file,
	)
	suite.RequireComplete(shell, "testcli ", "--help")
}

func (suite *Suite) TestSrcFromBashFunction() {
	cmdlibfile := suite.CreateFile("cmdlib.sh", `
		__my_piper_func () {
		cat - <<EOF
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
		EOF
		}
	`)

	shell := suite.AutogenParseCfg(
		`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_closure_func=__my_piper_func
		cfg autogen_closure_source=%s
		cfg outfile=-
		`,
		cmdlibfile,
	)

	suite.RequireComplete(shell, "testcli ", "--help")
}

func (suite *Suite) TestSrcFromCmd() {
	commandfile := suite.CreateFile("command_file", `
		#!/usr/bin/bash
		cat - <<EOF
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
		EOF
	`, 0744)

	shell := suite.AutogenParseCfg(
		`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_closure_cmd=%s
		cfg outfile=-
		`,
		commandfile,
	)

	suite.RequireComplete(shell, "testcli ", "--help")
}

func (suite *Suite) FutureTests() {
	suite.Run("order of operations is always the same", func() {})
	suite.Run("options with values", func() {})
	suite.Run("options with values", func() {})
	suite.Run("allow closures through comments", func() {})
	suite.Run("work with multiple files with same parser", func() {})
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
	suite.Run("nargs with known number", func() {})
	suite.Run("nargs with unknown number", func() {})
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
}
