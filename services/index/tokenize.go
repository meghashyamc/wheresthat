package index

import (
	"regexp"
	"strings"
)

func tokenize(text string) []Token {
	var tokens []Token

	// Convert to lowercase for case-insensitive search
	text = strings.ToLower(text)

	// Split into words using regex
	wordRegex := regexp.MustCompile(`\b\w+\b`)
	matches := wordRegex.FindAllStringIndex(text, -1)

	for _, match := range matches {
		word := text[match[0]:match[1]]

		// Skip very short words and common stop words
		if len(word) < 2 || isStopWord(word) {
			continue
		}

		// Calculate line and column for position tracking
		line, column := getLineColumn(text, match[0])

		token := Token{
			Text:     word,
			Position: match[0],
			Line:     line,
			Column:   column,
		}

		tokens = append(tokens, token)
	}

	return tokens
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}
	return stopWords[word]
}

func getLineColumn(text string, position int) (int, int) {
	line := 1
	column := 1

	for i, char := range text {
		if i >= position {
			break
		}
		if char == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}

	return line, column
}
