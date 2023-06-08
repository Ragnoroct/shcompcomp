package main

import (
	"bctils/pkg/lib"
	"bctils/pkg/testutil"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
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

func TestCompleteResponses(t *testing.T) {
	ctx := TestContext{t: t}
	ctx.Run("simple output", func(ctx TestContext) {
		shell := testutil.ParseOperationsStdinHelper(`
			cfg cli_name=bobman
			opt "-h"
			opt "--help"
		`)
		testutil.ExpectComplete(ctx.t, shell, "bobman ", "-h --help")
		testutil.ExpectComplete(ctx.t, shell, "bobman --he", "--help")
		testutil.ExpectComplete(ctx.t, shell, "bobman -h", "")
	})
}

func TestMainCalls(t *testing.T) {
	ctx := TestContext{t: t}
	ctx.Run("outputs to stdout", func(ctx TestContext) {
		stdout := mainWithStdout(
			[]string{"bctils", "-"},
			`
			cfg cli_name=bobman
			cfg outfile=-
			opt "-h"
			opt "--help"
			`,
		)
		if len(stdout) == 0 {
			ctx.t.Fatalf("stdout from calling main produced no output")
		}
	})
	ctx.Run("sort results by pos -> --help option", nil)
}

func TestEndToEndAutoGenWithReload(t *testing.T) {
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
		[]string{"-"},
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

	// todo: test that it only reloads when changes are made to trigger files
	testutil.ExpectCompleteFile(t, completeFile, "bobman ", "--awesome")

	writeFile("cmd.py", `
		from argparse import ArgumentParser
		parser = ArgumentParser()
		parser.add_argument("--awesome-times-infinity")
	`)

	testutil.ExpectCompleteFile(t, completeFile, "bobman ", "--awesome-times-infinity")
}

func mainWithStdout(args []string, stdin string) (stdout string) {
	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	var stdinReader bytes.Buffer
	stdinReader.WriteString(stdin)
	entry(&stdinReader, &stdoutWriter, &stderrWriter, Options{checkReload: false, args: args})
	return stdoutWriter.String()
}
