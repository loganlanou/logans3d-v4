package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

const returnToCookieName = "logans3d_return_to"

var disallowedReturnTo = map[string]struct{}{
	"/login":         {},
	"/sign-up":       {},
	"/logout":        {},
	"/auth/callback": {},
}

func sanitizeReturnTo(path string) (string, bool) {
	if path == "" {
		return "", false
	}

	if strings.ContainsAny(path, "\r\n") {
		return "", false
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "//") {
		return "", false
	}

	if !strings.HasPrefix(path, "/") {
		return "", false
	}

	base := path
	if idx := strings.IndexAny(path, "?#"); idx != -1 {
		base = path[:idx]
	}

	if _, blocked := disallowedReturnTo[base]; blocked {
		return "", false
	}

	if strings.HasPrefix(base, "/auth/") {
		return "", false
	}

	return path, true
}

func rememberReturnTo(c echo.Context, path string) {
	if sanitized, ok := sanitizeReturnTo(path); ok {
		c.SetCookie(&http.Cookie{
			Name:     returnToCookieName,
			Value:    url.QueryEscape(sanitized),
			Path:     "/",
			HttpOnly: true,
			Secure:   os.Getenv("ENVIRONMENT") == "production",
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300, // 5 minutes
		})
	}
}

func clearReturnTo(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     returnToCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("ENVIRONMENT") == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func popReturnTo(c echo.Context) string {
	cookie, err := c.Cookie(returnToCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	clearReturnTo(c)

	decoded, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return ""
	}

	sanitized, ok := sanitizeReturnTo(decoded)
	if !ok {
		return ""
	}

	return sanitized
}
