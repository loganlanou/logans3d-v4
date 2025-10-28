package handlers

import (
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
		return c.String(http.StatusInternalServerError, "Failed to load campaigns")
	}

	return admin.Promotions(c, campaigns, page).Render(c.Request().Context(), c.Response().Writer)
}

// HandlePromotionDetail shows detail for a single campaign
func (h *AdminPromotionsHandler) HandlePromotionDetail(c echo.Context) error {
	ctx := c.Request().Context()
	campaignID := c.Param("id")

	// Get campaign
	campaign, err := h.queries.GetPromotionCampaignByID(ctx, campaignID)
	if err != nil {
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

	// Get stats
	stats, err := h.queries.GetPromotionCodeStats(ctx, campaignID)
	if err != nil {
		stats = db.GetPromotionCodeStatsRow{}
	}

	return admin.PromotionDetail(c, campaign, codes, stats).Render(c.Request().Context(), c.Response().Writer)
}
