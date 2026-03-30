-- name: CreateShortLink :exec
INSERT INTO short_links (
  short_code, 
  original_url, 
  expires_at
) VALUES (
  $1, $2, $3
);

-- name: GetOriginalURLByCode :one
SELECT original_url
FROM short_links
WHERE short_code = $1
  AND is_active = TRUE
  AND (expires_at IS NULL OR expires_at > NOW())
LIMIT 1;