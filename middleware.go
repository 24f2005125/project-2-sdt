package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func EnsureAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost || c.FullPath() != "/ingest" {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, APIResponse[any]{
				Status:  "error",
				Message: "could_not_read_request_body",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var req TaskRequest

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, APIResponse[any]{
				Status:  "error",
				Message: "invalid_request_body",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		if req.Secret != os.Getenv("SECRET") {
			c.AbortWithStatusJSON(http.StatusForbidden, APIResponse[any]{
				Status:  "error",
				Message: "unauthorized",
				Error:   "invalid_secret",
				Data:    nil,
			})
			return
		}

		c.Next()
	}
}
