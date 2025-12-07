package service

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/loganlanou/logans3d-v4/internal/auth"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
	giftcertviews "github.com/loganlanou/logans3d-v4/views/giftcertificates"
)

// RegisterGiftCertificateRoutes registers admin gift certificate routes
func (s *Service) RegisterGiftCertificateRoutes(g *echo.Group) {
	g.GET("/gift-certificates", s.handleAdminGiftCertificates)
	g.GET("/gift-certificates/new", s.handleAdminGiftCertificateNew)
	g.POST("/gift-certificates", s.handleAdminGiftCertificateCreate)
	g.GET("/gift-certificates/:id", s.handleAdminGiftCertificateDetail)
	g.POST("/gift-certificates/:id/redeem", s.handleAdminGiftCertificateRedeem)
	g.POST("/gift-certificates/:id/void", s.handleAdminGiftCertificateVoid)
	g.POST("/gift-certificates/:id/regenerate", s.handleAdminGiftCertificateRegenerate)
	g.GET("/gift-certificates/:id/image/png", s.handleAdminGiftCertificateImagePNG)
	g.GET("/gift-certificates/:id/image/pdf", s.handleAdminGiftCertificateImagePDF)

	// API endpoints
	api := g.Group("/api")
	api.GET("/gift-certificates/count", s.handleAdminGiftCertificatesCount)
	api.GET("/gift-certificates/stats", s.handleAdminGiftCertificatesStats)
}

// RegisterPublicGiftCertificateRoutes registers public verification routes
func (s *Service) RegisterPublicGiftCertificateRoutes(e *echo.Echo) {
	e.GET("/gift-certificates/verify/:id", s.handlePublicGiftCertificateVerify)
}

// handleAdminGiftCertificates lists all gift certificates
func (s *Service) handleAdminGiftCertificates(c echo.Context) error {
	ctx := c.Request().Context()

	certificates, err := s.storage.Queries.ListAllGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to list gift certificates", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to fetch gift certificates")
	}

	activeCount, err := s.storage.Queries.CountActiveGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to count active gift certificates", "error", err)
		activeCount = 0
	}

	redeemedCount, err := s.storage.Queries.CountRedeemedGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to count redeemed gift certificates", "error", err)
		redeemedCount = 0
	}

	activeSum, err := s.storage.Queries.SumActiveGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to sum active gift certificates", "error", err)
		activeSum = 0.0
	}

	redeemedSum, err := s.storage.Queries.SumRedeemedGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to sum redeemed gift certificates", "error", err)
		redeemedSum = 0.0
	}

	// Convert to view data
	var certData []admin.GiftCertificateRow
	for _, cert := range certificates {
		certData = append(certData, admin.GiftCertificateRow{
			ID:                  cert.ID,
			Amount:              cert.Amount,
			Reference:           cert.Reference,
			IssuedAt:            cert.IssuedAt,
			RedeemedAt:          cert.RedeemedAt,
			RedeemerName:        cert.RedeemerName,
			CreatedByName:       cert.CreatedByName,
			VoidedAt:            cert.VoidedAt,
			VoidReason:          cert.VoidReason,
			VoidedByName:        cert.VoidedByAdminName,
			RedeemedByAdminName: cert.RedeemedByAdminName,
		})
	}

	data := admin.GiftCertificatesPageData{
		Certificates:  certData,
		ActiveCount:   activeCount,
		RedeemedCount: redeemedCount,
		TotalActive:   toFloat64(activeSum),
		TotalRedeemed: toFloat64(redeemedSum),
	}

	return templ.Handler(admin.GiftCertificates(c, data)).Component.Render(ctx, c.Response().Writer)
}

// handleAdminGiftCertificateNew shows the create form
func (s *Service) handleAdminGiftCertificateNew(c echo.Context) error {
	return templ.Handler(admin.GiftCertificateNew(c)).Component.Render(c.Request().Context(), c.Response().Writer)
}

// handleAdminGiftCertificateCreate creates a new gift certificate
func (s *Service) handleAdminGiftCertificateCreate(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse form data
	amountStr := c.FormValue("amount")
	reference := c.FormValue("reference")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		slog.Error("invalid amount for gift certificate", "error", err, "amount_str", amountStr)
		return c.String(http.StatusBadRequest, "Invalid amount")
	}

	// Get current user
	user, _ := auth.GetDBUser(c)
	var createdByUserID sql.NullString
	if user != nil {
		createdByUserID = sql.NullString{String: user.ID, Valid: true}
	}

	// Generate UUID
	id := uuid.New().String()

	// Create certificate
	cert, err := s.storage.Queries.CreateGiftCertificate(ctx, db.CreateGiftCertificateParams{
		ID:              id,
		Amount:          amount,
		Reference:       sql.NullString{String: reference, Valid: reference != ""},
		CreatedByUserID: createdByUserID,
	})
	if err != nil {
		slog.Error("failed to create gift certificate", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create gift certificate")
	}

	// Generate images (reads BASE_URL from env)
	pngPath, pdfPath, err := GenerateGiftCertificateImages(cert)
	if err != nil {
		slog.Error("failed to generate gift certificate images", "error", err)
		// Continue without images, they can be regenerated
	} else {
		// Update certificate with image paths
		err = s.storage.Queries.UpdateGiftCertificateImages(ctx, db.UpdateGiftCertificateImagesParams{
			ImagePngPath: sql.NullString{String: pngPath, Valid: pngPath != ""},
			ImagePdfPath: sql.NullString{String: pdfPath, Valid: pdfPath != ""},
			ID:           id,
		})
		if err != nil {
			slog.Error("failed to update gift certificate images", "error", err)
		}
	}

	return c.Redirect(http.StatusSeeOther, "/admin/gift-certificates/"+id)
}

// handleAdminGiftCertificateDetail shows a single gift certificate
func (s *Service) handleAdminGiftCertificateDetail(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	cert, err := s.storage.Queries.GetGiftCertificateWithCreator(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Gift certificate not found")
		}
		slog.Error("failed to get gift certificate", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to fetch gift certificate")
	}

	data := admin.GiftCertificateDetailData{
		Certificate: admin.GiftCertificateRow{
			ID:              cert.ID,
			Amount:          cert.Amount,
			Reference:       cert.Reference,
			IssuedAt:        cert.IssuedAt,
			RedeemedAt:      cert.RedeemedAt,
			RedeemerName:    cert.RedeemerName,
			RedemptionNotes: cert.RedemptionNotes,
			CreatedByName:   cert.CreatedByName,
			ImagePngPath:    cert.ImagePngPath,
			ImagePdfPath:    cert.ImagePdfPath,
			UpdatedAt:       cert.UpdatedAt,
			VoidedAt:        cert.VoidedAt,
			VoidReason:      cert.VoidReason,
			VoidedByName:    cert.VoidedByAdminName,
		},
		RedeemedByAdminName: cert.RedeemedByAdminName,
		IsRedeemed:          cert.RedeemedAt.Valid,
		IsVoided:            cert.VoidedAt.Valid,
	}

	return templ.Handler(admin.GiftCertificateDetail(c, data)).Component.Render(ctx, c.Response().Writer)
}

// handleAdminGiftCertificateRedeem marks a gift certificate as redeemed
func (s *Service) handleAdminGiftCertificateRedeem(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	// Get form data
	redeemerName := c.FormValue("redeemer_name")
	if redeemerName == "" {
		return c.String(http.StatusBadRequest, "Redeemer name is required")
	}
	notes := c.FormValue("notes")

	// Get current admin user
	user, _ := auth.GetDBUser(c)
	var redeemedByUserID sql.NullString
	if user != nil {
		redeemedByUserID = sql.NullString{String: user.ID, Valid: true}
	}

	// Redeem the certificate
	err := s.storage.Queries.RedeemGiftCertificate(ctx, db.RedeemGiftCertificateParams{
		RedeemedByUserID: redeemedByUserID,
		RedeemerName:     sql.NullString{String: redeemerName, Valid: redeemerName != ""},
		RedemptionNotes:  sql.NullString{String: notes, Valid: notes != ""},
		ID:               id,
	})
	if err != nil {
		slog.Error("failed to redeem gift certificate", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to redeem gift certificate")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/gift-certificates/"+id)
}

// handleAdminGiftCertificateVoid marks a gift certificate as voided/invalid
func (s *Service) handleAdminGiftCertificateVoid(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	// Get form data
	voidReason := c.FormValue("void_reason")
	if voidReason == "" {
		return c.String(http.StatusBadRequest, "Void reason is required")
	}

	// Get current admin user
	user, _ := auth.GetDBUser(c)
	var voidedByUserID sql.NullString
	if user != nil {
		voidedByUserID = sql.NullString{String: user.ID, Valid: true}
	}

	// Void the certificate
	err := s.storage.Queries.VoidGiftCertificate(ctx, db.VoidGiftCertificateParams{
		VoidedByUserID: voidedByUserID,
		VoidReason:     sql.NullString{String: voidReason, Valid: true},
		ID:             id,
	})
	if err != nil {
		slog.Error("failed to void gift certificate", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to void gift certificate")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/gift-certificates/"+id)
}

// handleAdminGiftCertificateRegenerate regenerates the certificate image
func (s *Service) handleAdminGiftCertificateRegenerate(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	cert, err := s.storage.Queries.GetGiftCertificate(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.String(http.StatusNotFound, "Gift certificate not found")
		}
		slog.Error("failed to get gift certificate for regeneration", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to fetch gift certificate")
	}

	pngPath, pdfPath, err := GenerateGiftCertificateImages(cert)
	if err != nil {
		slog.Error("failed to regenerate gift certificate images", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to regenerate images: "+err.Error())
	}

	err = s.storage.Queries.UpdateGiftCertificateImages(ctx, db.UpdateGiftCertificateImagesParams{
		ImagePngPath: sql.NullString{String: pngPath, Valid: pngPath != ""},
		ImagePdfPath: sql.NullString{String: pdfPath, Valid: pdfPath != ""},
		ID:           id,
	})
	if err != nil {
		slog.Error("failed to update gift certificate images", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to save image paths")
	}

	return c.Redirect(http.StatusSeeOther, "/admin/gift-certificates/"+id)
}

// handleAdminGiftCertificateImagePNG serves the PNG image
func (s *Service) handleAdminGiftCertificateImagePNG(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	cert, err := s.storage.Queries.GetGiftCertificate(ctx, id)
	if err != nil {
		slog.Error("failed to get gift certificate for PNG", "error", err, "id", id)
		return c.String(http.StatusNotFound, "Gift certificate not found")
	}

	if !cert.ImagePngPath.Valid || cert.ImagePngPath.String == "" {
		return c.String(http.StatusNotFound, "Image not generated")
	}

	// Check if download is requested
	if c.QueryParam("download") == "1" {
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gift-certificate-%s.png", id[:8]))
	}

	return c.File(cert.ImagePngPath.String)
}

// handleAdminGiftCertificateImagePDF serves the PDF
func (s *Service) handleAdminGiftCertificateImagePDF(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	cert, err := s.storage.Queries.GetGiftCertificate(ctx, id)
	if err != nil {
		slog.Error("failed to get gift certificate for PDF", "error", err, "id", id)
		return c.String(http.StatusNotFound, "Gift certificate not found")
	}

	if !cert.ImagePdfPath.Valid || cert.ImagePdfPath.String == "" {
		return c.String(http.StatusNotFound, "PDF not generated")
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gift-certificate-%s.pdf", id[:8]))
	return c.File(cert.ImagePdfPath.String)
}

// handlePublicGiftCertificateVerify shows public verification page
func (s *Service) handlePublicGiftCertificateVerify(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	cert, err := s.storage.Queries.GetGiftCertificate(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			data := giftcertviews.VerifyPageData{
				NotFound: true,
			}
			return templ.Handler(giftcertviews.Verify(c, data)).Component.Render(ctx, c.Response().Writer)
		}
		slog.Error("failed to get gift certificate for verification", "error", err, "id", id)
		return c.String(http.StatusInternalServerError, "Failed to fetch gift certificate")
	}

	// Check if user is admin
	user, _ := auth.GetDBUser(c)
	isAdmin := user != nil && user.IsAdmin

	data := giftcertviews.VerifyPageData{
		ID:             cert.ID,
		Amount:         cert.Amount,
		Reference:      cert.Reference.String,
		IssuedAt:       cert.IssuedAt,
		IsRedeemed:     cert.RedeemedAt.Valid,
		RedeemedAt:     cert.RedeemedAt,
		IsVoided:       cert.VoidedAt.Valid,
		IsAdmin:        isAdmin,
		AdminDetailURL: "/admin/gift-certificates/" + id,
	}

	return templ.Handler(giftcertviews.Verify(c, data)).Component.Render(ctx, c.Response().Writer)
}

// handleAdminGiftCertificatesCount returns count for dashboard
func (s *Service) handleAdminGiftCertificatesCount(c echo.Context) error {
	ctx := c.Request().Context()

	count, err := s.storage.Queries.CountGiftCertificates(ctx)
	if err != nil {
		slog.Error("failed to count gift certificates", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to count gift certificates"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"count": count})
}

// handleAdminGiftCertificatesStats returns stats for dashboard
func (s *Service) handleAdminGiftCertificatesStats(c echo.Context) error {
	ctx := c.Request().Context()

	activeCount, _ := s.storage.Queries.CountActiveGiftCertificates(ctx)
	redeemedCount, _ := s.storage.Queries.CountRedeemedGiftCertificates(ctx)
	activeSum, _ := s.storage.Queries.SumActiveGiftCertificates(ctx)
	redeemedSum, _ := s.storage.Queries.SumRedeemedGiftCertificates(ctx)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"active_count":   activeCount,
		"redeemed_count": redeemedCount,
		"active_sum":     toFloat64(activeSum),
		"redeemed_sum":   toFloat64(redeemedSum),
	})
}

// toFloat64 converts interface{} to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
