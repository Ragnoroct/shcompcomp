package main

import (
	generators "bctils/generators"
	"bufio"
	_ "embed"
	"flag"
	"fmt"
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
	CompleteType     string
	ClosureName      string
}

type OptionList struct {
	Parser      string
	ParserClean string
	Items       []Argument
}

type PositionalList struct {
	Parser      string
	ParserClean string
	Items       []Argument
}

type Config struct {
	SourceIncludes []string
}

type templateData struct {
	CliName      string
	CliNameClean string
	Config       Config
	Positionals  map[string]PositionalList
	Options      map[string]OptionList
	ParserNames  map[string]string
}

type cliFlags struct {
	autogenSrcFile string
	autogenLang    string
}

func main() {
	logCleanup := setupLogger()
	defer logCleanup()

	var flags = cliFlags{}
	flag.StringVar(&flags.autogenSrcFile, "autogen-src", "", "file to generate completion for")
	flag.StringVar(&flags.autogenLang, "autogen-lang", "", "language of file")
	flag.Parse()
	cliName := flag.Arg(0)

	var operationsStr string
	if flags.autogenLang == "py" {
		operationsStr = generators.GeneratePythonOperations(flags.autogenSrcFile)
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

	parseOperations(cliName, operationsStr)
}

func parseOperations(cliName string, operationsStr string) {
	cliNameClean := regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(cliName, "")

	data := templateData{
		Positionals:  make(map[string]PositionalList, 0),
		Options:      make(map[string]OptionList, 0),
		ParserNames:  make(map[string]string),
		CliNameClean: cliNameClean,
		CliName:      cliName,
		Config: Config{
			SourceIncludes: make([]string, 0),
		},
	}

	positionalCounter := map[string]int{}

	lines := strings.Split(operationsStr, "\n")
	scanner := bufio.NewScanner(os.Stdin)
	for _, argLine := range lines {
		if strings.TrimSpace(argLine) == "" {
			continue
		}

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

		// -p=parser
		if strings.HasPrefix(words[1], "-p=") {
			arg.Parser = chompQuotes(strings.Split(words[1], "=")[1])
			words = append(words[:1], words[1+1:]...)
		} else {
			arg.Parser = "base" // by default base Parser
		}

		if arg.argType == "opt" {
			arg.ArgName = chompQuotes(words[1])
		} else if arg.argType == "pos" {
			arg.ArgName = ""
		} else if arg.argType == "cfg" {
			keyValue := strings.Split(chompQuotes(words[1]), "=")
			if keyValue[0] == "INCLUDE_SOURCE" {
				data.Config.SourceIncludes = append(data.Config.SourceIncludes, keyValue[1])
			} else {
				panic("unknown config key " + keyValue[0])
			}
		} else {
			panic("unknown add operation " + arg.argType)
		}
		arg.CompleteType = ""

		if arg.argType == "pos" && strings.HasPrefix(arg.ArgName, "-") {
			panic("invalid pos " + arg.ArgName)
		}

		for _, word := range words {
			// --choices
			if strings.HasPrefix(word, "--choices") {
				choices := chompQuotes(strings.Split(word, "=")[1])
				arg.ValueChoices = strings.Fields(choices)
				arg.CompleteType = "choices"
			}

			// --closure
			if strings.HasPrefix(word, "--closure") {
				closureName := chompQuotes(strings.Split(word, "=")[1])
				arg.CompleteType = "closure"
				arg.ClosureName = closureName
			}
		}

		if _, ok := data.Positionals[arg.Parser]; !ok {
			data.Positionals[arg.Parser] = PositionalList{
				Parser:      arg.Parser,
				ParserClean: regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(arg.Parser, "_"),
			}
		}
		if _, ok := data.Options[arg.Parser]; !ok {
			data.Options[arg.Parser] = OptionList{
				Parser:      arg.Parser,
				ParserClean: regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(arg.Parser, "_"),
			}
		}

		if _, ok := data.ParserNames[arg.Parser]; !ok {
			data.ParserNames[arg.Parser] = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(arg.Parser, "_")
		}

		if arg.argType == "opt" {
			// options
			if entryOption, ok := data.Options[arg.Parser]; ok {
				entryOption.Items = append(entryOption.Items, arg)
				data.Options[arg.Parser] = entryOption
			}
		} else if arg.argType == "pos" {
			// positionals
			if _, ok := positionalCounter[arg.Parser]; !ok {
				positionalCounter[arg.Parser] = 0
			}

			positionalCounter[arg.Parser] = positionalCounter[arg.Parser] + 1
			arg.PositionalNumber = positionalCounter[arg.Parser]

			if entry, ok := data.Positionals[arg.Parser]; ok {
				entry.Items = append(entry.Items, arg)
				data.Positionals[arg.Parser] = entry
			}
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	for k := range data.Options {
		if len(data.Options[k].Items) == 0 {
			delete(data.Options, k)
		}
	}
	for k := range data.Positionals {
		if len(data.Positionals[k].Items) == 0 {
			delete(data.Positionals, k)
		}
	}

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

	tmpl, err := template.New("bctil-compile").Funcs(
		template.FuncMap{
			"StringsJoin":      strings.Join,
			"BashArray":        BashArray,
			"BashAssocQuote":   BashAssocQuote,
			"BashAssocNoQuote": BashAssocNoQuote,
			"BashAssoc":        BashAssoc,
		},
	).Parse(string(completeTemplateNew))
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}

func setupLogger() func() {
	f, err := os.OpenFile("/home/willy/mybash.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	check(err)
	log.SetOutput(f)
	log.SetFlags(0)
	log.SetPrefix(time.Now().Local().Format("[15:04:05.000]") + " [bctils] ")

	return func() {
		check(f.Close())
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
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

func (options OptionList) OptionNames() []string {
	values := make([]string, 0)
	for _, option := range options.Items {
		values = append(values, option.ArgName)
	}
	return values
}

func (options OptionList) OptionsDataAssoc() map[string]string {
	assoc := make(map[string]string, 0)
	for _, arg := range options.Items {
		if arg.CompleteType != "" {
			assoc["__type__,"+arg.ArgName] = arg.CompleteType
			if arg.CompleteType == "choices" {
				assoc["__value__,"+arg.ArgName] = strings.Join(arg.ValueChoices, " ")
			} else if arg.CompleteType == "closure" {
				assoc["__value__,"+arg.ArgName] = arg.ClosureName
			}
		}
	}

	return assoc
}

func BashAssocQuote(assoc map[string]string, indent int) string {
	return BashAssoc(assoc, indent, true)
}

func BashAssocNoQuote(assoc map[string]string, indent int) string {
	return BashAssoc(assoc, indent, false)
}

// BashAssoc todo: sorted if possible and prettier
func BashAssoc(assoc map[string]string, indent int, quoteKey bool) string {
	maxLength := 80
	arrayLines := make([]string, 0)
	indentStr := strings.Repeat(" ", indent)
	var quoteStr = ""
	if quoteKey {
		quoteStr = "\""
	}

	line := ""
	for key, value := range assoc {
		concatStr := "[" + quoteStr + key + quoteStr + "]=\"" + value + "\""
		if len(line) == 0 {
			line = concatStr
		} else if len(line)+len(concatStr)+1 > maxLength {
			arrayLines = append(arrayLines, line)
			line = concatStr
		} else {
			line = line + " " + concatStr
		}
	}

	if len(line) > 0 {
		arrayLines = append(arrayLines, line)
	}

	if len(arrayLines) == 0 {
		return "()"
	} else if len(arrayLines) == 1 {
		return "(" + arrayLines[0] + ")"
	} else {
		for i := range arrayLines {
			arrayLines[i] = indentStr + indentStr + arrayLines[i]
		}
		return "(\n" + strings.Join(arrayLines, "\n") + "\n" + indentStr + ")"
	}
}

func BashArray(values []string, indent int) string {
	maxLength := 80
	arrayLines := make([]string, 0)
	indentStr := strings.Repeat(" ", indent)

	line := ""
	for _, value := range values {
		concatStr := "\"" + value + "\""
		if len(line) == 0 {
			line = concatStr
		} else if len(line)+len(concatStr)+1 > maxLength {
			arrayLines = append(arrayLines, line)
			line = concatStr
		} else {
			line = line + " " + concatStr
		}
	}

	if len(line) > 0 {
		arrayLines = append(arrayLines, line)
	}

	if len(arrayLines) == 0 {
		return "()"
	} else if len(arrayLines) == 1 {
		return "(" + arrayLines[0] + ")"
	} else {
		for i := range arrayLines {
			arrayLines[i] = indentStr + indentStr + arrayLines[i]
		}
		return "(\n" + strings.Join(arrayLines, "\n") + "\n" + indentStr + ")"
	}
}
