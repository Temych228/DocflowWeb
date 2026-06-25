package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(rdb *redis.Client, rpm int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		route := c.Request.URL.Path
		key := fmt.Sprintf("ratelimit:%s:%s", clientIP, route)

		now := time.Now().Unix()
		windowStart := now - 60

		rdb.ZRemRangeByScore(context.Background(), key, "-inf", strconv.FormatInt(windowStart, 10))

		count, _ := rdb.ZCard(context.Background(), key).Result()

		if count >= int64(rpm) {
			resp := dto.NewErrorResponse(
				dto.ErrorRateLimitExceeded,
				fmt.Sprintf("Rate limit exceeded: %d requests per minute", rpm),
				nil,
			)
			c.JSON(429, resp)
			c.Abort()
			return
		}

		rdb.ZAdd(context.Background(), key, redis.Z{Score: float64(now), Member: now})
		rdb.Expire(context.Background(), key, 2*time.Minute)

		c.Set("remaining_requests", rpm-int(count)-1)
		c.Next()
	}
}
