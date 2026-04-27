package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK           = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeNotFound     = 404
	CodeError        = 500
)

type Body struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeOK,
		Data:    data,
		Message: "success",
	})
}

func Error(c *gin.Context, status int, code int, message string) {
	c.JSON(status, Body{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}
