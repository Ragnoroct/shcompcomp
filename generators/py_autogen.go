package generators

import (
	"context"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"os"
	"strings"
)

var lang *sitter.Language

func GeneratePythonOperations(srcFile string, argsVerbatim string, outFile string, extraWatchFiles []string) string {
	content, err := os.ReadFile(srcFile)
	if err != nil {
		fmt.Println("failed to read file: " + srcFile)
		os.Exit(1)
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
				operation = append(operation, fmt.Sprintf(`-p="%s"`, parser.parserName))
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
