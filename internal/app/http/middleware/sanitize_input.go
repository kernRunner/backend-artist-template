package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

// SanitizeAndCleanInputMiddleware cleans all string fields in JSON input using bluemonday
func SanitizeAndCleanInputMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only for JSON requests
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		// Read and decode JSON
		var body map[string]interface{}
		buf, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid body"})
			return
		}
		if err := json.Unmarshal(buf, &body); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Malformed JSON"})
			return
		}

		// Sanitize strings using bluemonday
		policy := bluemonday.StrictPolicy()
		for k, v := range body {
			if str, ok := v.(string); ok {
				// Clean string input
				body[k] = policy.Sanitize(str)
			}
		}

		// Marshal sanitized body back
		newBody, _ := json.Marshal(body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(newBody))
		c.Request.ContentLength = int64(len(newBody))

		c.Next()
	}
}
