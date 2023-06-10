package main

import (
	"bufio"
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
			return 1
		} else {
			return 0
		}
	}
}

func main() {
	logCleanup := lib.SetupLogger()
	defer logCleanup()

	isLegacy := len(os.Args) > 1 && os.Args[1] == "-legacy"

	if !isLegacy {
		options := Options{}
		flag.BoolVar(&options.checkReload, "reload-check", false, "")
		flag.Parse()
		options.args = flag.Args()
		exitCode := entry(os.Stdin, os.Stdout, os.Stderr, options)
		os.Exit(exitCode)
	} else {
		// legacy: allow tests to pass while reworking
		os.Args = append(os.Args[0:1], os.Args[2:]...)
		var flags = cliFlags{}
		flag.StringVar(&flags.autogenSrcFile, "autogen-src", "", "file to generate completion for")
		flag.StringVar(&flags.autogenLang, "autogen-lang", "", "language of file")
		flag.StringVar(&flags.autogenOutfile, "autogen-outfile", "", "outfile location so it can source itself")
		flag.Var(&flags.autogenExtraWatchFiles, "autogen-extra-watch", "extra files to trigger reloads")
		flag.Parse()
		cliName := flag.Arg(0)

		var operationsStr string
		if flags.autogenLang == "py" {
			argsVerbatim := flag.Arg(1)
			operationsStr = generators.GeneratePythonOperations(flags.autogenSrcFile, argsVerbatim, flags.autogenOutfile, flags.autogenExtraWatchFiles)
		} else if flags.autogenLang == "" {
			operationsStr = ""
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				argLine := scanner.Text()
				operationsStr += argLine + "\n"
			}
		} else {
			fmt.Printf("error: unknown lang '%s' for autogen", flags.autogenLang)
			os.Exit(1)
		}

		operationsStr = "cfg cli_name=" + cliName + "\n" + operationsStr
		cli := lib.ParseOperations(operationsStr)
		compiledShell, err := lib.CompileCli(cli)
		if err != nil {
			exit(1, err)
		}

		_, err = os.Stdout.WriteString(compiledShell)
		lib.Check(err)
	}
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

		cli := lib.ParseOperations(string(content))

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
