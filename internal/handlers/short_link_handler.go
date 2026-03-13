package handlers

import "github.com/gin-gonic/gin"

type ShortLinkHandler struct{}

func NewShortLinkHandler() *ShortLinkHandler {
	return &ShortLinkHandler{}
}

func (h *ShortLinkHandler) CreateShortLink(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "create short link success",
	})
}
