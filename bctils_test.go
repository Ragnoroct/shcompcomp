package main

import (
	"bctils/pkg/lib"
	"bctils/pkg/testutil"
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
}

func TestEndToEndAutoGenWithReload(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile := func(filename string, value string) {
		value = testutil.Dedent(value)
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

	mainWithStdout(
		[]string{"bctils", "-"},
		fmt.Sprintf(
			`
			cfg autogen_lang=py
			cfg autogen_closure_func=cmd_autogen_piper
			cfg autogen_closure_source=%s
			cfg autogen_reload_trigger=%s
			cfg outfile=%s
			opt "-h"
			opt "--help"
			`,
			path.Join(tmpDir, "cmd.bash"),
			path.Join(tmpDir, "cmd.py"),
			path.Join(tmpDir, "cmdlib.sh"),
		),
	)
}

func mainWithStdout(args []string, stdin string) (stdout string) {
	oldArgs := os.Args
	cleanupArgs := func() {
		os.Args = oldArgs
	}
	os.Args = args
	cleanupStdin := testutil.MockOsStdin(stdin)
	cleanupStdout, stdoutMock := testutil.MockOsStdout()
	defer cleanupStdout()
	defer cleanupStdin()
	defer cleanupArgs()
	os.Args = []string{"bctils", "-"}
	main()
	return stdoutMock.GetString()
}
