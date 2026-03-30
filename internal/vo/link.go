package vo

type CreateShortLinkReq struct {
	OriginalUrl string  `json:"original_url" binding:"required,url"`
	ExpireTime  *string `json:"expire_time"  binding:"omitempty,iso_datetime"`
	IsActive    *bool   `json:"is_active"    binding:"omitempty"`
}
