package generators

import (
	"context"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"log"
	"strings"
)

var lang *sitter.Language

func GeneratePython(srcFile string) {
	log.Printf("generating for: %s", srcFile)
}

func parseSrc(cliName string, srcStr string) string {
	lang = python.GetLanguage()
	src := []byte(srcStr)
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree, _ := parser.ParseCtx(context.Background(), nil, src)
	root := tree.RootNode()

	parserVarName := getParserVarName(root, src)

	operations := getArgumentOperations(root, cliName, parserVarName, src)

	return strings.Join(operations, "\n") + "\n"
}

func getArgumentOperations(root *sitter.Node, cliName string, parserVarName string, src []byte) []string {
	var operations []string
	patternArgumentParser := `(
		(call function: (attribute object: (identifier))) @parser-method-call
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
			objName := c.Node.ChildByFieldName("function").ChildByFieldName("object").Content(src)
			if objName == parserVarName {
				nodeArguments := c.Node.ChildByFieldName("arguments")
				curArgNode := nodeArguments.NamedChild(0)
				var pyArgs []string
				pyKwargs := make(map[string]interface{})
				for curArgNode != nil {
					if curArgNode.Type() == "string" {
						// args
						pyArgs = append(pyArgs, chompQuotes(curArgNode.Content(src)))
					} else if curArgNode.Type() == "keyword_argument" {
						// kwargs
						pyKey := curArgNode.ChildByFieldName("name").Content(src)
						valueNode := curArgNode.ChildByFieldName("value")
						if valueNode.Type() == "list" {
							// list of strings
							var pyList []string
							for i := 0; valueNode.NamedChild(i) != nil; i++ {
								elemNode := valueNode.NamedChild(i)
								if elemNode.Type() == "string" {
									pyList = append(pyList, chompQuotes(valueNode.NamedChild(i).Content(src)))
								} else {
									panic("not implemented for type " + elemNode.Type())
								}
							}
							pyKwargs[pyKey] = pyList
						} else {
							panic("don't know how to handle value type " + valueNode.Type())
						}
					}
					curArgNode = curArgNode.NextNamedSibling()
				}

				if strings.HasPrefix(pyArgs[0], "-") {
					// opt
					addOp := fmt.Sprintf(`bctils_cli_add %s opt "%s"`, cliName, pyArgs[0])
					operations = append(operations, addOp)
				} else {
					// pos
					addOp := fmt.Sprintf("bctils_cli_add %s pos", cliName)

					// --choices
					if _, ok := pyKwargs["choices"]; ok {
						switch v := pyKwargs["choices"].(type) {
						case []string:
							addOp += fmt.Sprintf(` --choices="%s"`, strings.Join(v, " "))
						default:
							panic(fmt.Sprintf("cannot handle type %T", v))
						}
					}

					operations = append(operations, addOp)
				}
			}
		}
	}

	return operations
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

func chompQuotes(str string) string {
	var found bool
	if str, found = strings.CutPrefix(str, "'"); found {
		str, _ = strings.CutSuffix(str, "'")
	} else if str, found = strings.CutPrefix(str, "\""); found {
		str, _ = strings.CutSuffix(str, "\"")
	}
	return str
}
