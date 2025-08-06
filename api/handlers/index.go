package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/meghashyamc/wheresthat/logger"
	"github.com/meghashyamc/wheresthat/services/index"
	"github.com/meghashyamc/wheresthat/validation"
)

type IndexRequest struct {
	Path string `json:"path" validate:"required,valid_path"`
}

type IndexStatusRequest struct {
	ID string `uri:"request_id" validate:"required,uuid"`
}

type IndexResponse struct {
	ID string `json:"request_id"`
}

type IndexStatusResponse struct {
	Status int    `json:"status"`
	ID     string `json:"request_id"`
}

func SetupIndex(ctx context.Context, router *gin.Engine, logger logger.Logger, indexer index.Indexer, metadataStore index.MetadataStore, validator *validation.Validator) {
	service := index.New(ctx, logger, indexer, metadataStore)
	router.POST("/index", handleCreateIndex(service, logger, validator))
	router.GET("/index/:request_id", handleGetIndexStatus(service, logger, validator))
}

func handleCreateIndex(indexService *index.Service, logger logger.Logger, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := IndexRequest{}
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Warn("could not extract expected query params from the input", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusUnprocessableEntity, []string{"failed to extract request body parameters"})
			return
		}

		if err := validator.Validate(request); err != nil {
			logger.Warn("could not validate request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusNotAcceptable, []string{err.Error()})
			return
		}

		requestID := uuid.New().String()

		if err := indexService.Build(request.Path, requestID); err != nil {
			logger.Error("failed to create index", "err", err.Error())
			writeResponse(c, nil, http.StatusConflict, []string{"failed to start indexing, possibly because another indexing operation is in progress"})
			return
		}

		response := IndexResponse{
			ID: requestID,
		}
		writeResponse(c, response, http.StatusAccepted, nil)
	}
}

func handleGetIndexStatus(indexService *index.Service, logger logger.Logger, validator *validation.Validator) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := IndexStatusRequest{}
		if err := c.ShouldBindUri(&request); err != nil {
			logger.Warn("could not extract expected params from 'get index status' request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusUnprocessableEntity, []string{"failed to extract URL parameters"})
			return
		}
		if err := validator.Validate(request); err != nil {
			logger.Warn("could not validate request", "err", err.Error())
			c.Abort()
			writeResponse(c, nil, http.StatusNotAcceptable, []string{err.Error()})
			return
		}

		status, err := indexService.GetStatus(request.ID)
		if err != nil {
			logger.Error("failed to get index status", "request_id", request.ID, "err", err.Error())
			writeResponse(c, nil, http.StatusNotFound, []string{"request not found"})
			return
		}

		response := IndexStatusResponse{
			Status: status,
			ID:     request.ID,
		}

		writeResponse(c, response, getResponseStatusFromServiceStatus(status), nil)
	}
}

func getResponseStatusFromServiceStatus(status int) int {
	responseStatus := http.StatusAccepted
	if status == index.ProgressStatusComplete {
		responseStatus = http.StatusOK
	}

	if status == index.ProgressStatusFailed {
		responseStatus = http.StatusInternalServerError
	}

	return responseStatus
}
