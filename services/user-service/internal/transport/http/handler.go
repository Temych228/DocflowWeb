package httptransport

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Temych228/DocflowWeb/services/user-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/user-service/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.UserService
}

func New(svc *service.UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.GET("/health", h.health)

		users := api.Group("/users")
		{
			users.POST("", h.createUser)
			users.GET("", h.listUsers)
			users.GET("/stats", h.userStats)
			users.GET("/batch", h.getUsersBatch)
			users.GET("/by-email", h.getUserByEmail)
			users.GET("/exists", h.checkUserExists)
			users.GET("/:id", h.getUser)
			users.PUT("/:id", h.updateUser)
			users.DELETE("/:id", h.deleteUser)
			users.PATCH("/:id/verify", h.verifyUser)
			users.PATCH("/:id/ban", h.banUser)
		}
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type createUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type banUserRequest struct {
	AdminID string `json:"admin_id"`
	Reason  string `json:"reason"`
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	user, err := h.svc.CreateUser(c.Request.Context(), domain.CreateInput{
		Email: req.Email,
		Name:  req.Name,
		Role:  domain.ParseRole(req.Role),
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) getUser(c *gin.Context) {
	user, err := h.svc.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) getUserByEmail(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "email is required")
		return
	}

	user, err := h.svc.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) updateUser(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	input := domain.UpdateInput{Name: req.Name}
	if role, ok := parseOptionalRole(req.Role); ok {
		input.Role = role
	}

	user, err := h.svc.UpdateUser(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) deleteUser(c *gin.Context) {
	if err := h.svc.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	role := strings.TrimSpace(c.Query("role"))

	users, total, err := h.svc.ListUsers(c.Request.Context(), page, pageSize, role)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"role":      role,
	})
}

func (h *Handler) getUsersBatch(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("ids"))
	if raw == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "ids is required")
		return
	}

	ids := strings.Split(raw, ",")
	users, err := h.svc.GetUsersBatch(c.Request.Context(), ids)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) checkUserExists(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "email is required")
		return
	}

	exists, userID, err := h.svc.CheckUserExists(c.Request.Context(), email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":  exists,
		"email":   email,
		"user_id": userID,
	})
}

func (h *Handler) verifyUser(c *gin.Context) {
	if err := h.svc.VerifyUserEmail(c.Request.Context(), c.Param("id")); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) banUser(c *gin.Context) {
	var req banUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.svc.BanUser(c.Request.Context(), c.Param("id"), req.AdminID, req.Reason); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) userStats(c *gin.Context) {
	stats, err := h.svc.GetUserStats(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func writeError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func handleServiceError(c *gin.Context, err error) {
	switch err {
	case domain.ErrUserNotFound:
		writeError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case domain.ErrEmailTaken:
		writeError(c, http.StatusConflict, "ALREADY_EXISTS", err.Error())
	case domain.ErrInvalidInput:
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case domain.ErrUserBanned:
		writeError(c, http.StatusForbidden, "USER_BANNED", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func parseOptionalRole(raw string) (domain.Role, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.RoleEmployee):
		return domain.RoleEmployee, true
	case string(domain.RoleManager):
		return domain.RoleManager, true
	case string(domain.RoleAdmin):
		return domain.RoleAdmin, true
	default:
		return "", false
	}
}
