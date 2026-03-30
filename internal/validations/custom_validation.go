package validations

import (
	"path/filepath"
	"regexp"
	"shortlink/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	blockedDomains = map[string]bool{
		"blacklist.com": true,
		"edu.vn":        true,
		"abc.com":       true,
	}

	slugRegex   = regexp.MustCompile(`^[a-z0-9]+(?:[-.][a-z0-9]+)*$`)
	searchRegex = regexp.MustCompile(`^[a-zA-Z0-9\s]+$`)

	lowerRegex   = regexp.MustCompile(`[a-z]`)
	upperRegex   = regexp.MustCompile(`[A-Z]`)
	digitRegex   = regexp.MustCompile(`[0-9]`)
	specialRegex = regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-=\[\]\{\};:'",.<>?/\\|]`)

	datetimeFormats = []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}
)

func RegisterCustomValidation(v *validator.Validate) {
	v.RegisterValidation("email_advanced", validateEmailAdvanced)
	v.RegisterValidation("password_strong", validatePasswordStrong)
	v.RegisterValidation("slug", validateSlug)
	v.RegisterValidation("search", validateSearch)
	v.RegisterValidation("min_int", validateMinInt)
	v.RegisterValidation("max_int", validateMaxInt)
	v.RegisterValidation("file_ext", validateFileExt)
	v.RegisterValidation("iso_datetime", validateIsoDatetime)
	v.RegisterValidation("future_datetime", validateFutureDatetime)
}

func validateEmailAdvanced(fl validator.FieldLevel) bool {
	email := fl.Field().String()

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	domain := utils.NormalizeString(parts[1])
	return !blockedDomains[domain]
}

func validatePasswordStrong(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 6 {
		return false
	}

	return lowerRegex.MatchString(password) &&
		upperRegex.MatchString(password) &&
		digitRegex.MatchString(password) &&
		specialRegex.MatchString(password)
}

func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

func validateSearch(fl validator.FieldLevel) bool {
	return searchRegex.MatchString(fl.Field().String())
}

func validateMinInt(fl validator.FieldLevel) bool {
	minVal, err := strconv.ParseInt(fl.Param(), 10, 64)
	if err != nil {
		return false
	}
	return fl.Field().Int() >= minVal
}

func validateMaxInt(fl validator.FieldLevel) bool {
	maxVal, err := strconv.ParseInt(fl.Param(), 10, 64)
	if err != nil {
		return false
	}
	return fl.Field().Int() <= maxVal
}

func validateFileExt(fl validator.FieldLevel) bool {
	allowedStr := fl.Param()
	if allowedStr == "" {
		return false
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fl.Field().String())), ".")
	for _, allowed := range strings.Fields(allowedStr) {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

func validateIsoDatetime(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" {
		return true
	}

	for _, format := range datetimeFormats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

func validateFutureDatetime(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	if str == "" {
		return true
	}

	for _, format := range datetimeFormats {
		if t, err := time.Parse(format, str); err == nil {
			return t.After(time.Now().UTC())
		}
	}
	return false
}
