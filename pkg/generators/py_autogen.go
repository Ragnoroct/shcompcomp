package generators

import (
	"bctils/pkg/lib"
	"context"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var lang *sitter.Language

func GeneratePythonOperations2(cli lib.Cli) lib.Cli {
	var operations = cli.Operations

	if cli.Config.AutogenClosureFunc != "" {
		src := callBashClosureFunc(cli.Config.AutogenClosureSource, cli.Config.AutogenClosureFunc)
		operations = append(operations, parseSrc(src)...)
	} else if cli.Config.AutogenClosureCmd != "" {
		src := runCmd(cli.Config.AutogenClosureCmd)
		operations = append(operations, parseSrc(src)...)
	}

	// strip int operations
	var newOperations []string
	for _, op := range operations {
		if strings.HasPrefix(op, "int ") {
			continue
		}
		newOperations = append(newOperations, op)
	}
	operations = newOperations

	cli = lib.ParseOperations(strings.Join(operations, "\n"))
	return cli
}

func CheckReload(stdin io.Reader, stdout io.Writer, stderr io.Writer) bool {
	content, err := io.ReadAll(stdin)
	lib.Check(err)
	cli := lib.ParseOperations(string(content))
	shouldReload := false
	for _, triggerFile := range cli.Config.AutogenReloadTriggers {
		fileInfo, _ := os.Stat(triggerFile.File)
		if triggerFile.Timestamp != fileInfo.ModTime().UnixMilli() {
			shouldReload = true
			if cli.Config.AutogenLang == "py" {
				cli = GeneratePythonOperations2(cli)
			}
			compiledShell, err := lib.CompileCli(cli)
			if err != nil {
				panic(err)
			}
			err = os.WriteFile(cli.Config.Outfile, []byte(compiledShell), 0644)
			if err != nil {
				panic(err)
			}
			break
		}
	}
	return shouldReload
}

func GeneratePythonOperations(srcFile string, argsVerbatim string, outFile string, extraWatchFiles []string) string {
	var content []byte
	var err error

	if srcFile == "-" {
		content, err = io.ReadAll(os.Stdin)
		check(err)
	} else {
		content, err = os.ReadFile(srcFile)
		if err != nil {
			fmt.Println("failed to read file: " + srcFile)
			os.Exit(1)
		}
	}

	operations := parseSrc(string(content))
	operations = append(operations, fmt.Sprintf(`cfg AUTOGEN_ARGS_VERBATIM="%s"`, argsVerbatim))
	operations = append(operations, fmt.Sprintf(`cfg AUTOGEN_OUTFILE="%s"`, outFile))
	operations = append(operations, fmt.Sprintf(`cfg RELOAD_FILE_TRIGGER="%s"`, srcFile))
	for _, file := range extraWatchFiles {
		operations = append(operations, fmt.Sprintf(`cfg RELOAD_FILE_TRIGGER="%s"`, file))
	}
	return strings.Join(operations, "\n") + "\n"
}

func parseSrc(srcStr string) []string {
	lang = python.GetLanguage()
	src := []byte(srcStr)
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree, _ := parser.ParseCtx(context.Background(), nil, src)
	root := tree.RootNode()

	parserVarName := getParserVarName(root, src)

	return getArgumentOperations(root, pyIdentifier(parserVarName), src)
}

type pyIdentifier string

type pyArguments struct {
	args   []interface{}
	kwargs map[string]interface{}
}

type pyAddArgumentCall struct {
	args pyArguments
}

type pyParser struct {
	parserIdentifier    pyIdentifier
	parserName          string
	parserParent        *pyParser
	subParsersIdentifer pyIdentifier
	subParserList       []*pyParser
	addArgumentCalls    []pyAddArgumentCall
}

type pyArgumentParserGraph struct {
	parserSequence    []*pyParser
	parsers           map[pyIdentifier]*pyParser
	subparsersParents map[pyIdentifier]*pyParser
}

func getArgumentOperations(root *sitter.Node, pyBaseParser pyIdentifier, src []byte) []string {
	patternArgumentParser := `(
		(call function: (attribute object: (identifier))) @parser-method-call
	)`

	var callGraph = pyArgumentParserGraph{
		parsers:           map[pyIdentifier]*pyParser{},
		subparsersParents: map[pyIdentifier]*pyParser{},
	}
	baseParser := &pyParser{
		parserIdentifier:    pyBaseParser,
		parserName:          "",
		subParsersIdentifer: "",
		subParserList:       []*pyParser{},
		addArgumentCalls:    []pyAddArgumentCall{},
	}
	callGraph.parsers[pyBaseParser] = baseParser
	callGraph.parserSequence = append(callGraph.parserSequence, baseParser)

	q, err := sitter.NewQuery([]byte(patternArgumentParser), lang)
	check(err)
	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			var node *sitter.Node
			var assignmentIdentifier pyIdentifier = ""

			callNode := c.Node
			functionNode := callNode.ChildByFieldName("function")
			callObjectIdentifier := pyIdentifier(functionNode.ChildByFieldName("object").Content(src))
			callFuncName := functionNode.ChildByFieldName("attribute").Content(src)

			switch callFuncName {
			case "add_parser":
			case "add_argument":
			case "add_subparsers":
			default:
				continue
			}

			callArguments := getPyArguments(callNode, src)

			if node = callNode.Parent().ChildByFieldName("left"); node != nil {
				assignmentIdentifier = pyIdentifier(node.Content(src))
			}

			if parentParser, ok := callGraph.subparsersParents[callObjectIdentifier]; ok {
				switch callFuncName {
				case "add_parser":
					var parserName string
					if len(callArguments.args) == 0 {
						panic("parser name not provided in args")
					}
					if str, ok := callArguments.args[0].(string); ok {
						parserName = str
					} else {
						panic("parser name is not a string")
					}

					// add new parser
					parserIdentifier := assignmentIdentifier
					newParser := pyParser{
						parserIdentifier: parserIdentifier,
						parserName:       parserName,
						parserParent:     parentParser,
						subParserList:    []*pyParser{},
						addArgumentCalls: []pyAddArgumentCall{},
					}
					callGraph.parsers[assignmentIdentifier] = &newParser
					parentParser.subParserList = append(parentParser.subParserList, &newParser)
					callGraph.parserSequence = append(callGraph.parserSequence, &newParser)
					callGraph.subparsersParents[callObjectIdentifier] = parentParser
				}
			}

			if parser, ok := callGraph.parsers[callObjectIdentifier]; ok {
				switch callFuncName {
				case "add_subparsers":
					if assignmentIdentifier != "" {
						callGraph.subparsersParents[assignmentIdentifier] = parser
						parser.subParsersIdentifer = assignmentIdentifier
					}
				case "add_argument":
					parser.addArgumentCalls = append(parser.addArgumentCalls, pyAddArgumentCall{
						args: callArguments,
					})
				}
				callGraph.parsers[callObjectIdentifier] = parser
			}
		}
	}

	var operations []string
	for _, parser := range callGraph.parserSequence {
		// add --choices for subparser names
		subparserNames := make([]string, len(parser.subParserList))
		for i, subparser := range parser.subParserList {
			subparserNames[i] = subparser.parserName
		}
		if len(subparserNames) > 0 {
			var operation []string
			operation = append(operation, "pos")
			if parser.parserName != "" {
				parsersString := parser.parserName
				for {
					parentParser := parser.parserParent
					if parentParser == nil || parentParser.parserName == "" {
						break
					}
					parsersString = parentParser.parserName + "." + parsersString
				}
				operation = append(operation, fmt.Sprintf(`-p="%s"`, parsersString))
			}
			operation = append(operation, fmt.Sprintf(`--choices="%s"`, strings.Join(subparserNames, " ")))
			operations = append(operations, strings.Join(operation, " "))
		}

		for _, addArgumentCall := range parser.addArgumentCalls {
			if addArgumentCall.args.Empty() {
				panic("zero arguments in add_argument call")
			}

			var operation []string
			var argumentName string
			args := addArgumentCall.args.args
			kwargs := addArgumentCall.args.kwargs

			if str, ok := args[0].(string); ok {
				argumentName = str
			} else {
				panic("add_argument first arg is not a string")
			}

			if strings.HasPrefix(argumentName, "-") {
				operation = append(operation, "opt")
			} else {
				operation = append(operation, "pos")
			}

			if parser.parserName != "" {
				parsersString := parser.parserName
				currentParserIter := parser
				for {
					parentParser := currentParserIter.parserParent
					if parentParser == nil || parentParser.parserName == "" {
						break
					}
					parsersString = parentParser.parserName + "." + parsersString
					currentParserIter = parentParser
				}
				operation = append(operation, fmt.Sprintf(`-p="%s"`, parsersString))
			}

			if strings.HasPrefix(argumentName, "-") {
				operation = append(operation, fmt.Sprintf(`"%s"`, argumentName))
			}

			if choices, ok := kwargs["choices"]; ok {
				choicesStr := ""
				if choicesList, ok := choices.([]interface{}); ok {
					for i, choice := range choicesList {
						if i != 0 {
							choicesStr += " "
						}
						switch v := choice.(type) {
						case string:
							choicesStr += v
						default:
							panic(fmt.Sprintf("cannot handle choice that isn't a string : %T", v))
						}
					}
					operation = append(operation, fmt.Sprintf(`--choices="%s"`, choicesStr))
				} else {
					panic("cannot handle choices that isnt' a list")
				}
			}

			operations = append(operations, strings.Join(operation, " "))
		}

	}

	return operations
}

func getPyArguments(callNode *sitter.Node, src []byte) pyArguments {
	var parseArgNode func(argNode *sitter.Node) interface{}
	var argumentsNode *sitter.Node

	pyArgs := pyArguments{
		args:   []interface{}{},
		kwargs: map[string]interface{}{},
	}

	if argumentsNode = callNode.ChildByFieldName("arguments"); argumentsNode == nil {
		return pyArgs
	}

	panicNode := func(n *sitter.Node, msg string) {
		lines := strings.Split(string(src), "\n")
		nodeLine := fmt.Sprintf("line %d: %s", n.StartPoint().Row, lines[n.StartPoint().Row])
		newMsg := msg + "\n" + nodeLine
		panic(newMsg)
	}

	parseArgNode = func(argNode *sitter.Node) interface{} {
		var value interface{}
		switch argNode.Type() {
		case "string":
			value = unquote(argNode.Content(src))
		case "list":
			list := make([]interface{}, argNode.NamedChildCount())
			for i := 0; i < int(argNode.NamedChildCount()); i++ {
				list[i] = parseArgNode(argNode.NamedChild(i))
			}
			value = list
		case "true":
			value = true
		case "false":
			value = false
		case "none":
			value = nil
		default:
			panicNode(argNode, fmt.Sprintf("unhandled Node.Type() '%s'", argNode.Type()))
		}
		return value
	}

	for i := 0; i < int(argumentsNode.NamedChildCount()); i++ {
		argNode := argumentsNode.NamedChild(i)
		if argNode.Type() == "keyword_argument" {
			pyKey := argNode.ChildByFieldName("name").Content(src)
			pyValue := parseArgNode(argNode.ChildByFieldName("value"))
			pyArgs.kwargs[pyKey] = pyValue
		} else {
			pyValue := parseArgNode(argNode)
			pyArgs.args = append(pyArgs.args, pyValue)
		}
	}

	return pyArgs
}

func getParserVarName(root *sitter.Node, src []byte) string {
	name := ""
	patternArgumentParser := `(
		assignment
			left: (identifier)
			right: (call
				function: (identifier) @func-name
				(#match? @func-name "ArgumentParser")
			)
	)`

	q, err := sitter.NewQuery([]byte(patternArgumentParser), lang)
	check(err)
	qc := sitter.NewQueryCursor()
	qc.Exec(q, root)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		m = qc.FilterPredicates(m, src)
		for _, c := range m.Captures {
			name = c.Node.Parent().Parent().ChildByFieldName("left").Content(src)
		}
	}

	return name
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func unquote(str string) string {
	var found bool
	if str, found = strings.CutPrefix(str, "'"); found {
		str, _ = strings.CutSuffix(str, "'")
	} else if str, found = strings.CutPrefix(str, "\""); found {
		str, _ = strings.CutSuffix(str, "\"")
	}
	return str
}

func (args pyArguments) Empty() bool {
	return len(args.args)+len(args.kwargs) == 0
}

var bashProcessRef *bashProcess

func callBashClosureFunc(closureFile string, closureName string) string {
	if bashProcessRef == nil {
		bashProcessRef = newBashProcess()
	}
	return bashProcessRef.runClosure(closureFile, closureName)
}

type bashProcess struct {
	mutex   sync.Mutex
	chanOut chan string
	stdin   io.WriteCloser
}

func (b *bashProcess) runClosure(closureFile string, closureName string) string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	_, _ = io.WriteString(b.stdin, fmt.Sprintf("%s:%s\n", closureFile, closureName))
	return <-b.chanOut
}

func runCmd(cmd string) string {
	out, err := exec.Command(cmd).Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

func newBashProcess() *bashProcess {
	var err error
	var bashOutBuffer []byte
	var chanOut = make(chan string)

	shellCode := lib.Dedent(`
		while IFS=: read -r closure_file closure_name; do
			source "$closure_file"
			"$closure_name"
			printf '\0'
		done
	`)

	proc := exec.Command("bash", "-c", shellCode)
	stdin, err := proc.StdinPipe()
	check(err)
	stdout, err := proc.StdoutPipe()
	check(err)
	err = proc.Start()
	check(err)

	go func() {
		var err error
		var n int
		buff := make([]byte, 256)
		for err == nil {
			n, err = stdout.Read(buff)
			for i := 0; i < n; i++ {
				if buff[i] == '\x00' {
					out := string(bashOutBuffer)
					bashOutBuffer = []byte{}
					chanOut <- out
				} else {
					bashOutBuffer = append(bashOutBuffer, buff[i])
				}
			}
		}
	}()

	bashProc := bashProcess{
		mutex:   sync.Mutex{},
		chanOut: chanOut,
		stdin:   stdin,
	}
	return &bashProc
}
