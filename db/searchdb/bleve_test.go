package searchdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var parseQuotedQueryTestCases = []struct {
	name              string
	input             string
	expectedQuoted    []string
	expectedRemaining string
}{
	{
		name:              "Simple quoted phrase",
		input:             `"hello world"`,
		expectedQuoted:    []string{"hello world"},
		expectedRemaining: "",
	},
	{
		name:              "Quoted phrase with remaining terms",
		input:             `"hello world" test golang`,
		expectedQuoted:    []string{"hello world"},
		expectedRemaining: "test golang",
	},
	{
		name:              "Multiple quoted phrases",
		input:             `"hello world" test "another phrase"`,
		expectedQuoted:    []string{"hello world", "another phrase"},
		expectedRemaining: "test",
	},
	{
		name:              "No quotes",
		input:             `hello world test`,
		expectedQuoted:    nil,
		expectedRemaining: "hello world test",
	},
	{
		name:              "Empty quoted phrase",
		input:             `"" test`,
		expectedQuoted:    nil,
		expectedRemaining: "test",
	},
	{
		name:              "Quoted phrase with extra spaces",
		input:             `"  hello world  " test`,
		expectedQuoted:    []string{"hello world"},
		expectedRemaining: "test",
	},
	{
		name:              "Multiple quoted phrases with spaces",
		input:             `  "first phrase"   test   "second phrase"  `,
		expectedQuoted:    []string{"first phrase", "second phrase"},
		expectedRemaining: "test",
	},
}

func TestParseQuotedQuery(t *testing.T) {
	assert := require.New(t)
	for _, testCase := range parseQuotedQueryTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			quoted, remaining := parseQuotedQuery(testCase.input)

			assert.Equal(quoted, testCase.expectedQuoted, "quoted phrases should match")
			assert.Equal(remaining, testCase.expectedRemaining, "remaining (not quoted) terms should match")
		})
	}
}
