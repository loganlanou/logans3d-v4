package types

import (
	"github.com/loganlanou/logans3d-v4/internal/social"
	"github.com/loganlanou/logans3d-v4/storage/db"
)

type ProductWithImage struct {
	Product        db.Product
	ImageURL       string
	IsNew          bool // Product created within last 60 days
	IsDiscontinued bool // Product is inactive (won't show on site)
}

type ProductWithStatus struct {
	Product           db.Product
	CategoryName      string
	PlatformsPosted   int64
	PlatformsPending  int64
	TotalPlatforms    int64
	HasGeneratedPosts bool
}

type GeneratedPostData struct {
	ProductID string
	Platform  social.Platform
	PostCopy  string
	Hashtags  string
	ShareURL  string
	Status    string
}
