package main

import (
	"bufio"
	_ "embed"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

//go:embed complete-template.txt
var completeTemplate string

type Argument struct {
	argType          string
	ArgName          string
	Parser           string
	PositionalNumber int
	ValueChoices     []string
}

type BaseOptions []Argument

type templateData struct {
	CliName         string
	CliNameClean    string
	PositionalsBase []Argument
	OptionsBase     BaseOptions
	Positionals     map[string][]Argument
	Options         map[string][]Argument
	ParserNames     map[string]string
}

func main() {
	// logger
	f, err := os.OpenFile("/home/willy/mybash.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	check(err)
	defer func(f *os.File) {
		err := f.Close()
		check(err)
	}(f)
	log.SetOutput(f)
	log.SetFlags(0)
	log.SetPrefix(time.Now().Local().Format("[15:04:05.000]") + " [bctils] ")

	cliName := os.Args[1]
	cliNameClean := regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(cliName, "")

	data := templateData{
		OptionsBase:     make([]Argument, 0),
		PositionalsBase: make([]Argument, 0),
		Positionals:     make(map[string][]Argument, 0),
		Options:         make(map[string][]Argument, 0),
		ParserNames:     make(map[string]string),
		CliNameClean:    cliNameClean,
		CliName:         cliName,
	}

	positionalCounter := map[string]int{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		argLine := scanner.Text()

		// https://stackoverflow.com/a/47489825
		quoted := false
		words := strings.FieldsFunc(argLine, func(r rune) bool {
			if r == '"' {
				quoted = !quoted
			}
			return !quoted && r == ' '
		})

		arg := Argument{}
		arg.argType = words[0]
		arg.ArgName = chompQuotes(words[1])
		arg.Parser = "base" // by default base Parser

		for _, word := range words {
			// --choices
			if strings.HasPrefix(word, "--choices") && arg.argType == "pos" {
				choices := chompQuotes(strings.Split(word, "=")[1])
				arg.ValueChoices = strings.Fields(choices)
			}

			// -p (Parser)
			if strings.HasPrefix(word, "-p") {
				arg.Parser = chompQuotes(strings.Split(word, "=")[1])
			}
		}

		if _, ok := data.Positionals[arg.Parser]; !ok {
			data.Positionals[arg.Parser] = make([]Argument, 0)
		}

		if _, ok := data.ParserNames[arg.Parser]; !ok {
			data.ParserNames[arg.Parser] = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(arg.Parser, "_")
		}

		if arg.argType == "opt" {
			// options
			if arg.Parser == "base" {
				data.OptionsBase = append(data.OptionsBase, arg)
			}
			data.Options[arg.Parser] = append(data.Options[arg.Parser], arg)
		} else if arg.argType == "pos" {
			// positionals
			if _, ok := positionalCounter[arg.Parser]; !ok {
				positionalCounter[arg.Parser] = 0
			}

			positionalCounter[arg.Parser] = positionalCounter[arg.Parser] + 1
			arg.PositionalNumber = positionalCounter[arg.Parser]
			data.Positionals[arg.Parser] = append(data.Positionals[arg.Parser], arg)

			if arg.Parser == "base" {
				data.PositionalsBase = append(data.PositionalsBase, arg)
			}
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	log.Printf("positionals: %v", data.Positionals)

	// new template feature '\}}' chomps next newline rather than trim all whitespace '-}}'
	pattern := regexp.MustCompile(`(^|\n)([\t\r ]+)(\{\{.*)\\(}}[\t\r ]*)\n(.*)($|\n)`)
	completeTemplateNew := pattern.ReplaceAllFunc([]byte(completeTemplate), func(matched []byte) []byte {
		match := pattern.FindStringSubmatch(string(matched))
		matchStart := match[1]
		matchStartWhitespace := match[2]
		matchAction := match[3] + match[4]
		matchNextLine := match[5] + match[6]
		nextLine, _ := strings.CutPrefix(matchNextLine, matchStartWhitespace)
		replaceStr := matchStart + matchStartWhitespace + matchAction + nextLine
		return []byte(replaceStr)
	})
	check(os.WriteFile("/home/willy/.dotfiles/bashcompletils/compile/complete-template.txt", completeTemplateNew, 0644))

	tmpl, err := template.New("bctil-compile").Parse(string(completeTemplateNew))
	tmpl = tmpl.Funcs(template.FuncMap{"StringsJoin": strings.Join})
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (options BaseOptions) Join() string {
	var sep string
	if sep == "" {
		sep = ", "
	}
	var strs []string
	for _, opt := range options {
		strs = append(strs, opt.ArgName)
	}
	return strings.Join(strs, sep)
}

func chompQuotes(str string) string {
	var found bool
	if str, found = strings.CutPrefix(str, "'"); found {
		str, _ = strings.CutSuffix(str, "'")
	} else if str, found = strings.CutPrefix(str, "\""); found {
		str, _ = strings.CutSuffix(str, "\"")
	}
	return str
}

func filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
