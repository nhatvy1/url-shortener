package handlers

import (
	"shortlink/internal/services"
	"shortlink/internal/utils"
	"shortlink/internal/validations"
	"shortlink/internal/vo"

	"github.com/gin-gonic/gin"
)

type ShortLinkHandler struct {
	shortlinkService services.ShortLinkService
}

func NewShortLinkHandler(sl services.ShortLinkService) *ShortLinkHandler {
	return &ShortLinkHandler{
		shortlinkService: sl,
	}
}

func (sl *ShortLinkHandler) CreateShortLink(c *gin.Context) {
	var createReq vo.CreateShortLinkReq
	if err := c.ShouldBindJSON(&createReq); err != nil {
		validations.HandleValidationError(c, err)
		return
	}

	result, err := sl.shortlinkService.CreateShortLink(&createReq)
	if utils.HandleError(c, err) {
		return
	}

	utils.SuccessResponse(c, 200, result)
}

func (sl *ShortLinkHandler) GetOriginalURL(c *gin.Context) {
	shortCode := c.Param("code")

	originalURL, err := sl.shortlinkService.GetOriginalUrl(shortCode)
	if utils.HandleError(c, err) {
		return
	}

	c.Redirect(302, originalURL)
}
