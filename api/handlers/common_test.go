// Common test helpers
package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/validation"
	"github.com/stretchr/testify/require"
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

type testCase struct {
	name             string
	requestHeaders   map[string]string
	requestBody      map[string]interface{}
	queryParams      map[string]string
	expectedStatus   int
	expectedResponse map[string]interface{}
}

func newTestLogger() logger.Logger {

	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
func setupTestServer(t *testing.T, assert *require.Assertions) (*gin.Engine, func()) {

	t.Setenv("INDEX_PATH", filepath.Join(tempDir, "index"))
	t.Setenv("KVDB_PATH", filepath.Join(tempDir, "kv.db"))
	for relPath, content := range testFiles {
		fullPath := filepath.Join(tempDir, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		assert.NoError(err, "could not create test sub-directory")
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NoError(err, "could not write test file")
	}

	testLogger := newTestLogger()

	searchDB, err := searchdb.New(testLogger)
	assert.NoError(err, "could not create search database")

	kvDB, err := kvdb.New(testLogger)
	assert.NoError(err, "could not create kv database")
	validator, err := validation.New(testLogger)
	assert.NoError(err, "could not create validator")
	gin.SetMode(gin.TestMode)
	router := gin.New()

	SetupIndex(router, testLogger, searchDB, kvDB, validator)

	cleanup := func() {
		var err error
		err = searchDB.Close()
		assert.NoError(err, "could not close search database")
		err = kvDB.Close()
		assert.NoError(err, "could not close kv database")
		err = os.RemoveAll(tempDir)
		assert.NoError(err, "could not remove temporary directory")
	}

	return router, cleanup
}

func makeTestHTTPRequest(router *gin.Engine, assert *require.Assertions, method string, endpoint string, headers map[string]string, requestBodyMap map[string]interface{}, queryParams map[string]string) *httptest.ResponseRecorder {

	var err error
	w := httptest.NewRecorder()

	if len(queryParams) > 0 {
		endpoint = endpoint + "?"
		for key, value := range queryParams {
			if endpoint[len(endpoint)-1] != '?' {
				endpoint = endpoint + "&"
			}
			endpoint = endpoint + key + "=" + value
		}
	}
	var jsonBody []byte
	if requestBodyMap != nil {
		jsonBody, err = json.Marshal(requestBodyMap)
		assert.NoError(err)
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBody))
	assert.NoError(err)

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	router.ServeHTTP(w, req)

	return w
}

func mustGetAbsolutePath(relativePath string) string {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		panic(err)
	}
	return absPath
}
