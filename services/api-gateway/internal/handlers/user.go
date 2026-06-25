package handlers

import (
	"github.com/Temych228/DocflowWeb/api-gateway/internal/clients"
	"github.com/Temych228/DocflowWeb/api-gateway/internal/middleware"
	"github.com/Temych228/DocflowWeb/api-gateway/pkg/dto"
	userpb "github.com/Temych228/docflow-protos-final/user/v1"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userClient *clients.UserClient
}

func NewUserHandler(userClient *clients.UserClient) *UserHandler {
	return &UserHandler{userClient: userClient}
}

func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	resp, err := h.userClient.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "User not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.User))
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	pageSize := parseQueryInt(c, "page_size", 10)

	resp, err := h.userClient.ListUsers(c.Request.Context(), int32(page), int32(pageSize))
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"users": resp.Users, "total": resp.Total}))
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.userClient.CreateUser(c.Request.Context(), &userpb.CreateUserRequest{
		Email:    req.Email,
		Name:     req.Name,
		Role:     userRole(req.Role),
		Password: req.Password,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}
	_ = resp

	c.JSON(201, dto.NewSuccessResponse(gin.H{"message": "User created successfully"}))
}

func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, "email is required", nil))
		return
	}

	resp, err := h.userClient.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(404, dto.NewErrorResponse(dto.ErrorNotFound, "User not found", nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.User))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	resp, err := h.userClient.UpdateUser(c.Request.Context(), &userpb.UpdateUserRequest{
		Id:   req.ID,
		Name: req.Name,
		Role: userRole(req.Role),
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(resp.User))
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.userClient.DeleteUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"success": resp.Success, "id": id}))
}

func (h *UserHandler) CheckUserExists(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, "email is required", nil))
		return
	}

	_, err := h.userClient.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(200, dto.NewSuccessResponse(gin.H{"exists": false}))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"exists": true}))
}

func (h *UserHandler) BanUser(c *gin.Context) {
	targetUserID := c.Param("user_id")
	adminID := middleware.ExtractUserID(c)

	var req BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.NewErrorResponse(dto.ErrorValidation, err.Error(), nil))
		return
	}

	_, err := h.userClient.Client.BanUser(c.Request.Context(), &userpb.BanUserRequest{
		AdminId: adminID,
		UserId:  targetUserID,
		Reason:  req.Reason,
	})
	if err != nil {
		c.JSON(500, dto.NewErrorResponse(dto.ErrorInternal, grpcErrMessage(err), nil))
		return
	}

	c.JSON(200, dto.NewSuccessResponse(gin.H{"banned": true, "user_id": targetUserID}))
}

type BanUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func userRole(raw string) userpb.UserRole {
	switch raw {
	case "employee":
		return userpb.UserRole_USER_ROLE_EMPLOYEE
	case "manager":
		return userpb.UserRole_USER_ROLE_MANAGER
	case "admin":
		return userpb.UserRole_USER_ROLE_ADMIN
	default:
		return userpb.UserRole_USER_ROLE_UNSPECIFIED
	}
}
