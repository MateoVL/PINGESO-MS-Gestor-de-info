package utils

import "github.com/gin-gonic/gin"

const RequestIDKey = "request_id"

func RequestID(c *gin.Context) string {
	return c.GetString(RequestIDKey)
}
