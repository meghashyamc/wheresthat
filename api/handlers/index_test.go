package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/meghashyamc/wheresthat/db/searchdb"
	"github.com/stretchr/testify/require"
)

const testFileSystemRootIndex = "./.wheresthat_index_test"

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
		requestBody:    map[string]any{"path": mustGetAbsolutePath(testFileSystemRootIndex)},
		expectedStatus: http.StatusAccepted,
	},
	{
		name:           "SuccessDuplicate",
		requestHeaders: defaultTestRequestHeaders,
		requestBody:    map[string]any{"path": mustGetAbsolutePath(testFileSystemRootIndex)},
		expectedStatus: http.StatusAccepted,
	}}

func TestHandleCreateIndex(t *testing.T) {
	assert := require.New(t)
	server, cleanup := setupTestServer(assert, "indextest", testFileSystemRootIndex)
	defer cleanup()

	for _, testCase := range createIndexHandlerTestCases {

		t.Run(testCase.name, func(t *testing.T) {
			assert := require.New(t)
			w := makeTestHTTPRequest(server, assert, http.MethodPost, "/index", testCase.requestHeaders, testCase.requestBody, testCase.queryParams)
			responseBytes := w.Body.Bytes()
			assert.Equal(testCase.expectedStatus, w.Code, fmt.Sprintf("response gotten was %s", string(responseBytes)))
			if testCase.expectedResponse != nil {
				var responseMap map[string]any
				err := json.Unmarshal(responseBytes, &responseMap)
				assert.NoError(err)
				assert.Equal(testCase.expectedResponse, responseMap)
			}

			if testCase.expectedStatus == http.StatusAccepted {
				assertSuccessfulIndexCreation(assert, server, responseBytes)
			}
		})
	}

	numOfDocuments, err := server.indexer.(*searchdb.BleveDB).GetDocCount()
	assert.Nil(err, "could not get document count")
	assert.Equal(len(testFiles), int(numOfDocuments), "document count of index should be equal to number of test files")
}

func assertSuccessfulIndexCreation(assert *require.Assertions, server *testServer, responseBytes []byte) {

	type indexResponse struct {
		Data   IndexResponse `json:"data"`
		Errors []string      `json:"errors"`
	}
	actualResponse := indexResponse{}
	err := json.Unmarshal(responseBytes, &actualResponse)
	assert.NoError(err, "could not unmarshal gotten response")
	requestID, err := uuid.Parse(actualResponse.Data.ID)
	assert.NoError(err, "got an error parsing gotten request_id into UUID")

	maxWaitForIndexCreation := 10 * time.Second

	for startTime := time.Now().UTC(); time.Since(startTime) < maxWaitForIndexCreation; time.Sleep(500 * time.Millisecond) {
		w := makeTestHTTPRequest(server, assert, http.MethodGet, fmt.Sprintf("/index/%s", requestID), nil, nil, nil)
		if w.Code == http.StatusOK {
			return
		}
	}
	assert.Fail("timed out waiting for index creation: ", requestID.String())
}
