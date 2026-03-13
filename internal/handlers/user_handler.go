package handlers

import (
	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) Register(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "hello",
	})
}
