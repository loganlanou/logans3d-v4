package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/social"
	"github.com/loganlanou/logans3d-v4/internal/types"
	"github.com/loganlanou/logans3d-v4/storage/db"
	"github.com/loganlanou/logans3d-v4/views/admin"
)

func (h *AdminHandler) HandleAdminSocialMedia(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := h.storage.Queries.ListProductsWithPostingStatus(ctx)
	if err != nil {
		slog.Error("failed to list products with posting status", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}

	productsWithStatus := make([]types.ProductWithStatus, 0, len(products))
	for _, p := range products {
		product, err := h.storage.Queries.GetProduct(ctx, p.ID)
		if err != nil {
			slog.Debug("failed to get product", "error", err, "product_id", p.ID)
			continue
		}

		categoryName := "Products"
		if p.CategoryName.Valid {
			categoryName = p.CategoryName.String
		}

		productsWithStatus = append(productsWithStatus, types.ProductWithStatus{
			Product:           product,
			CategoryName:      categoryName,
			PlatformsPosted:   p.PlatformsPosted,
			PlatformsPending:  p.PlatformsPending,
			TotalPlatforms:    p.TotalPlatforms,
			HasGeneratedPosts: p.TotalPlatforms > 0,
		})
	}

	return admin.SocialMediaDashboard(c, productsWithStatus).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) HandleGeneratePostsForProduct(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("product_id")

	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Product ID is required")
	}

	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	categoryName := "Products"
	if product.CategoryID.Valid {
		category, err := h.storage.Queries.GetCategory(ctx, product.CategoryID.String)
		if err != nil {
			slog.Debug("failed to get category", "error", err, "category_id", product.CategoryID.String)
		} else {
			categoryName = category.Name
		}
	}

	images, err := h.storage.Queries.GetProductImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product images", "error", err, "product_id", productID)
	}

	primaryImage := "/public/images/products/default.jpg"
	if len(images) > 0 {
		for _, img := range images {
			if img.IsPrimary.Valid && img.IsPrimary.Bool {
				primaryImage = "/public/images/products/" + img.ImageUrl
				break
			}
		}
		if primaryImage == "/public/images/products/default.jpg" && len(images) > 0 {
			primaryImage = "/public/images/products/" + images[0].ImageUrl
		}
	}

	bestSellers, err := h.storage.Queries.GetBestSellingProducts(ctx, 20)
	if err != nil {
		slog.Debug("failed to get best sellers", "error", err)
		bestSellers = []db.GetBestSellingProductsRow{}
	}

	isBestSeller := false
	for _, bs := range bestSellers {
		if bs.ID == productID {
			isBestSeller = true
			break
		}
	}

	baseURL := os.Getenv("SITE_URL")
	if baseURL == "" {
		baseURL = "https://www.logans3dcreations.com"
	}

	description := ""
	if product.Description.Valid {
		description = product.Description.String
	}
	shortDescription := ""
	if product.ShortDescription.Valid {
		shortDescription = product.ShortDescription.String
	}

	productData := social.ProductData{
		ID:               product.ID,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      description,
		ShortDescription: shortDescription,
		PriceCents:       product.PriceCents,
		CategoryName:     categoryName,
		ImageURL:         primaryImage,
		IsBestSeller:     isBestSeller,
	}

	generatedPosts := social.GeneratePostsForProduct(productData, baseURL)

	for _, post := range generatedPosts {
		existingPost, err := h.storage.Queries.GetSocialMediaPostByProductAndPlatform(ctx, db.GetSocialMediaPostByProductAndPlatformParams{
			ProductID: productID,
			Platform:  string(post.Platform),
		})

		if err == sql.ErrNoRows {
			_, err = h.storage.Queries.CreateSocialMediaPost(ctx, db.CreateSocialMediaPostParams{
				ID:        uuid.New().String(),
				ProductID: productID,
				Platform:  string(post.Platform),
				PostCopy:  post.PostCopy,
				Hashtags:  sql.NullString{String: post.Hashtags, Valid: post.Hashtags != ""},
			})
			if err != nil {
				slog.Error("failed to create social media post", "error", err, "product_id", productID, "platform", post.Platform)
			}
		} else if err == nil {
			err = h.storage.Queries.UpdateSocialMediaPost(ctx, db.UpdateSocialMediaPostParams{
				ID:       existingPost.ID,
				PostCopy: post.PostCopy,
				Hashtags: sql.NullString{String: post.Hashtags, Valid: post.Hashtags != ""},
			})
			if err != nil {
				slog.Error("failed to update social media post", "error", err, "post_id", existingPost.ID)
			}
		} else {
			slog.Error("failed to check for existing post", "error", err, "product_id", productID, "platform", post.Platform)
		}

		_, err = h.storage.Queries.GetSocialMediaTaskByProductAndPlatform(ctx, db.GetSocialMediaTaskByProductAndPlatformParams{
			ProductID: productID,
			Platform:  string(post.Platform),
		})

		if err == sql.ErrNoRows {
			_, err = h.storage.Queries.CreateSocialMediaTask(ctx, db.CreateSocialMediaTaskParams{
				ID:        uuid.New().String(),
				ProductID: productID,
				Platform:  string(post.Platform),
				Status:    "pending",
			})
			if err != nil {
				slog.Error("failed to create social media task", "error", err, "product_id", productID, "platform", post.Platform)
			}
		} else if err != nil {
			slog.Error("failed to check for existing task", "error", err, "product_id", productID, "platform", post.Platform)
		}
	}

	c.Response().Header().Set("HX-Redirect", "/admin/social-media/product/"+productID)
	return c.NoContent(http.StatusOK)
}

func (h *AdminHandler) HandleSocialMediaProductView(c echo.Context) error {
	ctx := c.Request().Context()
	productID := c.Param("product_id")

	if productID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Product ID is required")
	}

	product, err := h.storage.Queries.GetProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get product", "error", err, "product_id", productID)
		return echo.NewHTTPError(http.StatusNotFound, "Product not found")
	}

	categoryName := "Products"
	if product.CategoryID.Valid {
		category, err := h.storage.Queries.GetCategory(ctx, product.CategoryID.String)
		if err != nil {
			slog.Debug("failed to get category", "error", err, "category_id", product.CategoryID.String)
		} else {
			categoryName = category.Name
		}
	}

	images, err := h.storage.Queries.GetProductImages(ctx, productID)
	if err != nil {
		slog.Debug("failed to get product images", "error", err, "product_id", productID)
	}

	primaryImage := "/public/images/products/default.jpg"
	if len(images) > 0 {
		for _, img := range images {
			if img.IsPrimary.Valid && img.IsPrimary.Bool {
				primaryImage = "/public/images/products/" + img.ImageUrl
				break
			}
		}
		if primaryImage == "/public/images/products/default.jpg" && len(images) > 0 {
			primaryImage = "/public/images/products/" + images[0].ImageUrl
		}
	}

	posts, err := h.storage.Queries.GetSocialMediaPostsByProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get social media posts", "error", err, "product_id", productID)
		posts = []db.SocialMediaPost{}
	}

	tasks, err := h.storage.Queries.GetSocialMediaTasksByProduct(ctx, productID)
	if err != nil {
		slog.Error("failed to get social media tasks", "error", err, "product_id", productID)
		tasks = []db.SocialMediaTask{}
	}

	taskMap := make(map[string]db.SocialMediaTask)
	for _, task := range tasks {
		taskMap[task.Platform] = task
	}

	generatedPosts := make([]types.GeneratedPostData, 0, len(posts))
	for _, post := range posts {
		task, exists := taskMap[post.Platform]
		status := "pending"
		if exists {
			status = task.Status
		}

		baseURL := os.Getenv("SITE_URL")
		if baseURL == "" {
			baseURL = "https://www.logans3dcreations.com"
		}
		productURL := baseURL + "/shop/product/" + product.Slug
		imageURL := baseURL + primaryImage

		shareURL := social.GenerateShareURL(social.Platform(post.Platform), productURL, imageURL, post.PostCopy)

		generatedPosts = append(generatedPosts, types.GeneratedPostData{
			ProductID: productID,
			Platform:  social.Platform(post.Platform),
			PostCopy:  post.PostCopy,
			Hashtags:  post.Hashtags.String,
			ShareURL:  shareURL,
			Status:    status,
		})
	}

	return admin.SocialMediaProductView(c, product, categoryName, primaryImage, generatedPosts).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AdminHandler) HandleUpdatePostStatus(c echo.Context) error {
	ctx := c.Request().Context()

	productID := c.FormValue("product_id")
	platform := c.FormValue("platform")
	status := c.FormValue("status")

	if productID == "" || platform == "" || status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required fields")
	}

	if status != "pending" && status != "posted" && status != "skipped" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid status")
	}

	task, err := h.storage.Queries.GetSocialMediaTaskByProductAndPlatform(ctx, db.GetSocialMediaTaskByProductAndPlatformParams{
		ProductID: productID,
		Platform:  platform,
	})

	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "Task not found")
	} else if err != nil {
		slog.Error("failed to get task", "error", err, "product_id", productID, "platform", platform)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get task")
	}

	postedAt := sql.NullTime{}
	if status == "posted" {
		postedAt = sql.NullTime{Valid: true, Time: time.Now()}
	}

	err = h.storage.Queries.UpdateSocialMediaTaskStatus(ctx, db.UpdateSocialMediaTaskStatusParams{
		ID:       task.ID,
		Status:   status,
		PostedAt: postedAt,
	})

	if err != nil {
		slog.Error("failed to update task status", "error", err, "task_id", task.ID, "status", status)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update status")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (h *AdminHandler) HandleBulkGeneratePosts(c echo.Context) error {
	ctx := c.Request().Context()

	products, err := h.storage.Queries.ListProducts(ctx)
	if err != nil {
		slog.Error("failed to list products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load products")
	}

	count := 0
	for _, product := range products {
		if !product.IsActive.Valid || !product.IsActive.Bool {
			continue
		}

		existingPosts, err := h.storage.Queries.GetSocialMediaPostsByProduct(ctx, product.ID)
		if err == nil && len(existingPosts) > 0 {
			continue
		}

		categoryName := "Products"
		if product.CategoryID.Valid {
			category, err := h.storage.Queries.GetCategory(ctx, product.CategoryID.String)
			if err != nil {
				slog.Debug("failed to get category", "error", err, "category_id", product.CategoryID.String)
			} else {
				categoryName = category.Name
			}
		}

		images, err := h.storage.Queries.GetProductImages(ctx, product.ID)
		if err != nil {
			slog.Debug("failed to get product images", "error", err, "product_id", product.ID)
		}

		primaryImage := "/public/images/products/default.jpg"
		if len(images) > 0 {
			for _, img := range images {
				if img.IsPrimary.Valid && img.IsPrimary.Bool {
					primaryImage = "/public/images/products/" + img.ImageUrl
					break
				}
			}
			if primaryImage == "/public/images/products/default.jpg" && len(images) > 0 {
				primaryImage = "/public/images/products/" + images[0].ImageUrl
			}
		}

		bestSellers, err := h.storage.Queries.GetBestSellingProducts(ctx, 20)
		if err != nil {
			slog.Debug("failed to get best sellers", "error", err)
			bestSellers = []db.GetBestSellingProductsRow{}
		}

		isBestSeller := false
		for _, bs := range bestSellers {
			if bs.ID == product.ID {
				isBestSeller = true
				break
			}
		}

		baseURL := os.Getenv("SITE_URL")
		if baseURL == "" {
			baseURL = "https://www.logans3dcreations.com"
		}

		description := ""
		if product.Description.Valid {
			description = product.Description.String
		}
		shortDescription := ""
		if product.ShortDescription.Valid {
			shortDescription = product.ShortDescription.String
		}

		productData := social.ProductData{
			ID:               product.ID,
			Name:             product.Name,
			Slug:             product.Slug,
			Description:      description,
			ShortDescription: shortDescription,
			PriceCents:       product.PriceCents,
			CategoryName:     categoryName,
			ImageURL:         primaryImage,
			IsBestSeller:     isBestSeller,
		}

		generatedPosts := social.GeneratePostsForProduct(productData, baseURL)

		for _, post := range generatedPosts {
			_, err = h.storage.Queries.CreateSocialMediaPost(ctx, db.CreateSocialMediaPostParams{
				ID:        uuid.New().String(),
				ProductID: product.ID,
				Platform:  string(post.Platform),
				PostCopy:  post.PostCopy,
				Hashtags:  sql.NullString{String: post.Hashtags, Valid: post.Hashtags != ""},
			})
			if err != nil {
				slog.Error("failed to create social media post", "error", err, "product_id", product.ID, "platform", post.Platform)
			}

			_, err = h.storage.Queries.CreateSocialMediaTask(ctx, db.CreateSocialMediaTaskParams{
				ID:        uuid.New().String(),
				ProductID: product.ID,
				Platform:  string(post.Platform),
				Status:    "pending",
			})
			if err != nil {
				slog.Error("failed to create social media task", "error", err, "product_id", product.ID, "platform", post.Platform)
			}
		}

		count++
	}

	c.Response().Header().Set("HX-Redirect", "/admin/social-media")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Generated posts for " + string(rune(count)) + " products",
	})
}

func (h *AdminHandler) HandleDeleteAllPendingPosts(c echo.Context) error {
	ctx := c.Request().Context()

	err := h.storage.Queries.DeleteAllPendingPosts(ctx)
	if err != nil {
		slog.Error("failed to delete pending posts", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pending posts")
	}

	err = h.storage.Queries.DeleteAllPendingTasks(ctx)
	if err != nil {
		slog.Error("failed to delete pending tasks", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pending tasks")
	}

	c.Response().Header().Set("HX-Redirect", "/admin/social-media")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Deleted all pending posts",
	})
}
