package handlers

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

// HandleQuoteDraftsList handles GET /admin/quotes - shows list of all quote drafts with filtering
func (h *AdminHandler) HandleQuoteDraftsList(c echo.Context) error {
	ctx := c.Request().Context()

	// Get filter parameters - default to "completed"
	status := c.QueryParam("status")
	if status == "" {
		status = "completed"
	}
	search := c.QueryParam("search")

	// Get counts for stat cards
	countRow, err := h.storage.Queries.CountDraftsByStatus(ctx)
	if err != nil {
		slog.Error("failed to get draft counts", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to fetch draft counts")
	}

	counts := admin.DraftCounts{
		Total:      countRow.Total,
		Completed:  countRow.Completed,
		Abandoned:  countRow.Abandoned,
		InProgress: countRow.InProgress,
		WithEmail:  countRow.WithEmail,
		Archived:   countRow.Archived,
	}

	// Get filtered list
	var drafts []db.CustomQuoteDraft
	limit := int64(100)
	offset := int64(0)

	if search != "" {
		// Search across name, email, description
		searchPattern := "%" + search + "%"
		drafts, err = h.storage.Queries.SearchDrafts(ctx, db.SearchDraftsParams{
			Name:        sql.NullString{String: searchPattern, Valid: true},
			Email:       sql.NullString{String: searchPattern, Valid: true},
			Description: sql.NullString{String: searchPattern, Valid: true},
			Limit:       limit,
			Offset:      offset,
		})
	} else {
		switch status {
		case "completed":
			drafts, err = h.storage.Queries.ListCompletedDrafts(ctx, db.ListCompletedDraftsParams{
				Limit:  limit,
				Offset: offset,
			})
		case "abandoned":
			drafts, err = h.storage.Queries.ListAbandonedDrafts(ctx, db.ListAbandonedDraftsParams{
				Limit:  limit,
				Offset: offset,
			})
		case "in_progress":
			drafts, err = h.storage.Queries.ListInProgressDrafts(ctx, db.ListInProgressDraftsParams{
				Limit:  limit,
				Offset: offset,
			})
		case "archived":
			drafts, err = h.storage.Queries.ListArchivedDrafts(ctx, db.ListArchivedDraftsParams{
				Limit:  limit,
				Offset: offset,
			})
		default: // "all"
			drafts, err = h.storage.Queries.ListAllDrafts(ctx, db.ListAllDraftsParams{
				Limit:  limit,
				Offset: offset,
			})
		}
	}

	if err != nil {
		slog.Error("failed to list drafts", "error", err, "status", status, "search", search)
		return c.String(http.StatusInternalServerError, "Failed to fetch quote drafts")
	}

	filters := admin.DraftFilters{
		Status: status,
		Search: search,
	}

	return Render(c, admin.QuoteDraftsList(c, drafts, counts, filters))
}

// HandleQuoteDraftDetail handles GET /admin/quotes/:id - shows detail of a single quote draft
func (h *AdminHandler) HandleQuoteDraftDetail(c echo.Context) error {
	ctx := c.Request().Context()
	draftID := c.Param("id")

	if draftID == "" {
		return c.String(http.StatusBadRequest, "Draft ID is required")
	}

	// Get the draft
	draft, err := h.storage.Queries.GetDraftByID(ctx, draftID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Quote draft not found")
		}
		slog.Error("failed to get draft", "error", err, "draft_id", draftID)
		return c.String(http.StatusInternalServerError, "Failed to fetch quote draft")
	}

	// Get associated files from multiple sources
	var files []admin.QuoteFile

	// 1. Check if draft is linked to a quote_request and get files from there (preferred)
	if draft.QuoteRequestID.Valid && draft.QuoteRequestID.String != "" {
		quoteFiles, err := h.storage.Queries.GetQuoteFiles(ctx, draft.QuoteRequestID.String)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("failed to get quote files by quote_request_id", "error", err, "quote_request_id", draft.QuoteRequestID.String)
		}
		for _, f := range quoteFiles {
			files = append(files, admin.QuoteFile{
				ID:           f.ID,
				Filename:     f.Filename,
				OriginalName: f.OriginalFilename,
				FilePath:     f.FilePath,
				FileSize:     f.FileSize,
				FileType:     f.MimeType,
			})
		}
		slog.Debug("loaded files via quote_request_id link", "draft_id", draftID, "quote_request_id", draft.QuoteRequestID.String, "file_count", len(files))
	}

	// 2. If no files found via link, check custom_quote_draft_files table
	if len(files) == 0 {
		draftFiles, err := h.storage.Queries.GetDraftFiles(ctx, draftID)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("failed to get draft files", "error", err, "draft_id", draftID)
		}
		for _, f := range draftFiles {
			files = append(files, admin.QuoteFile{
				ID:           f.ID,
				Filename:     f.Filename,
				OriginalName: f.Filename,
				FilePath:     f.FilePath,
				FileSize:     f.FileSize,
				FileType:     f.FileType,
			})
		}
	}

	// 3. Fallback: If still no files and draft has email, check quote_files table by email
	//    This handles legacy drafts that weren't properly linked
	if len(files) == 0 && draft.Email.Valid && draft.Email.String != "" {
		quoteFiles, err := h.storage.Queries.GetQuoteFilesByEmailLatest(ctx, draft.Email.String)
		if err != nil && err != sql.ErrNoRows {
			slog.Error("failed to get quote files by email", "error", err, "email", draft.Email.String)
		}
		for _, f := range quoteFiles {
			files = append(files, admin.QuoteFile{
				ID:           f.ID,
				Filename:     f.Filename,
				OriginalName: f.OriginalFilename,
				FilePath:     f.FilePath,
				FileSize:     f.FileSize,
				FileType:     f.MimeType,
			})
		}
	}

	slog.Debug("loaded quote draft detail",
		"draft_id", draftID,
		"quote_request_id", draft.QuoteRequestID.String,
		"total_files", len(files))

	return Render(c, admin.QuoteDraftDetail(c, draft, files))
}

// HandleSendQuoteDraftRecoveryEmail handles POST /admin/quotes/:id/send-recovery
func (h *AdminHandler) HandleSendQuoteDraftRecoveryEmail(c echo.Context) error {
	ctx := c.Request().Context()
	draftID := c.Param("id")

	if draftID == "" {
		return c.String(http.StatusBadRequest, "Draft ID is required")
	}

	// Get form values
	subject := c.FormValue("subject")
	customMessage := c.FormValue("custom_message")

	if subject == "" {
		subject = "Continue Your Custom Quote - Logan's 3D Creations"
	}

	// Get the draft
	draft, err := h.storage.Queries.GetDraftByID(ctx, draftID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Quote draft not found")
		}
		slog.Error("failed to get draft for recovery email", "error", err, "draft_id", draftID)
		return c.String(http.StatusInternalServerError, "Failed to fetch quote draft")
	}

	// Validate email exists
	if !draft.Email.Valid || draft.Email.String == "" {
		return c.String(http.StatusBadRequest, "Cannot send recovery email: draft has no email address")
	}

	// Send the recovery email
	err = h.emailService.SendQuoteDraftRecoveryEmail(ctx, draft, subject, customMessage)
	if err != nil {
		slog.Error("failed to send recovery email", "error", err, "draft_id", draftID, "email", draft.Email.String)
		return c.String(http.StatusInternalServerError, "Failed to send recovery email: "+err.Error())
	}

	// Mark recovery email sent
	err = h.storage.Queries.MarkDraftRecoveryEmailSent(ctx, draftID)
	if err != nil {
		slog.Error("failed to mark recovery email sent", "error", err, "draft_id", draftID)
		// Don't fail - email was sent successfully
	}

	slog.Info("sent quote draft recovery email", "draft_id", draftID, "email", draft.Email.String)

	// Redirect back to detail page with success message
	return c.Redirect(http.StatusSeeOther, "/admin/quotes/"+draftID+"?success=email_sent")
}

// Helper function to determine draft status
func GetDraftStatus(draft db.CustomQuoteDraft) string {
	if draft.CompletedAt.Valid {
		return "completed"
	}
	if draft.Email.Valid && draft.Email.String != "" {
		// Check if it's been more than 24 hours since last update
		if time.Since(draft.UpdatedAt) > 24*time.Hour {
			return "abandoned"
		}
	}
	return "in_progress"
}

// Helper function to format time ago for display
func FormatDraftTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// HandleArchiveQuoteDraft handles POST /admin/quotes/:id/archive
func (h *AdminHandler) HandleArchiveQuoteDraft(c echo.Context) error {
	ctx := c.Request().Context()
	draftID := c.Param("id")

	if draftID == "" {
		return c.String(http.StatusBadRequest, "Draft ID is required")
	}

	// Archive the draft
	err := h.storage.Queries.ArchiveDraft(ctx, draftID)
	if err != nil {
		slog.Error("failed to archive draft", "error", err, "draft_id", draftID)
		return c.String(http.StatusInternalServerError, "Failed to archive quote draft")
	}

	slog.Info("archived quote draft", "draft_id", draftID)

	// Check if request wants JSON response (for HTMX)
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.String(http.StatusOK, "")
	}

	// Redirect back to list or detail page
	referer := c.Request().Header.Get("Referer")
	if referer != "" {
		return c.Redirect(http.StatusSeeOther, referer)
	}
	return c.Redirect(http.StatusSeeOther, "/admin/quotes")
}

// HandleUnarchiveQuoteDraft handles POST /admin/quotes/:id/unarchive
func (h *AdminHandler) HandleUnarchiveQuoteDraft(c echo.Context) error {
	ctx := c.Request().Context()
	draftID := c.Param("id")

	if draftID == "" {
		return c.String(http.StatusBadRequest, "Draft ID is required")
	}

	// Unarchive the draft
	err := h.storage.Queries.UnarchiveDraft(ctx, draftID)
	if err != nil {
		slog.Error("failed to unarchive draft", "error", err, "draft_id", draftID)
		return c.String(http.StatusInternalServerError, "Failed to unarchive quote draft")
	}

	slog.Info("unarchived quote draft", "draft_id", draftID)

	// Check if request wants JSON response (for HTMX)
	if c.Request().Header.Get("HX-Request") == "true" {
		return c.String(http.StatusOK, "")
	}

	// Redirect back to list or detail page
	referer := c.Request().Header.Get("Referer")
	if referer != "" {
		return c.Redirect(http.StatusSeeOther, referer)
	}
	return c.Redirect(http.StatusSeeOther, "/admin/quotes?status=archived")
}
