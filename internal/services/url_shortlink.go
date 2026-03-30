package services

import (
	"context"
	"errors"
	"shortlink/internal/utils"
	"shortlink/internal/vo"
	sqlc "shortlink/sqlc/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ShortLinkService interface {
	CreateShortLink(data *vo.CreateShortLinkReq) (string, error)
	GetOriginalUrl(shortLink string) (string, error)
	UpdateOriginalUrl(shortLink string, newOriginalUrl string) error
	DeleteShortLink(shortLink string) error
}

type shortLinkService struct {
	queries *sqlc.Queries
}

func NewShortLinkService(queries *sqlc.Queries) ShortLinkService {
	return &shortLinkService{
		queries: queries,
	}
}

func (s *shortLinkService) CreateShortLink(data *vo.CreateShortLinkReq) (string, error) {
	if data == nil {
		return "", utils.NewAppError(400, "create short link request is required")
	}

	if s.queries == nil {
		return "", errors.New("short link repository is not initialized")
	}

	expiresAt, err := utils.ParseExpireTime(data.ExpireTime)
	if err != nil {
		return "", utils.NewAppError(400, "invalid expire_time")
	}

	ctx := context.Background()

	for range 5 {
		shortCode, err := utils.GenerateRandomShortCode()
		if err != nil {
			return "", err
		}

		err = s.queries.CreateShortLink(ctx, sqlc.CreateShortLinkParams{
			ShortCode:   shortCode,
			OriginalUrl: data.OriginalUrl,
			ExpiresAt:   expiresAt,
		})
		if err == nil {
			return shortCode, nil
		}

		if !isUniqueViolation(err) {
			return "", err
		}
	}

	return "", errors.New("failed to create unique short code")
}

func (s *shortLinkService) GetOriginalUrl(shortLink string) (string, error) {
	if s.queries == nil {
		return "", errors.New("short link repository is not initialized")
	}

	originalURL, err := s.queries.GetOriginalURLByCode(context.Background(), shortLink)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", utils.NewAppError(404, "short link not found or expired")
		}

		return "", err
	}

	return originalURL, nil
}

func (s *shortLinkService) UpdateOriginalUrl(shortLink string, newOriginalUrl string) error {
	return nil
}

func (s *shortLinkService) DeleteShortLink(shortLink string) error {
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
