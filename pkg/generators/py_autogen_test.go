package generators

import (
	"bctils/pkg/lib"
	"bctils/pkg/testutil"
	"fmt"
	"github.com/stretchr/testify/suite"
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

func (suite *Suite) FutureTests() {
	suite.Run("order of operations is always the same", func() {})
	suite.Run("options with values", func() {})
	suite.Run("options with values", func() {})
	suite.Run("allow closures through comments", func() {})
	suite.Run("work with multiple files with same parser", func() {})
}
