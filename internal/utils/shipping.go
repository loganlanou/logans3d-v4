package utils

// Shipping time message constants - change these to update site-wide
const (
	// ShippingTimeInStock is displayed when stock > 0
	ShippingTimeInStock = "Ships in 1-3 days"

	// ShippingTimeOutOfStock is displayed when stock = 0 (needs printing)
	ShippingTimeOutOfStock = "Ships in 4-5 days"

	// Short versions for compact displays
	ShippingTimeInStockShort    = "1-3 days"
	ShippingTimeOutOfStockShort = "4-5 days"
)

// ShippingTimeMessage returns the appropriate shipping message based on stock
func ShippingTimeMessage(stockQuantity int64) string {
	if stockQuantity > 0 {
		return ShippingTimeInStock
	}
	return ShippingTimeOutOfStock
}

// ShippingTimeShort returns the short version of shipping time
func ShippingTimeShort(stockQuantity int64) string {
	if stockQuantity > 0 {
		return ShippingTimeInStockShort
	}
	return ShippingTimeOutOfStockShort
}

// NeedsPrinting returns true if the item needs to be printed (no stock)
func NeedsPrinting(stockQuantity int64) bool {
	return stockQuantity <= 0
}
