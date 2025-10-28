package helpers

import (
	"database/sql"
	"fmt"
	"time"
)

// FormatInt formats an integer as a string
func FormatInt(n int64) string {
	return fmt.Sprintf("%d", n)
}

// FormatNullInt64 formats a sql.NullInt64, returning default value if null
func FormatNullInt64(n sql.NullInt64, defaultVal string) string {
	if n.Valid {
		return fmt.Sprintf("%d", n.Int64)
	}
	return defaultVal
}

// FormatNullFloat64 formats a sql.NullFloat64 as integer, returning default value if null
func FormatNullFloat64AsInt(n sql.NullFloat64, defaultVal string) string {
	if n.Valid {
		return fmt.Sprintf("%.0f", n.Float64)
	}
	return defaultVal
}

// FormatPrice formats cents as dollars (e.g., 1599 -> "$15.99")
func FormatPrice(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

// FormatPercentage formats an integer as a percentage (e.g., 15 -> "15%")
func FormatPercentage(n int64) string {
	return fmt.Sprintf("%d%%", n)
}

// FormatDate formats a time.Time as "Jan 2, 2006"
func FormatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatNullTime formats a sql.NullTime, returning default value if null
func FormatNullTime(t sql.NullTime, layout string, defaultVal string) string {
	if t.Valid {
		return t.Time.Format(layout)
	}
	return defaultVal
}

// FormatDateShort is a convenience for "Jan 2, 2006" format
func FormatDateShort(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatDateTime formats a time.Time as "Jan 2, 2006 3:04 PM"
func FormatDateTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

// FormatFloat formats a float with specified decimal places
func FormatFloat(f float64, decimals int) string {
	return fmt.Sprintf("%.*f", decimals, f)
}
