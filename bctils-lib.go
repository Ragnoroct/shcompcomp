package main

import (
	"bctils/pkg/generators"
	"bctils/pkg/lib"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
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

//type mainFlags struct {
//}

func main() {
	var err error
	logCleanup := setupLogger()
	defer logCleanup()

	isLegacy := len(os.Args) > 1 && os.Args[1] == "-legacy"

	if !isLegacy {
		//var flags = mainFlags{}

		//flag.StringVar(&flags.autogenSrcFile, "autogen-src", "", "file to generate completion for")
		//flag.StringVar(&flags.autogenLang, "autogen-lang", "", "language of file")
		//flag.StringVar(&flags.autogenOutfile, "autogen-outfile", "", "outfile location so it can source itself")
		//flag.Var(&flags.autogenExtraWatchFiles, "autogen-extra-watch", "extra files to trigger reloads")

		flag.Parse()
		infile := flag.Arg(0)
		if infile == "-" {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				exit(1, err)
			} else if len(content) == 0 {
				exit(1, "stdin is empty")
			}
			cli := lib.ParseOperations(string(content))
			compiledShell, err := lib.CompileCli(cli)
			exitCheck(err, 1, "")

			if cli.Config.Outfile == "-" {
				_, err = os.Stdout.WriteString(compiledShell)
				exitCheck(err, 1, "")
			}
		}
		lib.Check(err)
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

func exitCheck(err error, code int, msg string) {
	if err != nil {
		if msg != "" {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", msg)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(code)
	}
}

func exit(code int, msg any) {
	_, _ = fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}

func setupLogger() func() {
	home, _ := os.UserHomeDir()
	f, err := os.OpenFile(path.Join(home, "bashscript.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	lib.Check(err)
	log.SetOutput(f)
	log.SetFlags(0)
	log.SetPrefix(time.Now().Local().Format("[15:04:05.000]") + " [bctils] ")

	return func() {
		lib.Check(f.Close())
	}
}
