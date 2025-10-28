package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

type AdminPromotionsHandler struct {
	queries *db.Queries
}

func NewAdminPromotionsHandler(queries *db.Queries) *AdminPromotionsHandler {
	return &AdminPromotionsHandler{
		queries: queries,
	}
}

// HandlePromotionsList shows all promotion campaigns
func (h *AdminPromotionsHandler) HandlePromotionsList(c echo.Context) error {
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

	// Get campaigns
	campaigns, err := h.queries.GetAllPromotionCampaigns(ctx, db.GetAllPromotionCampaignsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.Error("failed to get promotion campaigns", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load campaigns")
	}

	// Get composite stats across all active campaigns
	overallStats, err := h.queries.GetActivePromotionsOverallStats(ctx)
	if err != nil {
		overallStats = db.GetActivePromotionsOverallStatsRow{}
	}

	totalEmailsToNonUsers, err := h.queries.CountTotalEmailsToNonUsersActive(ctx)
	if err != nil {
		totalEmailsToNonUsers = 0
	}

	totalActiveCodes, err := h.queries.CountTotalActiveCodesAcrossActive(ctx)
	if err != nil {
		totalActiveCodes = 0
	}

	// Combine composite stats
	compositeStats := admin.CompositePromotionStats{
		TotalCodesIssued:      overallStats.TotalCodesIssued,
		TotalCodesRedeemed:    overallStats.TotalCodesRedeemed,
		OverallRedemptionRate: overallStats.OverallRedemptionRate,
		TotalEmailsToNonUsers: totalEmailsToNonUsers,
		TotalActiveCodes:      totalActiveCodes,
	}

	return admin.Promotions(c, campaigns, page, compositeStats).Render(c.Request().Context(), c.Response().Writer)
}

// HandlePromotionDetail shows detail for a single campaign
func (h *AdminPromotionsHandler) HandlePromotionDetail(c echo.Context) error {
	ctx := c.Request().Context()
	campaignID := c.Param("id")

	// Get campaign
	campaign, err := h.queries.GetPromotionCampaignByID(ctx, campaignID)
	if err != nil {
		slog.Error("failed to get promotion campaign by ID", "error", err, "campaign_id", campaignID)
		return c.String(http.StatusNotFound, "Campaign not found")
	}

	// Get codes for this campaign
	codes, err := h.queries.GetPromotionCodesByCampaign(ctx, db.GetPromotionCodesByCampaignParams{
		CampaignID: campaignID,
		Limit:      100,
		Offset:     0,
	})
	if err != nil {
		codes = []db.PromotionCode{}
	}

	// Get basic stats
	basicStats, err := h.queries.GetPromotionCodeStats(ctx, campaignID)
	if err != nil {
		basicStats = db.GetPromotionCodeStatsRow{}
	}

	// Get emails to non-users count
	emailsToNonUsers, err := h.queries.CountEmailsToNonUsers(ctx, campaignID)
	if err != nil {
		emailsToNonUsers = 0
	}

	// Get active codes stats
	activeStats, err := h.queries.GetActiveCodesStats(ctx, campaignID)
	if err != nil {
		activeStats = db.GetActiveCodesStatsRow{}
	}

	// Combine all stats
	combinedStats := admin.CombinedPromotionStats{
		TotalCodes:            basicStats.TotalCodes,
		UsedCodes:             basicStats.UsedCodes,
		TotalUses:             basicStats.TotalUses,
		EmailsToNonUsers:      emailsToNonUsers,
		ActiveCodesIssued:     activeStats.ActiveCodesIssued,
		ActiveCodesRedeemed:   activeStats.ActiveCodesRedeemed,
		RedemptionRatePercent: activeStats.RedemptionRatePercent,
	}

	return admin.PromotionDetail(c, campaign, codes, combinedStats).Render(c.Request().Context(), c.Response().Writer)
}

// HandlePopupStatus checks if popup has been shown to an email
func (h *AdminPromotionsHandler) HandlePopupStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Get email from query parameter
	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(http.StatusOK, map[string]bool{"shown": false})
	}

	// Check if popup has been shown
	popupShownAt, err := h.queries.CheckPopupShownForEmail(ctx, email)
	if err != nil {
		// Email not found in database, popup not shown
		return c.JSON(http.StatusOK, map[string]bool{"shown": false})
	}

	// If popup_shown_at is set, popup has been shown
	shown := popupShownAt.Valid
	return c.JSON(http.StatusOK, map[string]bool{"shown": shown})
}
