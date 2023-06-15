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
	cli, err := lib.ParseOperations(fmt.Sprintf(`
		cfg cli_name=testcli
		cfg autogen_lang=py
		cfg autogen_file=%s
		cfg outfile=-
	`, filename))
	check(err)
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
	cli, err := lib.ParseOperations(cfg)
	check(err)
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

func (suite *Suite) TestNargs() {}
