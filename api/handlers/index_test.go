package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleCreateIndex(t *testing.T) {
	assert := require.New(t)
	router, cleanup := setupTestServer(t, assert)
	defer cleanup()

	for _, testCase := range createIndexHandlerTestCases {

		t.Run(testCase.name, func(t *testing.T) {
			assert := require.New(t)
			w := makeTestHTTPRequest(router, assert, http.MethodPost, "/index", testCase.requestHeaders, testCase.requestBody, testCase.queryParams)
			responseBytes := w.Body.Bytes()
			assert.Equal(testCase.expectedStatus, w.Code, fmt.Sprintf("response gotten was %s", string(responseBytes)))
			if testCase.expectedResponse != nil {
				var responseMap map[string]interface{}
				err := json.Unmarshal(responseBytes, &responseMap)
				assert.NoError(err)
				assert.Equal(testCase.expectedResponse, responseMap)
			}
		})

	}
}
