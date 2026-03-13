package di_container

import (
	"fmt"
	"shortlink/internal/cache"
	"shortlink/internal/database"
	"shortlink/internal/handlers"
	"shortlink/internal/routes"
	sqlc "shortlink/sqlc/db"

	"github.com/gin-gonic/gin"
)

type Container struct {
	queries *sqlc.Queries
	cache   cache.Cache
	bloom   cache.BloomFilter

	userHandler      *handlers.UserHandler
	shortLinkHandler *handlers.ShortLinkHandler
}

func NewContainer() (*Container, error) {
	c := &Container{}

	// -- 1. Database ---------
	err := database.InitDB()
	if err != nil {
		return nil, fmt.Errorf("init database : %w\n", err)
	}

	// -- 2. Redis cache -------
	redis, err := cache.InitRedis()
	if err != nil {
		return nil, fmt.Errorf("init redis client : %w\n", err)
	}

	c.cache = cache.NewRedisCache(redis)
	c.bloom = cache.NewRedisBloomFilter(redis)

	return c, nil
}

func (c *Container) SetupRouter() *gin.Engine {
	r := gin.Default()

	routes.Setup(r, routes.Dependencies{
		UserHandler:      c.userHandler,
		ShortLinkHandler: c.shortLinkHandler,
	})

	return r
}
