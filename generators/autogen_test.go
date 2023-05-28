package generators

import (
	"strings"
	"testing"
)

func TestSimpleArgument(t *testing.T) {
	expectedArgs := dedent(`
		bctils_cli_add examplecli pos --choices="c1 c2 c3"
		bctils_cli_add examplecli pos --choices="c4 c5 c6"
	`)
	actualArgs := parseSrc(
		"examplecli",
		dedent(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("arg1", choices=["c1", "c2", "c3"])
		parser.add_argument("arg2", choices=["c4", "c5", "c6"])
	`))
	expectEqual(t, expectedArgs, actualArgs, "")
}

func TestSimpleOption(t *testing.T) {
	expectedArgs := dedent(`
		bctils_cli_add examplecli opt "--help"
	`)
	actualArgs := parseSrc(
		"examplecli",
		dedent(`
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--help")
	`))
	expectEqual(t, expectedArgs, actualArgs, "")
}

func expectEqual(t *testing.T, expected string, actual string, msg string) {
	if msg == "" {
		msg = "strings are not equal"
	}
	if actual != expected {
		t.Fatalf(
			"%s\nactual:\n'''\n%s'''\nexpected:\n'''\n%s'''", msg,
			actual,
			expected,
		)
	}
}

func dedent(str string) string {
	if str[0] == '\n' {
		str = str[1:]
	}
	lines := strings.Split(str, "\n")
	minIndent := -1
	for _, line := range lines {
		for i, c := range line {
			if c == ' ' {
				panic("cannot handle mixing spaces with tab")
			} else if c != '\t' {
				if minIndent == -1 || i < minIndent {
					minIndent = i
				}
				break
			}
		}
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
