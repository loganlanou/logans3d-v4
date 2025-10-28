package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

type AdminEmailsHandler struct {
	queries *db.Queries
}

func NewAdminEmailsHandler(queries *db.Queries) *AdminEmailsHandler {
	return &AdminEmailsHandler{
		queries: queries,
	}
}

// HandleEmailHistory shows all emails sent by the system
func (h *AdminEmailsHandler) HandleEmailHistory(c echo.Context) error {
	ctx := c.Request().Context()

	// Get pagination parameters
	page := 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := int64(50)
	offset := int64((page - 1) * 50)

	// Get emails
	emails, err := h.queries.GetAllEmailHistory(ctx, db.GetAllEmailHistoryParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load email history")
	}

	// Get total count
	totalCount, err := h.queries.CountEmailHistory(ctx)
	if err != nil {
		totalCount = 0
	}

	return admin.EmailHistory(c, emails, page, int(totalCount)).Render(c.Request().Context(), c.Response().Writer)
}
