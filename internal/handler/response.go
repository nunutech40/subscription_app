package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ─── Standard Response Format ───────────────────────────────────────
//
// Semua endpoint /api/* harus pakai format ini agar konsisten.
//
// SUCCESS:
//   { "data": { ... }, "message": "optional" }
//
// ERROR:
//   { "error": { "code": "ERROR_CODE", "message": "Human readable" } }
//
// LIST:
//   { "data": [...], "meta": { "page": 1, "per_page": 20, "total": 100 } }
//
// ─────────────────────────────────────────────────────────────────────

// ErrorCode defines machine-readable error codes for the frontend.
type ErrorCode string

const (
	ErrCodeValidation         ErrorCode = "VALIDATION_ERROR"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeConflict           ErrorCode = "CONFLICT"
	ErrCodeQuotaFull          ErrorCode = "QUOTA_FULL"
	ErrCodeRateLimit          ErrorCode = "RATE_LIMITED"
	ErrCodeInternal           ErrorCode = "INTERNAL_ERROR"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       ErrorCode = "TOKEN_INVALID"
	ErrCodeAccountInactive    ErrorCode = "ACCOUNT_INACTIVE"
)

// errorResponse is the standard error envelope.
type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// successResponse is the standard success envelope.
type successResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// PaginationMeta holds pagination info for list endpoints.
type PaginationMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

// listResponse is the standard list envelope with pagination.
type listResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// ─── Response Helpers ───────────────────────────────────────────────

// RespondSuccess sends a standard success response.
//
//	RespondSuccess(c, http.StatusOK, data)
//	RespondSuccess(c, http.StatusCreated, data, "Berhasil dibuat")
func RespondSuccess(c *gin.Context, status int, data interface{}, message ...string) {
	resp := successResponse{Data: data}
	if len(message) > 0 {
		resp.Message = message[0]
	}
	c.JSON(status, resp)
}

// RespondList sends a standard list response with pagination.
func RespondList(c *gin.Context, data interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, listResponse{
		Data: data,
		Meta: meta,
	})
}

// RespondError sends a standard error response.
//
//	RespondError(c, http.StatusBadRequest, ErrCodeValidation, "Email tidak valid")
func RespondError(c *gin.Context, status int, code ErrorCode, message string) {
	c.JSON(status, errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// ─── Shortcut Helpers ───────────────────────────────────────────────

// RespondBadRequest sends a 400 validation error.
func RespondBadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, ErrCodeValidation, message)
}

// RespondUnauthorized sends a 401 error.
func RespondUnauthorized(c *gin.Context, message string) {
	RespondError(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// RespondForbidden sends a 403 error.
func RespondForbidden(c *gin.Context, message string) {
	RespondError(c, http.StatusForbidden, ErrCodeForbidden, message)
}

// RespondNotFound sends a 404 error.
func RespondNotFound(c *gin.Context, message string) {
	RespondError(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// RespondConflict sends a 409 error.
func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, ErrCodeConflict, message)
}

// RespondQuotaFull sends a 503 quota error.
func RespondQuotaFull(c *gin.Context, message string) {
	RespondError(c, http.StatusServiceUnavailable, ErrCodeQuotaFull, message)
}

// RespondInternalError sends a 500 error (generic, don't leak details).
func RespondInternalError(c *gin.Context) {
	RespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Terjadi kesalahan server")
}
