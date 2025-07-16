package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/stretchr/testify/require"
)

const testFileSystemRootSearch = "./.wheresthat_search_test"

var searchHandlerTestCases = []testCase{
	{
		name:           "NoQuery",
		queryParams:    map[string]string{},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "EmptyQuery",
		queryParams:    map[string]string{"query": ""},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "QueryTooLong",
		queryParams:    map[string]string{"query": strings.Repeat("a", 1001)},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "InvalidPerPage",
		queryParams:    map[string]string{"query": "test", "per_page": "-1"},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "InvalidPage",
		queryParams:    map[string]string{"query": "test", "page": "-1"},
		expectedStatus: http.StatusNotAcceptable,
	},
	{
		name:           "SearchContentWithinFile",
		queryParams:    map[string]string{"query": "test content"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file1.txt"),
					},
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/file3.md"),
					},
				},
			},
		}},
	{
		name:           "SearchGoKeyword",
		queryParams:    map[string]string{"query": "main"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file2.go"),
					},
				},
			},
		},
	},
	{
		name:           "SearchMarkdownContent",
		queryParams:    map[string]string{"query": "markdown"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/file3.md"),
					},
				},
			},
		},
	},
	{
		name:           "SearchPrefixOfContent",
		queryParams:    map[string]string{"query": "conten"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file1.txt"),
					},
				},
			},
		},
	},
	{
		name:           "SearchPrefixOfFilename",
		queryParams:    map[string]string{"query": "file1"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file1.txt"),
					},
				},
			},
		},
	},
	{
		name:           "SearchExactFilename",
		queryParams:    map[string]string{"query": "file2.go"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file2.go"),
					},
				},
			},
		},
	},
	{
		name:           "SearchNestedFile",
		queryParams:    map[string]string{"query": "Hello World"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/nested/file5.py"),
					},
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file2.go"),
					},
				},
			},
		},
	},
	{
		name:           "SearchJSONContent",
		queryParams:    map[string]string{"query": "value"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/file4.json"),
					},
				},
			},
		},
	},
	{
		name:           "SearchNoResults",
		queryParams:    map[string]string{"query": "nonexistent"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{},
				PageDetails: Pagination{
					CurrentPage:  1,
					PageSize:     20,
					TotalPages:   1,
					HasNextPage:  false,
					HasPrevPage:  false,
					TotalResults: 0,
				},
			},
		},
	},
	{
		name:           "SearchWithPagination",
		queryParams:    map[string]string{"query": "test", "per_page": "1", "page": "1"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/file3.md"),
					},
				},
				PageDetails: Pagination{
					CurrentPage:  1,
					PageSize:     1,
					HasPrevPage:  false,
					HasNextPage:  true,
					TotalPages:   2,
					TotalResults: 2,
				},
			},
		},
	},
	{
		name:           "SearchCaseInsensitive",
		queryParams:    map[string]string{"query": "HELLO"},
		expectedStatus: http.StatusOK,
		expectedResponse: &response{
			Data: SearchResponse{
				Results: []searchdb.Result{
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/nested/file5.py"),
					},
					{
						Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file2.go"),
					},
				},
			},
		},
	},
}

var searchBeforeFileChanges = testCase{

	name:           "SearchBeforeFileChanges",
	queryParams:    map[string]string{"query": "HELLO"},
	expectedStatus: http.StatusOK,
	expectedResponse: &response{
		Data: SearchResponse{
			Results: []searchdb.Result{
				{
					Path: mustGetAbsolutePath(testFileSystemRootSearch + "/subdir/nested/file5.py"),
				},
				{
					Path: mustGetAbsolutePath(testFileSystemRootSearch + "/file2.go"),
				},
			},
		},
	},
}

var searchAfterFileChanges = testCase{

	name:           "SearchAfterFileChanges",
	queryParams:    map[string]string{"query": "HELLO"},
	expectedStatus: http.StatusOK,
	expectedResponse: &response{
		Data: SearchResponse{
			Results: []searchdb.Result{
				{
					Path: mustGetAbsolutePath(testFileSystemRootSearch + "/newfile.txt"),
				},
			},
		},
	},
}

func TestHandleSearch(t *testing.T) {

	assert := require.New(t)
	server, cleanup := setupTestServer(assert, "searchtest", testFileSystemRootSearch)
	defer cleanup()

	indexRequestBody := map[string]any{
		"path": mustGetAbsolutePath(testFileSystemRootSearch),
	}
	w := makeTestHTTPRequest(server, assert, http.MethodPost, "/index", defaultTestRequestHeaders, indexRequestBody, nil)
	assert.Equal(http.StatusNoContent, w.Code, "index creation should succeed before running search tests")

	for _, testCase := range searchHandlerTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertSearchResults(t, testCase, server)
		})
	}

	// Testing scenario where the index is created multiple times with and without file changes
	// Call /index with the same request body again
	w = makeTestHTTPRequest(server, assert, http.MethodPost, "/index", defaultTestRequestHeaders, indexRequestBody, nil)
	assert.Equal(http.StatusNoContent, w.Code, "duplicate index creation should succeed")
	t.Run(searchBeforeFileChanges.name, func(t *testing.T) {
		assertSearchResults(t, searchBeforeFileChanges, server)
	})

	makeFileChanges(assert, testFileSystemRootSearch)

	// Call /index with the same request body again after file changes
	w = makeTestHTTPRequest(server, assert, http.MethodPost, "/index", defaultTestRequestHeaders, indexRequestBody, nil)
	assert.Equal(http.StatusNoContent, w.Code, "duplicate index creation should succeed")

	// Try searching after file changes
	t.Run(searchAfterFileChanges.name, func(t *testing.T) {
		assertSearchResults(t, searchAfterFileChanges, server)
	})

}

func makeFileChanges(assert *require.Assertions, testFileSystemRootSearch string) {

	// 1. Edit a file
	editedFilePath := filepath.Join(testFileSystemRootSearch, "subdir/nested/file5.py")
	err := os.WriteFile(editedFilePath, []byte("print('ping')"), 0644)
	assert.NoError(err, "should be able to edit existing file")

	// 2. Delete a file
	deletedFilePath := filepath.Join(testFileSystemRootSearch, "file2.go")
	err = os.Remove(deletedFilePath)
	assert.NoError(err, "should be able to delete file")

	// 3. Add a new file
	newFilePath := filepath.Join(testFileSystemRootSearch, "newfile.txt")
	err = os.WriteFile(newFilePath, []byte("Hello, this is a new file"), 0644)
	assert.NoError(err, "should be able to create new file")

}
func assertSearchResults(t *testing.T, testCase testCase, server *testServer) {
	type searchResponse struct {
		Data   SearchResponse `json:"data"`
		Errors []string       `json:"errors"`
	}
	assert := require.New(t)
	w := makeTestHTTPRequest(server, assert, http.MethodGet, "/search", testCase.requestHeaders, nil, testCase.queryParams)
	responseBytes := w.Body.Bytes()
	assert.Equal(testCase.expectedStatus, w.Code, fmt.Sprintf("response gotten was %s", string(responseBytes)))

	if testCase.expectedResponse == nil {
		return
	}

	actualResponse := searchResponse{}
	err := json.Unmarshal(responseBytes, &actualResponse)
	assert.NoError(err, "could not unmarshal gotten response")

	expectedResponseData := testCase.expectedResponse.Data.(SearchResponse)

	assert.Equal(len(expectedResponseData.Results), len(actualResponse.Data.Results), "should have the expected number of results")

	for i, expectedResult := range expectedResponseData.Results {
		assert.Equal(expectedResult.Path, actualResponse.Data.Results[i].Path)
	}

	if expectedResponseData.PageDetails == (Pagination{}) {
		return
	}

	assert.Equal(expectedResponseData.PageDetails, actualResponse.Data.PageDetails)

}
