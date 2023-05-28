package generators

import (
	"context"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"log"
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

	fmt.Println(root.String())
	parserVarName := getParserVarName(root, src)
	fmt.Printf("%s = ArgumentParser()\n", parserVarName)

	getArgumentOperations(root, parserVarName, src)

	return "bob\n"
}

func getArgumentOperations(root *sitter.Node, parserVarName string, src []byte) []string {
	patternArgumentParser := `(
		call
			function: (identifier) @func-name
			(#match? @func-name "^(add_argument)$")
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
			fmt.Printf("found match: %s", c.Node.Content(src))
		}
	}

	return make([]string, 0)
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
