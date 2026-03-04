package services

type ShortLink interface {
	CreateShortLink(originalUrl string) (string, error)
	GetOriginalUrl(shortLink string) (string, error)
	UpdateOriginalUrl(shortLink string, newOriginalUrl string) error
	DeleteShortLink(shortLink string) error
}

type ShortLinkService struct{}

func NewShortLinkService() ShortLink {
	return &ShortLinkService{}
}

func (s *ShortLinkService) CreateShortLink(originalUrl string) (string, error) {
	return "short_link", nil
}

func (s *ShortLinkService) GetOriginalUrl(shortLink string) (string, error) {
	return "original_url", nil
}

func (s *ShortLinkService) UpdateOriginalUrl(shortLink string, newOriginalUrl string) error {
	return nil
}

func (s *ShortLinkService) DeleteShortLink(shortLink string) error {
	return nil
}
