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
	CompleteType     string
	ClosureName      string
}

type BaseOptions []Argument
type OptionList []Argument

type PositionalList struct {
	Parser      string
	ParserClean string
	Items       []Argument
}

type Config struct {
	SourceIncludes []string
}

type templateData struct {
	CliName         string
	CliNameClean    string
	Config          Config
	PositionalsBase []Argument
	OptionsBase     BaseOptions
	Positionals     map[string]PositionalList
	Options         map[string]OptionList
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
		Positionals:     make(map[string]PositionalList, 0),
		Options:         make(map[string]OptionList, 0),
		ParserNames:     make(map[string]string),
		CliNameClean:    cliNameClean,
		CliName:         cliName,
		Config: Config{
			SourceIncludes: make([]string, 0),
		},
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
			if strings.HasPrefix(word, "--choices") && arg.argType == "pos" {
				choices := chompQuotes(strings.Split(word, "=")[1])
				arg.ValueChoices = strings.Fields(choices)
				arg.CompleteType = "choices"
			}

			// --closure
			if strings.HasPrefix(word, "--closure") && arg.argType == "pos" {
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

			if entry, ok := data.Positionals[arg.Parser]; ok {
				entry.Items = append(entry.Items, arg)
				data.Positionals[arg.Parser] = entry
			}

			if arg.Parser == "base" {
				data.PositionalsBase = append(data.PositionalsBase, arg)
			}
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	for k := range data.Options {
		if len(data.Options[k]) == 0 {
			delete(data.Options, k)
		}
	}
	for k := range data.Positionals {
		if len(data.Positionals[k].Items) == 0 {
			delete(data.Positionals, k)
		}
	}

	log.Printf("options: %v", data.Options)
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

	tmpl, err := template.New("bctil-compile").Funcs(
		template.FuncMap{"StringsJoin": strings.Join, "BashArray": BashArray, "BashAssoc": BashAssoc},
	).Parse(string(completeTemplateNew))
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

func (options OptionList) OptionNames() []string {
	values := make([]string, 0)
	for _, option := range options {
		if option.argType == "opt" {
			values = append(values, option.ArgName)
		}
	}
	return values
}

func BashAssoc(assoc map[string]string, indent int) string {
	maxLength := 80
	arrayLines := make([]string, 0)
	indentStr := strings.Repeat(" ", indent)

	line := ""
	for key, value := range assoc {
		concatStr := "[" + key + "]=\"" + value + "\""
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
			arrayLines[i] = indentStr + arrayLines[i]
		}
		return "(\n" + arrayLines[0] + "\n" + indentStr + ")"
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
			arrayLines[i] = indentStr + arrayLines[i]
		}
		return "(\n" + arrayLines[0] + "\n" + indentStr + ")"
	}
}
