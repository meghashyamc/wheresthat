package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const tempDirSearch = "./.wheresthat_search_test"

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
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file1.txt"),
					},
				},
			},
		},
	},
	{
		name:           "SearchGoKeyword",
		queryParams:    map[string]string{"query": "main"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file2.go"),
					},
				},
			},
		},
	},
	{
		name:           "SearchMarkdownContent",
		queryParams:    map[string]string{"query": "markdown"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/subdir/file3.md"),
					},
				},
			},
		},
	},
	{
		name:           "SearchPrefixOfContent",
		queryParams:    map[string]string{"query": "This is"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file1.txt"),
					},
				},
			},
		},
	},
	{
		name:           "SearchPrefixOfFilename",
		queryParams:    map[string]string{"query": "file1"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file1.txt"),
					},
				},
			},
		},
	},
	{
		name:           "SearchExactFilename",
		queryParams:    map[string]string{"query": "file2.go"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file2.go"),
					},
				},
			},
		},
	},
	{
		name:           "SearchNestedFile",
		queryParams:    map[string]string{"query": "Hello World"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/subdir/nested/file5.py"),
					},
				},
			},
		},
	},
	{
		name:           "SearchJSONContent",
		queryParams:    map[string]string{"query": "value"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/subdir/file4.json"),
					},
				},
			},
		},
	},
	{
		name:           "SearchNoResults",
		queryParams:    map[string]string{"query": "nonexistent"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{},
				"page_details": map[string]any{
					"current_page":  float64(1),
					"page_size":     float64(20),
					"total_pages":   float64(1),
					"has_next_page": false,
					"has_prev_page": false,
					"total_results": float64(0),
				},
			},
		},
	},
	{
		name:           "SearchWithPagination",
		queryParams:    map[string]string{"query": "test", "per_page": "1", "page": "1"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"page_details": map[string]any{
					"current_page":  float64(1),
					"page_size":     float64(1),
					"has_prev_page": false,
				},
			},
		},
	},
	{
		name:           "SearchCaseInsensitive",
		queryParams:    map[string]string{"query": "HELLO"},
		expectedStatus: http.StatusOK,
		expectedResponse: map[string]any{
			"data": map[string]any{
				"results": []any{
					map[string]any{
						"file_path": mustGetAbsolutePath(tempDirSearch + "/file2.go"),
					},
				},
			},
		},
	},
}

func TestHandleSearch(t *testing.T) {
	assert := require.New(t)
	router, cleanup := setupTestServer(t, assert, tempDirSearch)
	defer cleanup()

	indexRequestBody := map[string]any{
		"path": mustGetAbsolutePath(tempDirSearch),
	}
	w := makeTestHTTPRequest(router, assert, http.MethodPost, "/index", defaultTestRequestHeaders, indexRequestBody, nil)
	assert.Equal(http.StatusNoContent, w.Code, "Index creation should succeed before running search tests")

	for _, testCase := range searchHandlerTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert := require.New(t)
			w := makeTestHTTPRequest(router, assert, http.MethodGet, "/search", testCase.requestHeaders, nil, testCase.queryParams)
			responseBytes := w.Body.Bytes()
			assert.Equal(testCase.expectedStatus, w.Code, fmt.Sprintf("response gotten was %s", string(responseBytes)))

			if testCase.expectedResponse != nil {
				var responseMap map[string]any
				err := json.Unmarshal(responseBytes, &responseMap)
				assert.NoError(err)

				// Check specific fields that we care about
				if expectedData, exists := testCase.expectedResponse["data"]; exists {
					actualData, dataExists := responseMap["data"]
					assert.True(dataExists, "Expected data field in response")

					expectedDataMap := expectedData.(map[string]any)
					actualDataMap := actualData.(map[string]any)

					// Check results if specified
					if expectedResults, hasResults := expectedDataMap["results"]; hasResults {
						actualResults, resultsExist := actualDataMap["results"]
						assert.True(resultsExist, "Expected results field in response data")

						expectedResultsSlice := expectedResults.([]any)
						actualResultsSlice := actualResults.([]any)

						if len(expectedResultsSlice) > 0 {
							assert.GreaterOrEqual(len(actualResultsSlice), len(expectedResultsSlice), "Should have at least the expected number of results")

							// Check that expected file paths are present
							for _, expectedResult := range expectedResultsSlice {
								expectedResultMap := expectedResult.(map[string]any)
								expectedFilePath := expectedResultMap["file_path"].(string)

								found := false
								for _, actualResult := range actualResultsSlice {
									actualResultMap := actualResult.(map[string]any)
									if actualResultMap["file_path"] == expectedFilePath {
										found = true
										break
									}
								}
								assert.True(found, fmt.Sprintf("Expected file path %s not found in results", expectedFilePath))
							}
						} else {
							assert.Equal(len(expectedResultsSlice), len(actualResultsSlice), "Should have exactly the expected number of results")
						}
					}

					// Check pagination details if specified
					if expectedPageDetails, hasPageDetails := expectedDataMap["page_details"]; hasPageDetails {
						actualPageDetails, pageDetailsExist := actualDataMap["page_details"]
						assert.True(pageDetailsExist, "Expected page_details field in response data")

						expectedPageDetailsMap := expectedPageDetails.(map[string]any)
						actualPageDetailsMap := actualPageDetails.(map[string]any)

						// Check specific pagination fields
						for key, expectedValue := range expectedPageDetailsMap {
							actualValue, exists := actualPageDetailsMap[key]
							assert.True(exists, fmt.Sprintf("Expected pagination field %s not found", key))
							assert.Equal(expectedValue, actualValue, fmt.Sprintf("Pagination field %s mismatch", key))
						}
					}
				}
			}
		})
	}
}
