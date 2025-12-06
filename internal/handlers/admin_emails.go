package handlers

import (
	"log/slog"
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

	// Get filter parameter
	emailFilter := c.QueryParam("email")

	// Get pagination parameters
	page := 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := int64(50)
	offset := int64((page - 1) * 50)

	var emails []db.EmailHistory
	var totalCount int64
	var err error

	if emailFilter != "" {
		// Filter by email address
		emails, err = h.queries.GetEmailHistoryByEmail(ctx, db.GetEmailHistoryByEmailParams{
			RecipientEmail: emailFilter,
			Limit:          limit,
			Offset:         offset,
		})
		if err != nil {
			slog.Error("failed to get email history by email", "error", err, "email", emailFilter)
			return c.String(http.StatusInternalServerError, "Failed to load email history")
		}

		// Count filtered emails (we need to get all to count, or add a count query)
		// For now, estimate based on returned results
		allEmails, err := h.queries.GetEmailHistoryByEmail(ctx, db.GetEmailHistoryByEmailParams{
			RecipientEmail: emailFilter,
			Limit:          10000, // Large limit to get count
			Offset:         0,
		})
		if err != nil {
			totalCount = int64(len(emails))
		} else {
			totalCount = int64(len(allEmails))
		}
	} else {
		// Get all emails
		emails, err = h.queries.GetAllEmailHistory(ctx, db.GetAllEmailHistoryParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			slog.Error("failed to get email history", "error", err)
			return c.String(http.StatusInternalServerError, "Failed to load email history")
		}

		// Get total count
		totalCount, err = h.queries.CountEmailHistory(ctx)
		if err != nil {
			slog.Error("failed to count email history", "error", err)
			totalCount = 0
		}
	}

	return admin.EmailHistory(c, emails, page, int(totalCount), emailFilter).Render(c.Request().Context(), c.Response().Writer)
}
