package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func IdempotencyMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Method != "POST" && c.Request.Method != "PATCH" && c.Request.Method != "PUT" {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			resp := dto.NewErrorResponse(
				dto.ErrorValidation,
				"Idempotency-Key header is required for POST/PATCH/PUT",
				nil,
			)
			c.JSON(400, resp)
			c.Abort()
			return
		}

		cacheKey := fmt.Sprintf("idempotency:%s", idempotencyKey)

		cachedResponse, err := rdb.Get(context.Background(), cacheKey).Result()
		if err == nil {

			var response map[string]interface{}
			if err := json.Unmarshal([]byte(cachedResponse), &response); err == nil {
				c.JSON(200, response)
				c.Abort()
				return
			}
		}

		rdb.Set(context.Background(), cacheKey+":processing", "1", 30*time.Second)

		responseWriter := &ResponseWriter{
			ResponseWriter: c.Writer,
			body:           []byte{},
		}
		c.Writer = responseWriter

		c.Next()

		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			rdb.Set(
				context.Background(),
				cacheKey,
				string(responseWriter.body),
				24*time.Hour,
			)
		}

		rdb.Del(context.Background(), cacheKey+":processing")
	}
}

type ResponseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, []byte(s)...)
	return w.ResponseWriter.WriteString(s)
}
