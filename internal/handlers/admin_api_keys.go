package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

func (h *AdminHandler) HandleAdminAPIKeys(c echo.Context) error {
	ctx := c.Request().Context()

	keys, err := h.storage.Queries.ListAPIKeys(ctx)
	if err != nil {
		slog.Error("failed to list API keys", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load API keys")
	}

	return admin.APIKeys(c, keys).Render(ctx, c.Response())
}

func (h *AdminHandler) HandleAdminAPIKeysList(c echo.Context) error {
	ctx := c.Request().Context()

	keys, err := h.storage.Queries.ListAPIKeys(ctx)
	if err != nil {
		slog.Error("failed to list API keys", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load API keys")
	}

	return admin.APIKeysList(keys).Render(ctx, c.Response())
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type CreateAPIKeyResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
	Key    string `json:"key"`
}

func (h *AdminHandler) HandleAdminAPIKeyCreate(c echo.Context) error {
	ctx := c.Request().Context()

	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name is required"})
	}

	plaintext, hash, prefix, err := auth.GenerateAPIKey(req.Name)
	if err != nil {
		slog.Error("failed to generate API key", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate API key"})
	}

	id := uuid.New().String()

	_, err = h.storage.Queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		ID:          id,
		Name:        req.Name,
		KeyHash:     hash,
		KeyPrefix:   prefix,
		Permissions: sql.NullString{String: "products:read,products:write", Valid: true},
	})
	if err != nil {
		slog.Error("failed to create API key", "error", err, "name", req.Name)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create API key"})
	}

	slog.Info("API key created", "name", req.Name, "id", id)

	return c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		ID:     id,
		Name:   req.Name,
		Prefix: prefix,
		Key:    plaintext,
	})
}

func (h *AdminHandler) HandleAdminAPIKeyDelete(c echo.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")

	err := h.storage.Queries.DeleteAPIKey(ctx, id)
	if err != nil {
		slog.Error("failed to delete API key", "error", err, "id", id)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete API key")
	}

	slog.Info("API key deleted", "id", id)

	// Return empty string for hx-swap="outerHTML" to remove the row
	return c.String(http.StatusOK, "")
}
