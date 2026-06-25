package dto

import "github.com/google/uuid"

type ErrorCode string

const (
	ErrorValidation          ErrorCode = "VALIDATION_ERROR"
	ErrorInvalidUUID         ErrorCode = "INVALID_UUID"
	ErrorInvalidDateFormat   ErrorCode = "INVALID_DATE_FORMAT"
	ErrorInvalidStatus       ErrorCode = "INVALID_STATUS_TRANSITION"
	ErrorUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrorTokenExpired        ErrorCode = "TOKEN_EXPIRED"
	ErrorInvalidCreds        ErrorCode = "INVALID_CREDENTIALS"
	ErrorForbidden           ErrorCode = "FORBIDDEN"
	ErrorRoleInsufficient    ErrorCode = "ROLE_INSUFFICIENT"
	ErrorUserBanned          ErrorCode = "USER_BANNED"
	ErrorNotFound            ErrorCode = "NOT_FOUND"
	ErrorAlreadyExists       ErrorCode = "ALREADY_EXISTS"
	ErrorIdempotencyConflict ErrorCode = "IDEMPOTENCY_CONFLICT"
	ErrorUnprocessable       ErrorCode = "UNPROCESSABLE"
	ErrorRateLimitExceeded   ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrorInternal            ErrorCode = "INTERNAL_ERROR"
	ErrorServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorDatabase            ErrorCode = "DATABASE_ERROR"
)

type ErrorDetails map[string]interface{}

type ErrorResponse struct {
	Error struct {
		Code    ErrorCode    `json:"code"`
		Message string       `json:"message"`
		Details ErrorDetails `json:"details,omitempty"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func NewErrorResponse(code ErrorCode, message string, details ErrorDetails) *ErrorResponse {
	resp := &ErrorResponse{
		RequestID: uuid.New().String(),
	}
	resp.Error.Code = code
	resp.Error.Message = message
	resp.Error.Details = details
	if resp.Error.Details == nil {
		resp.Error.Details = make(ErrorDetails)
	}
	return resp
}

type SuccessResponse struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

func NewSuccessResponse(data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Data:      data,
		RequestID: uuid.New().String(),
	}
}
