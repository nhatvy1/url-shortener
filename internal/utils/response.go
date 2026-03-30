package utils

import "github.com/gin-gonic/gin"

type ErrCode string

type AppErr struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Detail     any    `json:"detail,omitempty"`
}

type ApiResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Status  int    `json:"status"`
}

func SuccessResponse(ctx *gin.Context, status int, data any, message ...string) {
	msg := "success"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	ctx.JSON(status, ApiResponse{
		Status:  status,
		Message: msg,
		Data:    data,
	})
}

func ErrorResponse(ctx *gin.Context, status int, err string, detail any) {
	ctx.JSON(status, AppErr{
		StatusCode: status,
		Message:    err,
		Detail:     detail,
	})
}

func (e *AppErr) Error() string {
	return e.Message
}

func NewAppError(status int, msg string) *AppErr {
	return &AppErr{
		StatusCode: status,
		Message:    msg,
	}
}

func HandleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	if appErr, ok := err.(*AppErr); ok {
		ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr.Detail)
		return true
	}

	ErrorResponse(c, 500, "internal server error", nil)

	return true
}
