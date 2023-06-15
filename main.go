package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"shcomp2/pkg/generators"
	"shcomp2/pkg/lib"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type cliFlags struct {
	autogenLang            string
	autogenOutfile         string
	autogenSrcFile         string
	autogenExtraWatchFiles arrayFlags
}

type Options struct {
	args        []string
	checkReload bool
}

func entry(stdin io.Reader, stdout io.Writer, stderr io.Writer, options Options) (code int) {
	if options.checkReload {
		if generators.CheckReload(stdin, stdout, stderr) {
			return 5
		} else {
			return 0
		}
	} else {
		err := HandleCompileShell(options.args[0], stdin, stdout, stderr)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		} else {
			return 0
		}
	}
}

func main() {
	logCleanup := lib.SetupLogger()
	defer logCleanup()

	options := Options{}
	flag.BoolVar(&options.checkReload, "reload-check", false, "")
	flag.Parse()
	options.args = flag.Args()
	exitCode := entry(os.Stdin, os.Stdout, os.Stderr, options)
	os.Exit(exitCode)
}

func HandleCompileShell(infile string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if infile == "-" {
		content, err := io.ReadAll(stdin)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", err)
			return errors.New("unable to read stdin")
		} else if len(content) == 0 {
			_, _ = fmt.Fprintf(stderr, "stdin is empty\n")
			return errors.New("stdin is empty but infile is - ")
		}

		cli, err := lib.ParseOperations(string(content))
		if err != nil {
			return err
		}

		if cli.Config.AutogenLang == "py" {
			cli = generators.GeneratePythonOperations2(cli)
		}

		compiledShell, err := lib.CompileCli(cli)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", err)
			return errors.New("unable to compile shell")
		}
		err = lib.CommitCli(cli, compiledShell, stdout)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", err)
			return errors.New("unable to commit shell")
		}
		return nil
	} else {
		_, _ = fmt.Fprintf(stderr, "must provide - as first argument\n")
		return errors.New("infile as other files not implemented yet")
	}
}

func exit(code int, msg any) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
