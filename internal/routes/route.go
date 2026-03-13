package routes

import (
	"shortlink/internal/handlers"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	UserHandler      *handlers.UserHandler
	ShortLinkHandler *handlers.ShortLinkHandler
}

func Setup(r *gin.Engine, deps Dependencies) {
	v1 := r.Group("/api/v1")

	RegisterUserRoutes(v1, deps.UserHandler)
	RegisterShortLinkRoutes(v1, deps.ShortLinkHandler)
}
