package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage"
)

type ctxKeyAPIKey struct{}

type APIKeyInfo struct {
	ID          string
	Name        string
	Permissions string
}

// APIKeyAuth creates middleware that authenticates requests using API keys.
// Supports both X-API-Key header and Bearer token authentication.
func APIKeyAuth(store *storage.Storage) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var key string

			if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
				key = apiKey
			} else if auth := c.Request().Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				key = strings.TrimPrefix(auth, "Bearer ")
			}

			if key == "" {
				return echo.NewHTTPError(401, "Missing API key")
			}

			if !strings.HasPrefix(key, "l3d_") {
				return echo.NewHTTPError(401, "Invalid API key format")
			}

			h := sha256.Sum256([]byte(key))
			hash := hex.EncodeToString(h[:])

			apiKey, err := store.Queries.GetAPIKeyByHash(c.Request().Context(), hash)
			if err != nil {
				slog.Debug("API key lookup failed", "error", err)
				return echo.NewHTTPError(401, "Invalid or inactive API key")
			}

			if !apiKey.IsActive.Valid || apiKey.IsActive.Int64 != 1 {
				return echo.NewHTTPError(401, "API key is inactive")
			}

			go func() {
				_ = store.Queries.UpdateAPIKeyLastUsed(context.Background(), apiKey.ID)
			}()

			permissions := ""
			if apiKey.Permissions.Valid {
				permissions = apiKey.Permissions.String
			}

			info := &APIKeyInfo{
				ID:          apiKey.ID,
				Name:        apiKey.Name,
				Permissions: permissions,
			}

			ctx := context.WithValue(c.Request().Context(), ctxKeyAPIKey{}, info)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// GetAPIKeyInfo retrieves API key info from the request context.
func GetAPIKeyInfo(ctx context.Context) *APIKeyInfo {
	if k, ok := ctx.Value(ctxKeyAPIKey{}).(*APIKeyInfo); ok {
		return k
	}
	return nil
}

// HasPermission checks if the API key has the specified permission.
func (a *APIKeyInfo) HasPermission(permission string) bool {
	if a == nil {
		return false
	}
	permissions := strings.Split(a.Permissions, ",")
	for _, p := range permissions {
		if strings.TrimSpace(p) == permission {
			return true
		}
	}
	return false
}

// GenerateAPIKey creates a new API key with the given name.
// Returns the plaintext key (show once), hash (store), and prefix (display).
func GenerateAPIKey(name string) (plaintext, hash, prefix string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}
	random := hex.EncodeToString(bytes)
	plaintext = "l3d_" + random
	prefix = "l3d_" + random[:8] + "..."

	h := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(h[:])
	return plaintext, hash, prefix, nil
}
