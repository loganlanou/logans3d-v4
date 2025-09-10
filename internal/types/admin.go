package types

import "github.com/loganlanou/logans3d-v4/storage/db"

type ProductWithImage struct {
	Product       db.Product
	ImageURL      string
	IsNew         bool // Product created within last 60 days
	IsDiscontinued bool // Product is inactive (won't show on site)
}