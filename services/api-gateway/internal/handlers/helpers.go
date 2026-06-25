package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/status"
)

func grpcErrMessage(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}

func parseQueryInt(c *gin.Context, param string, defaultVal int) int {
	val := c.DefaultQuery(param, "")
	if val == "" {
		return defaultVal
	}
	if intVal, err := strconv.Atoi(val); err == nil {
		return intVal
	}
	return defaultVal
}
