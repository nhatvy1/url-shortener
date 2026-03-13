package routes

import (
	"shortlink/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterShortLinkRoutes(r *gin.RouterGroup, h *handlers.ShortLinkHandler) {

	auth := r.Group("/short-link")
	{
		auth.POST("/create", h.CreateShortLink)
	}
}
