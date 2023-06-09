package lib

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

//go:embed complete-template.go.sh
var completeTemplate string

const (
	CompleteTypeClosure = "closure"
	CompleteTypeChoices = "choices"
	DefaultParser       = "base"
)

type CliParserName string
type CliParser struct {
	parserName  CliParserName
	positionals []CliPositional
	optionals   []CliOptional
}

type CliParsers struct {
	parserMap map[CliParserName]CliParser
	parserSeq []CliParserName
}

func (parsers *CliParsers) addPositional(pos CliPositional) {
	name := pos.parser
	if parser, ok := parsers.parserMap[name]; ok {
		pos.Number = len(parser.positionals) + 1
		parser.positionals = append(parser.positionals, pos)
		parsers.parserMap[name] = parser
	} else {
		parser = CliParser{
			parserName:  name,
			positionals: []CliPositional{},
			optionals:   []CliOptional{},
		}
		pos.Number = len(parser.positionals) + 1
		parser.positionals = append(parser.positionals, pos)
		parsers.parserMap[name] = parser
		parsers.parserSeq = append(parsers.parserSeq, name)
	}
}

func (parsers *CliParsers) addOptional(opt CliOptional) {
	name := opt.parser
	if parser, ok := parsers.parserMap[name]; ok {
		parser.optionals = append(parser.optionals, opt)
		parsers.parserMap[name] = parser
	} else {
		parser = CliParser{
			parserName:  name,
			positionals: []CliPositional{},
			optionals:   []CliOptional{},
		}
		parser.optionals = append(parser.optionals, opt)
		parsers.parserMap[name] = parser
		parsers.parserSeq = append(parsers.parserSeq, name)
	}
}

func (parsers *CliParsers) parser(name CliParserName) CliParser {
	if parser, ok := parsers.parserMap[name]; ok {
		return parser
	} else {
		parser = CliParser{
			parserName:  name,
			positionals: []CliPositional{},
			optionals:   []CliOptional{},
		}
		parsers.parserMap[name] = parser
		parsers.parserSeq = append(parsers.parserSeq, name)
		return parser
	}
}

func (parser CliParser) NameClean() string {
	return cleanShellIdentifier(string(parser.parserName))
}

func (parser CliParser) Positionals() []CliPositional {
	return parser.positionals
}

func (parser CliParser) OptionalsNames() []string {
	names := make([]string, len(parser.optionals))
	for i, optional := range parser.optionals {
		names[i] = optional.name
	}
	return names
}

func (parser CliParser) OptionalsData() map[string]string {
	assoc := make(map[string]string, 0)
	for _, optional := range parser.optionals {
		if optional.completeType != "" {
			assoc["__type__,"+optional.name] = optional.completeType
			if optional.completeType == "choices" {
				assoc["__value__,"+optional.name] = strings.Join(optional.choices, " ")
			} else if optional.completeType == "closure" {
				assoc["__value__,"+optional.name] = optional.closureName
			}
		}
	}

	return assoc
}

type CliPositional struct {
	parser       CliParserName
	Number       int
	CompleteType string
	ClosureName  string
	Choices      []string
}

type CliOptional struct {
	parser       CliParserName
	name         string
	completeType string
	closureName  string
	choices      []string
}

type ReloadTrigger struct {
	File      string
	Timestamp int64
}

func (r ReloadTrigger) String() string {
	return r.File
}

type CliConfig struct {
	Outfile               string
	AutogenLang           string
	AutogenClosureCmd     string
	AutogenClosureFunc    string
	AutogenClosureSource  string
	AutogenReloadTriggers []ReloadTrigger
	SourceIncludes        []string // deprecated
	ReloadTriggerFiles    []string // deprecated
	AutoGenArgsVerbatim   string   // deprecated
	AutoGenOutfile        string   // deprecated
}

func (c Cli) CliName() string {
	return c.cliName
}

func (c Cli) CliNameClean() string {
	return cleanShellIdentifier(c.cliName)
}

func (c Cli) OperationsComment() string {
	return "# " + strings.Join(c.Operations, "\n# ")
}

func (c Cli) OperationsReloadConfig() []string {
	configOperations := make([]string, 0)
	for _, opt := range c.Operations {
		if strings.HasPrefix(opt, "cfg ") || strings.HasPrefix(opt, "int ") {
			configOperations = append(configOperations, opt)
		}
	}
	return configOperations
}

type Cli struct {
	cliName    string
	Config     CliConfig
	Parsers    *CliParsers
	Operations []string
}

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

type templateData struct {
	ModifiedTimeMs int64
	Cli            Cli
}

func (d templateData) ParserNameMap() map[string]string {
	parserNames := make(map[string]string, len(d.Cli.Parsers.parserSeq))
	for _, name := range d.Cli.Parsers.parserSeq {
		parserNameCommaSep := strings.ReplaceAll(string(name), ".", ",")
		parserNames[parserNameCommaSep] = cleanShellIdentifier(string(name))
	}
	return parserNames
}

func (d templateData) Parsers() []CliParser {
	parsers := make([]CliParser, len(d.Cli.Parsers.parserSeq))
	for i, name := range d.Cli.Parsers.parserSeq {
		parsers[i] = d.Cli.Parsers.parserMap[name]
	}
	return parsers
}

func (d templateData) OperationsComment() string {
	return "# " + strings.Join(d.Cli.Operations, "\n# ")
}

func (d templateData) StringsJoin(values []string, indent int) string {
	if len(values) == 0 {
		return ""
	}
	indentStr := strings.Repeat(" ", indent)
	return strings.Join(values, "\n"+indentStr)
}

func ParseOperationsStdin(stdin io.Reader) string {
	content, err := io.ReadAll(stdin)
	Check(err)
	cli := ParseOperations(string(content))
	completeCode, _ := CompileCli(cli)
	return completeCode
}

func ParseOperations(operationsStr string) Cli {
	var parsers = CliParsers{
		parserMap: map[CliParserName]CliParser{},
		parserSeq: []CliParserName{},
	}

	cli := Cli{
		Config: CliConfig{Outfile: "-"},
	}
	var operationLinesParsed []string
	operationLines := strings.Split(operationsStr, "\n")
	for opIndex, opStr := range operationLines {
		opStr = strings.TrimSpace(opStr)
		if opStr == "" {
			continue
		}

		words := splitOperation(opStr)
		opType := words[0]
		var intOperations []string

		switch opType {
		case "int":
			continue
		case "cfg":
			configName, configValue, valid := strings.Cut(unquote(words[1]), "=")
			if !valid {
				panic(fmt.Sprintf("error : cfg op is invalid : %s", opStr))
			}
			configName = unquote(configName)
			configValue = unquote(configValue)
			switch configName {
			case "INCLUDE_SOURCE":
				cli.Config.SourceIncludes = append(cli.Config.SourceIncludes, configValue)
			case "RELOAD_FILE_TRIGGER":
				cli.Config.ReloadTriggerFiles = append(cli.Config.ReloadTriggerFiles, configValue)
			case "AUTOGEN_OUTFILE":
				cli.Config.AutoGenOutfile = configValue
			case "AUTOGEN_ARGS_VERBATIM":
				configValue = strings.Replace(configValue, "--source", "", 1)
				cli.Config.AutoGenArgsVerbatim = configValue
			case "cli_name":
				cli.cliName = configValue
			case "outfile":
				cli.Config.Outfile = configValue
			case "autogen_lang":
				cli.Config.AutogenLang = configValue
			case "autogen_closure_cmd":
				cli.Config.AutogenClosureCmd = configValue
			case "autogen_closure_func":
				cli.Config.AutogenClosureFunc = configValue
			case "autogen_closure_source":
				cli.Config.AutogenClosureSource = configValue
			case "autogen_reload_trigger":
				reloadTrigger := ReloadTrigger{
					File:      configValue,
					Timestamp: 0,
				}
				if len(operationLines) > opIndex+1 {
					nextOp := strings.TrimSpace(operationLines[opIndex+1])
					if strAfter, found := strings.CutPrefix(nextOp, "int autogen_reload_trigger_ts="); found {
						timestamp, _ := strconv.Atoi(strAfter)
						reloadTrigger.Timestamp = int64(timestamp)
					}
				}
				cli.Config.AutogenReloadTriggers = append(cli.Config.AutogenReloadTriggers, reloadTrigger)

				if reloadTrigger.Timestamp == 0 {
					fileInfo, err := os.Stat(configValue)
					reloadTrigger.Timestamp = fileInfo.ModTime().UnixMilli()
					if err != nil {
						panic(err)
					}
				}
				intOperations = append(intOperations, fmt.Sprintf("int autogen_reload_trigger_ts=%d", reloadTrigger.Timestamp))
			}
		case "pos":
			arg := CliPositional{}

			// -p=parser
			if strings.HasPrefix(words[1], "-p=") {
				if value, ok := tryOption(words[1], "-"); ok {
					arg.parser = CliParserName(value)
					words = append(words[:1], words[1+1:]...)
				}
			} else {
				arg.parser = DefaultParser
			}

			for _, word := range words {
				if value, ok := tryOption(word, "--choices"); ok {
					arg.CompleteType = CompleteTypeChoices
					arg.Choices = strings.Fields(value)
				}
				if value, ok := tryOption(word, "--closure"); ok {
					arg.CompleteType = CompleteTypeClosure
					arg.ClosureName = value
				}
			}

			parsers.addPositional(arg)
		case "opt":
			opt := CliOptional{}

			// -p=parser can come before name
			if strings.HasPrefix(words[1], "-p=") {
				if value, ok := tryOption(words[1], "-"); ok {
					opt.parser = CliParserName(value)
					words = append(words[:1], words[1+1:]...)
				}
			} else {
				opt.parser = DefaultParser
			}

			opt.name = unquote(words[1])

			for _, word := range words {
				if value, ok := tryOption(word, "--choices"); ok {
					opt.completeType = CompleteTypeChoices
					opt.choices = strings.Fields(value)
				}
				if value, ok := tryOption(word, "--closure"); ok {
					opt.completeType = CompleteTypeClosure
					opt.closureName = value
				}
			}

			parsers.addOptional(opt)
		default:
			panic(fmt.Sprintf("error : unknown operation : %s", opType))
		}

		operationLinesParsed = append(operationLinesParsed, opStr)
		if intOperations != nil {
			operationLinesParsed = append(operationLinesParsed, intOperations...)
		}
	}

	cli.Parsers = &parsers
	cli.Operations = operationLinesParsed
	return cli
}

func CompileCli(cli Cli) (string, error) {
	data := templateData{
		ModifiedTimeMs: time.Now().UnixMilli(),
		Cli:            cli,
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
	Check(os.WriteFile("/home/willy/.dotfiles/bashcompletils/compile/complete-template.txt", completeTemplateNew, 0644))

	templateParsed, err := template.New("bctil-compile").Funcs(
		template.FuncMap{
			"StringsJoin":      strings.Join,
			"BashArray":        BashArray,
			"BashAssocQuote":   BashAssocQuote,
			"BashAssocNoQuote": BashAssocNoQuote,
			"BashAssoc":        BashAssoc,
		},
	).Parse(string(completeTemplateNew))
	Check(err)

	var buffer bytes.Buffer
	err = templateParsed.Execute(&buffer, data)
	if err != nil {
		re := regexp.MustCompile(`bctil-compile:(\d+):(\d+)`)
		matches := re.FindStringSubmatch(err.Error())
		col, _ := strconv.Atoi(matches[2])
		return "", fmt.Errorf(
			"error in template ./complete-template.go.sh:%s:%d: \n%s",
			matches[1],
			col+1,
			err,
		)
	}

	compiledShell := buffer.String()

	// collapse multiple newlines into one
	compiledShell = regexp.MustCompile(`(?m)^\s+\n`).ReplaceAllString(compiledShell, "\n")

	return compiledShell, nil
}

func tryOption(word string, name string) (string, bool) {
	if strings.HasPrefix(word, name) {
		_, optionValue, valid := strings.Cut(word, "=")
		if valid {
			return unquote(optionValue), true
		} else {
			return "", true
		}
	} else {
		return "", false
	}
}

func cleanShellIdentifier(identifier string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(identifier, "")
}

// https://stackoverflow.com/a/47489825
func splitOperation(op string) []string {
	quoted := false
	return strings.FieldsFunc(op, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func unquote(str string) string {
	if strings.HasPrefix(str, `'`) && strings.HasSuffix(str, `'`) {
		str, _ = strings.CutPrefix(str, `'`)
		str, _ = strings.CutSuffix(str, `'`)
	} else if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
		str, _ = strings.CutPrefix(str, `"`)
		str, _ = strings.CutSuffix(str, `"`)
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

func Dedent(str string) string {
	mixingSpacesAndTabs := false
	if len(str) == 0 {
		return str
	}
	if str[0] == '\n' {
		str = str[1:]
	}
	lines := strings.Split(str, "\n")
	minIndent := -1
	for _, line := range lines {
		for i, c := range line {
			if c == ' ' {
				mixingSpacesAndTabs = true
				continue
			} else if c != '\t' {
				if minIndent == -1 || i < minIndent {
					minIndent = i
				}
				break
			}
		}
	}

	if minIndent == 0 {
		return strings.TrimSpace(str) + "\n"
	} else if mixingSpacesAndTabs {
		panic("cannot handle mixing spaces with tab")
	}

	indentStr := strings.Repeat("\t", minIndent)
	for i := range lines {
		newLine, _ := strings.CutPrefix(lines[i], indentStr)
		lines[i] = newLine
	}

	if strings.Trim(lines[len(lines)-1], " \t\n") == "" {
		lines = lines[0 : len(lines)-1]
	}

	return strings.Join(lines, "\n") + "\n"
}
