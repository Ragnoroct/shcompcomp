package generators

import (
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	defer setupLogger()()
	log.Printf("=== RUNNING TESTS")
	code := m.Run()
	if code == 1 {
		log.Printf("=== RESULTS FAIL : %s", time.Now().Format("3:4:5.000"))
	} else {
		log.Printf("=== RESULTS PASS : %s", time.Now().Format("3:4:5.000"))
	}
	os.Exit(code)
}

func TestAutoGen(t *testing.T) {
	runSubtest(t, "simple subparser", func(t *testing.T) {
		expectOperations(
			t,
			`
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--some-way")
			subparsers = parser.add_subparsers()
			parser_cmd = subparsers.add_parser("sub-cmd-name")
			parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"])`,
			`
			pos --choices="sub-cmd-name"
			opt "--some-way"
			pos -p="sub-cmd-name" --choices="c1 c2 c3"
			`,
		)
	})

	runSubtest(t, "true/false arguments", func(t *testing.T) {
		expectOperations(
			t,
			`
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--some-way")
			subparsers = parser.add_subparsers(help="sub-command help", dest="command", required=True)
			parser_cmd = subparsers.add_parser("sub-cmd-name")
			parser_cmd.add_argument("arg1", choices=["c1", "c2", "c3"], required=False)`,
			`
			pos --choices="sub-cmd-name"
			opt "--some-way"
			pos -p="sub-cmd-name" --choices="c1 c2 c3"
			`,
		)
	})

	runSubtest(t, "simple option", func(t *testing.T) {
		expectOperations(
			t,
			`
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("--help")
			`,
			`
			opt "--help"
			`,
		)
	})

	runSubtest(t, "simple option", func(t *testing.T) {
		expectOperations(
			t,
			`
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("arg1", choices=["c1", "c2", "c3"])
			parser.add_argument("arg2", choices=["c4", "c5", "c6"])
			`,
			`
			pos --choices="c1 c2 c3"
			pos --choices="c4 c5 c6"
			`,
		)
	})

	runSubtest(t, "ignores other methods", func(t *testing.T) {
		expectOperations(
			t,
			`
			from argparse import ArgumentParser
			parser = ArgumentParser()
			parser.add_argument("arg1", choices=["c1", "c2", "c3"])
			parser.add_argument("arg2", choices=["c4", "c5", "c6"])
			parser.add_argument_ignored("arg3", choices=["c7", "c8", "c9"])
			`,
			`
			pos --choices="c1 c2 c3"
			pos --choices="c4 c5 c6"
			`,
		)
	})

	runSubtest(t, "add subparser without left operand still works", func(t *testing.T) {
		expectOperations(
			t,
			`
			parser = ArgumentParser()
			subparsers = parser.add_subparsers(help="sub-command help", dest="command", required=True)
			subparsers.add_parser("pio")
			parser_release_freeze = subparsers.add_parser("release-freeze")
			parser_release_freeze.add_argument("version", help="example: 2.14.0")
			parser_release_freeze.add_argument("part", help="example: minor", nargs="?", choices=["major", "minor", "patch"], default="minor")
			`,
			`
			pos --choices="pio release-freeze"
			pos -p="release-freeze"
			pos -p="release-freeze" --choices="major minor patch"
			`,
		)
	})

	runSubtest(t, "order of operations is always the same", nil)

	runSubtest(t, "options with values", nil)
	runSubtest(t, "options with values", nil)
	runSubtest(t, "allow closures through comments", nil)
	runSubtest(t, "work with multiple files with same parser", nil)
}

func expectOperations(t *testing.T, src string, expected string) {
	actual := strings.TrimSpace(strings.Join(parseSrc(dedent(src)), "\n")) + "\n"
	expected = strings.TrimSpace(dedent(expected)) + "\n"
	expectEqual(t, expected, actual, "")
}

func runSubtest(t *testing.T, name string, testMethod func(t *testing.T)) {
	if testMethod == nil {
		return
	}

	onBefore := func(t *testing.T) {
		log.Printf("=== TEST : %s/%s", t.Name(), name)
	}
	onAfter := func(t *testing.T, passed bool) {
		if passed {
			log.Printf("PASS: %s/%s", t.Name(), name)
		} else {
			log.Printf("FAIL: %s/%s", t.Name(), name)
		}
	}

	onBefore(t)
	passed := t.Run(name, func(t *testing.T) {
		testMethod(t)
	})
	onAfter(t, passed)
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

func setupLogger() func() {
	f, err := os.OpenFile("/home/willy/bashscript.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	check(err)
	log.SetOutput(f)
	log.SetFlags(0)
	log.SetPrefix(time.Now().Local().Format("[15:04:05.000]") + " [bctils] ")

	return func() {
		check(f.Close())
	}
}
