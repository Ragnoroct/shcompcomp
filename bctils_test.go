package main

import (
	"bctils/testutil"
	"log"
	"os"
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

func TestEndToEnd(t *testing.T) {
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
