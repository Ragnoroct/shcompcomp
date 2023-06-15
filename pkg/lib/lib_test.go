package lib

import (
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"os"
	"path"
	"testing"
)

var loggerCleanup func()

func TestSuite(t *testing.T) {
	suite.Run(t, new(LibTestSuite))
}

type LibTestSuite struct {
	suite.Suite
	tmpdir string
}

func (suite *LibTestSuite) SetupSuite() {
	loggerCleanup = SetupLogger()
	log.Printf("RUNNING TESTS")
}

func (suite *LibTestSuite) SetupTest() {
	suite.tmpdir = ""
}

func (suite *LibTestSuite) SetupSubTest() {
	suite.tmpdir = ""
}

func (suite *LibTestSuite) TearDownSuite() {
	defer loggerCleanup()
}

func (suite *LibTestSuite) CreateFile(filename string, contents string) (filepath string) {
	if suite.tmpdir == "" {
		suite.tmpdir = suite.T().TempDir()
	}

	filepath = path.Join(suite.tmpdir, filename)
	contents = Dedent(contents)
	err := os.WriteFile(filepath, []byte(contents), 0644)
	if err != nil {
		panic(err)
	}

	return filepath
}

func (suite *LibTestSuite) TestParseWords() {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"one two three", "one two three", []string{"one", "two", "three"}},
		{"escape unquoted double quote", `one \"`, []string{"one", "\""}},
		{"escape quoted double quote", `one "b\"b"`, []string{"one", "b\"b"}},
		{"escape unquoted single quote", `one \'`, []string{"one", "'"}},
		{"escaped doublequote inside single quote", `'"one"'`, []string{`"one"`}},
		{"escaped space", `one tw\ o three`, []string{`one`, `tw o`, `three`}},
		{"escaped \\", `one tw\\o three`, []string{`one`, `tw\o`, `three`}},
		{"many spaces", `one       three`, []string{`one`, `three`}},
		{"quoted spaces", `one "  two  " three`, []string{`one`, `  two  `, `three`}},
		{"many spaces 2", `one   two   three`, []string{`one`, `two`, `three`}},
		{"merge noquoted and quoted", `one"two"three`, []string{`onetwothree`}},
		{"merge noquoted and quoted ' and \"", `one"two"'three'`, []string{`onetwothree`}},
		{"= with quoted value", `one"two" --opt='three'`, []string{`onetwo`, `--opt=three`}},
		{"= without quoted value", `one"two" --opt=three`, []string{`onetwo`, `--opt=three`}},
		{"mixing tabs and spaces", "one\t two\t \tthree", []string{`one`, `two`, `three`}},
		{"quoted tabs and spaces", "one \"\t two\t \t\" three", []string{`one`, "\t two\t \t", `three`}},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.Assert().Equal(tt.expect, parseWords(tt.input))
		})
	}
}
