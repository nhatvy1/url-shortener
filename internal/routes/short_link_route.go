package routes

import (
	"shortlink/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterShortLinkRoutes(r *gin.RouterGroup, h *handlers.ShortLinkHandler) {

	shortlink := r.Group("/short-link")
	{
		shortlink.POST("/", h.CreateShortLink)
		shortlink.GET("/:code", h.GetOriginalURL)
	}
}
