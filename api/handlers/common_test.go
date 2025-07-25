// Common test helpers
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/meghashyamc/wheresthat/config"
	"github.com/meghashyamc/wheresthat/db/kvdb"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/index"
	"github.com/meghashyamc/wheresthat/validation"
	"github.com/stretchr/testify/require"
)

var defaultTestRequestHeaders = map[string]string{"Content-Type": "application/json"}

var testFiles = map[string]string{
	"file1.txt":              "This is test content for file1",
	"file2.go":               "package main\n\nfunc main() {\n\tprint(\"Hello\")\n}",
	"subdir/file3.md":        "# Test Markdown\n\nThis is a test markdown file",
	"subdir/file4.json":      `{"key": "value", "number": 42}`,
	"subdir/nested/file5.py": "def hello():\n    print('Hello World')",
}

type testServer struct {
	router  *gin.Engine
	kvDB    kvdb.DB
	indexer index.Indexer
}

type testCase struct {
	name             string
	requestHeaders   map[string]string
	requestBody      map[string]any
	queryParams      map[string]string
	expectedStatus   int
	expectedResponse *response
}

func newTestLogger() logger.Logger {

	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
func setupTestServer(assert *require.Assertions, testEnv string, testFileSystemRoot string) (*testServer, func()) {

	cfg, err := config.Load(testEnv)
	assert.NoError(err, "could not load config")

	for relPath, content := range testFiles {
		fullPath := filepath.Join(testFileSystemRoot, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		assert.NoError(err, "could not create test sub-directory")
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NoError(err, "could not write test file")
	}

	testLogger := newTestLogger()

	searchDB, err := searchdb.New(testLogger, cfg)
	assert.NoError(err, "could not create search database")

	kvDB, err := kvdb.New(testLogger, cfg)
	assert.NoError(err, "could not create kv database")
	validator, err := validation.New(testLogger)
	assert.NoError(err, "could not create validator")
	gin.SetMode(gin.TestMode)
	router := gin.New()

	ctx := context.Background()

	SetupIndex(ctx, router, testLogger, searchDB, kvDB, validator)
	SetupSearch(router, testLogger, searchDB, validator)

	cleanup := func() {
		var err error
		err = searchDB.Close()
		assert.NoError(err, "could not close search database")
		err = kvDB.Close()
		assert.NoError(err, "could not close kv database")
		err = os.RemoveAll(testFileSystemRoot)
		assert.NoError(err, "could not remove temporary file system directory")
		err = os.RemoveAll(cfg.GetStoragePath())
		assert.NoError(err, "could not remove storage directory")
	}

	return &testServer{
		router:  router,
		kvDB:    kvDB,
		indexer: searchDB,
	}, cleanup
}

func makeTestHTTPRequest(server *testServer, assert *require.Assertions, method string, endpoint string, headers map[string]string, requestBodyMap map[string]interface{}, queryParams map[string]string) *httptest.ResponseRecorder {

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
	var req *http.Request
	if requestBodyMap != nil {
		jsonBody, err = json.Marshal(requestBodyMap)
		assert.NoError(err)
	}

	slog.Info("Making test request", "method", method, "endpoint", endpoint, "headers", headers, "body", string(jsonBody))

	if len(jsonBody) > 0 {
		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBody))
	} else {
		req, err = http.NewRequest(method, endpoint, nil)
	}
	assert.NoError(err)

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	server.router.ServeHTTP(w, req)

	return w
}

func mustGetAbsolutePath(relativePath string) string {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		panic(err)
	}
	return absPath
}
