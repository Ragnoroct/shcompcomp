package main

import (
	"bytes"
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

func TestSub(t *testing.T) {
	ctx := TestContext{t: t}
	ctx.Run("simple output", func(ctx TestContext) {
		var stdin bytes.Buffer
		stdin.Write([]byte(`
			cfg cli_name=boman
			opt "-h"
			opt "--help"
		`))
		result := parseOperationsStdin(&stdin)
		ctx.ExpectEqualLines("hunter2", result, "")
	})
}
