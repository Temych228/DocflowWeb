package middleware

import (
	"context"
	"strings"

	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	"github.com/gin-gonic/gin"
)

func JWTMiddleware(authClient *clients.AuthClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			resp := dto.NewErrorResponse(
				dto.ErrorUnauthorized,
				"Missing Authorization header",
				nil,
			)
			c.JSON(401, resp)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			resp := dto.NewErrorResponse(
				dto.ErrorUnauthorized,
				"Invalid Authorization header format",
				nil,
			)
			c.JSON(401, resp)
			c.Abort()
			return
		}

		token := parts[1]

		validateResp, err := authClient.ValidateToken(context.Background(), token)
		if err != nil {
			resp := dto.NewErrorResponse(
				dto.ErrorTokenExpired,
				"Invalid or expired token",
				nil,
			)
			c.JSON(401, resp)
			c.Abort()
			return
		}

		if !validateResp.Valid {
			resp := dto.NewErrorResponse(
				dto.ErrorUnauthorized,
				"Token is not valid",
				nil,
			)
			c.JSON(401, resp)
			c.Abort()
			return
		}

		c.Set("user_id", validateResp.UserId)
		c.Set("email", validateResp.Email)
		c.Set("role", validateResp.Role)

		c.Next()
	}
}

func ExtractUserID(c *gin.Context) string {
	val, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	if userID, ok := val.(string); ok {
		return userID
	}
	return ""
}

func ExtractRole(c *gin.Context) string {
	val, exists := c.Get("role")
	if !exists {
		return ""
	}
	if role, ok := val.(string); ok {
		return role
	}
	return ""
}

func RBACMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := ExtractRole(c)
		if userRole == "" {
			resp := dto.NewErrorResponse(
				dto.ErrorUnauthorized,
				"No role in token",
				nil,
			)
			c.JSON(401, resp)
			c.Abort()
			return
		}

		allowed := false
		for _, role := range requiredRoles {
			if userRole == role || userRole == "admin" {
				allowed = true
				break
			}
		}

		if !allowed {
			resp := dto.NewErrorResponse(
				dto.ErrorRoleInsufficient,
				"Insufficient permissions for this operation",
				nil,
			)
			c.JSON(403, resp)
			c.Abort()
			return
		}

		c.Next()
	}
}
