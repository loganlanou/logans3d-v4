package types

import "github.com/loganlanou/logans3d-v4/storage/db"

type ProductWithImage struct {
	Product  db.Product
	ImageURL string
}