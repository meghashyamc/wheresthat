package handlers

import (
	"net/http"
	"path/filepath"
)

var defaultTestRequestHeaders = map[string]string{"Content-Type": "application/json"}
var tempDir = "./.wheresthat_test"

var testFiles = map[string]string{
	"file1.txt":              "This is test content for file1",
	"file2.go":               "package main\n\nfunc main() {\n\tprint(\"Hello\")\n}",
	"subdir/file3.md":        "# Test Markdown\n\nThis is a test markdown file",
	"subdir/file4.json":      `{"key": "value", "number": 42}`,
	"subdir/nested/file5.py": "def hello():\n    print('Hello World')",
}

var createIndexHandlerTestCases = []testCase{
	{
		name:           "NoRequestBody",
		requestHeaders: defaultTestRequestHeaders,
		requestBody:    nil,
		expectedStatus: http.StatusUnprocessableEntity,
	},
	{
		name:           "EmptyPath",
		requestHeaders: defaultTestRequestHeaders,
		requestBody:    map[string]any{"path": ""},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "NonExistentPath",
		requestHeaders: defaultTestRequestHeaders,
		requestBody:    map[string]any{"path": "./abc"},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "Success",
		requestHeaders: defaultTestRequestHeaders,
		requestBody:    map[string]any{"path": mustGetAbsolutePath(tempDir)},
		expectedStatus: http.StatusNoContent,
	}}

func mustGetAbsolutePath(relativePath string) string {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		panic(err)
	}
	return absPath
}
