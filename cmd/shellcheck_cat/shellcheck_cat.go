// / 2>/dev/null ; gorun "$0" "$@" ; exit $?
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type comment struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Column    int    `json:"column"`
	EndColumn int    `json:"endColumn"`
	Level     string `json:"level"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Fix       string `json:"fix"`
}
type shellcheck struct {
	Comments []comment `json:"comments"`
}

func main() {
	bytes, _ := exec.Command("shellcheck", "--format=json1", os.Args[1]).Output()
	var result shellcheck
	_ = json.Unmarshal(bytes, &result)

	contents, _ := os.ReadFile(os.Args[1])
	lines := strings.Split(string(contents), "\n")
	var lineHightlights []string

	for i := len(result.Comments) - 1; i >= 0; i-- {
		cmt := result.Comments[i]
		lineIndex := cmt.EndLine
		start := "# "

		indent := strings.Repeat(" ", cmt.Column-(1+len(start)))
		var pointerLine string
		colWidth := cmt.EndColumn - cmt.Column
		if colWidth >= 3 {
			pointerLine = "^" + strings.Repeat("-", colWidth-2) + "^"
		} else if colWidth == 2 {
			pointerLine = "^^"
		} else {
			pointerLine = "^"
		}

		annotation := fmt.Sprintf("%s%s%s SC%d (%s): %s", start, indent, pointerLine, cmt.Code, cmt.Level, cmt.Message)
		annotation += "\n" + start + indent + fmt.Sprintf("https://github.com/koalaman/shellcheck/wiki/SC%d", cmt.Code)
		annotation += "\n" + start + indent + fmt.Sprintf("line: %d", cmt.EndLine)

		lines = append(lines[:lineIndex+1], lines[lineIndex:]...)
		lines[lineIndex] = annotation

		lineHightlights = append(lineHightlights, fmt.Sprintf("--highlight-line=%d:+3", cmt.EndLine+(i*3)))
	}

	content := []byte(strings.Join(lines, "\n"))
	tmpfile, err := os.CreateTemp(os.TempDir(), "batstdin")
	if err != nil {
		log.Fatal(err)
	}

	if _, err = tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}

	_, err = tmpfile.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}

	err = syscall.Dup2(int(tmpfile.Fd()), syscall.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(tmpfile.Name())
	if err != nil {
		log.Fatal(err)
	}

	binary, err := exec.LookPath("bat")
	if err != nil {
		log.Fatal(err)
	}

	args := []string{
		binary,
		"-",
		"--paging=never",
		"--color=always",
		"--file-name=" + os.Args[1],
	}
	args = append(args, os.Args[2:]...)
	args = append(args, lineHightlights...)

	if err := syscall.Exec(binary, args, []string{}); err != nil {
		log.Fatal(err)
	}
}
