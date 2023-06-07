package main

import (
	"log"
	"strings"
	"testing"
)

type TestContext struct {
	t *testing.T
}

func (ctx TestContext) ExpectEqualLines(expected string, actual string, msg string) {
	if msg == "" {
		msg = "strings are not equal"
	}
	expected = dedent(expected)
	actual = dedent(actual)
	if actual != expected {
		ctx.t.Fatalf(
			"%s\nactual:\n'''\n%s'''\nexpected:\n'''\n%s'''", msg,
			actual,
			expected,
		)
	}
}

func (ctx TestContext) Run(name string, testFunc func(ctx TestContext)) {
	if testFunc == nil {
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

	onBefore(ctx.t)
	passed := ctx.t.Run(name, func(t *testing.T) {
		ctxSubTest := TestContext{t: t}
		testFunc(ctxSubTest)
	})
	onAfter(ctx.t, passed)
}

//func (ctx TestContext) CaptureStdout()

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
