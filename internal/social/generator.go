package social

import (
	"fmt"
	"net/url"
	"strings"
)

type Platform string

const (
	PlatformFacebook  Platform = "facebook"
	PlatformInstagram Platform = "instagram"
	PlatformTwitter   Platform = "twitter"
	PlatformPinterest Platform = "pinterest"
)

var AllPlatforms = []Platform{
	PlatformFacebook,
	PlatformInstagram,
	PlatformTwitter,
	PlatformPinterest,
}

type ProductData struct {
	ID               string
	Name             string
	Slug             string
	Description      string
	ShortDescription string
	PriceCents       int64
	CategoryName     string
	ImageURL         string
	ProductURL       string
	IsBestSeller     bool
}

type PostTemplate struct {
	Hook       string
	Body       string
	Price      string
	CTA        string
	Hashtags   []string
	MaxLength  int
	EmojiHeavy bool
}

type GeneratedPost struct {
	Platform Platform
	PostCopy string
	Hashtags string
	ShareURL string
}

func GeneratePostsForProduct(product ProductData, baseURL string) []GeneratedPost {
	posts := make([]GeneratedPost, 0, len(AllPlatforms))

	for _, platform := range AllPlatforms {
		post := GeneratePost(product, platform, baseURL)
		posts = append(posts, post)
	}

	return posts
}

func GeneratePost(product ProductData, platform Platform, baseURL string) GeneratedPost {
	productURL := fmt.Sprintf("%s/shop/product/%s", baseURL, product.Slug)
	imageURL := fmt.Sprintf("%s%s", baseURL, product.ImageURL)

	var postCopy string
	var hashtags string

	switch platform {
	case PlatformFacebook:
		postCopy = generateFacebookPost(product)
		hashtags = strings.Join(getFacebookHashtags(product), " ")
	case PlatformInstagram:
		postCopy = generateInstagramPost(product)
		hashtags = strings.Join(getInstagramHashtags(product), " ")
	case PlatformTwitter:
		postCopy = generateTwitterPost(product, productURL)
		hashtags = strings.Join(getTwitterHashtags(product), " ")
	case PlatformPinterest:
		postCopy = generatePinterestPost(product)
		hashtags = strings.Join(getPinterestHashtags(product), " ")
	}

	shareURL := GenerateShareURL(platform, productURL, imageURL, postCopy)

	return GeneratedPost{
		Platform: platform,
		PostCopy: postCopy,
		Hashtags: hashtags,
		ShareURL: shareURL,
	}
}

func generateFacebookPost(product ProductData) string {
	emoji := getProductEmoji(product.CategoryName)
	bestSellerText := ""
	if product.IsBestSeller {
		bestSellerText = " - one of our bestsellers"
	}

	priceText := formatPrice(product.PriceCents)

	template := fmt.Sprintf(`%s Just finished printing this incredible %s!
💰 %s

Every detail meticulously crafted in high-quality PLA+. Museum-quality finish with hand-selected colors and careful post-processing to bring out the finest details.

Perfect for collectors, educators, or anyone who loves %s%s!

🚚 Made in Cadott, WI - Shipped nationwide

Which one should we print next? Drop your vote in the comments! 👇`,
		emoji,
		product.Name,
		priceText,
		strings.ToLower(product.CategoryName),
		bestSellerText,
	)

	return template
}

func generateInstagramPost(product ProductData) string {
	emoji := getProductEmoji(product.CategoryName)
	priceText := formatPrice(product.PriceCents)

	bestSellerBadge := ""
	if product.IsBestSeller {
		bestSellerBadge = " 🔥"
	}

	description := product.ShortDescription
	if description == "" {
		description = truncateText(product.Description, 80)
	}

	template := fmt.Sprintf(`%s %s%s
💵 %s | Link in bio

%s

🏠 Handcrafted in Wisconsin
📦 Ships nationwide

✨ Every piece is hand-finished for museum-quality results`,
		emoji,
		product.Name,
		bestSellerBadge,
		priceText,
		description,
	)

	return template
}

func generateTwitterPost(product ProductData, productURL string) string {
	emoji := getProductEmoji(product.CategoryName)
	priceText := formatPrice(product.PriceCents)

	bestSellerBadge := ""
	if product.IsBestSeller {
		bestSellerBadge = " 🔥"
	}

	template := fmt.Sprintf(`%s New: %s%s

Museum-quality 3D print | %s
Handcrafted in Wisconsin 🏠`,
		emoji,
		product.Name,
		bestSellerBadge,
		priceText,
	)

	maxLength := 280 - len(productURL) - 5
	if len(template) > maxLength {
		template = truncateText(template, maxLength)
	}

	return template
}

func generatePinterestPost(product ProductData) string {
	emoji := getProductEmoji(product.CategoryName)
	priceText := formatPrice(product.PriceCents)

	description := product.Description
	if description == "" {
		description = product.ShortDescription
	}

	template := fmt.Sprintf(`%s %s - %s

%s

Perfect gift for %s lovers! Museum-quality 3D printed with incredible detail. Hand-finished and carefully crafted in Wisconsin.

🎁 Great for: Collectors, educators, gifts, home decor, STEM enthusiasts
📐 Precision 3D printing with hand-selected colors
🚚 Ships nationwide from Cadott, WI`,
		emoji,
		product.Name,
		priceText,
		truncateText(description, 150),
		strings.ToLower(product.CategoryName),
	)

	return template
}

func getFacebookHashtags(product ProductData) []string {
	base := []string{
		"#3DPrinting",
		"#Handmade",
		"#STEM",
		"#LogansCreations",
		"#MadeInWisconsin",
	}

	categoryTag := fmt.Sprintf("#%s", strings.ReplaceAll(product.CategoryName, " ", ""))
	base = append(base, categoryTag)

	return base
}

func getInstagramHashtags(product ProductData) []string {
	base := []string{
		"#3DPrinting",
		"#3DPrintedArt",
		"#Handmade",
		"#CollectorItems",
		"#STEM",
		"#Educational",
		"#MakerMovement",
		"#SmallBusiness",
		"#MadeInWisconsin",
		"#MuseumQuality",
		"#GiftIdeas",
		"#UniqueGifts",
		"#HomeDecor",
		"#LogansCreations",
	}

	categoryTag := fmt.Sprintf("#%s", strings.ReplaceAll(product.CategoryName, " ", ""))
	base = append(base, categoryTag)

	if strings.Contains(strings.ToLower(product.CategoryName), "dinosaur") {
		base = append(base, "#DinosaurCollector", "#DinosaurToys", "#DinosaurLover")
	}

	return base
}

func getTwitterHashtags(product ProductData) []string {
	base := []string{
		"#3DPrinting",
		"#Handmade",
	}

	categoryTag := fmt.Sprintf("#%s", strings.ReplaceAll(product.CategoryName, " ", ""))
	base = append(base, categoryTag)

	return base
}

func getPinterestHashtags(product ProductData) []string {
	base := []string{
		"#3DPrinting",
		"#Handmade",
		"#GiftIdeas",
		"#HomeDecor",
		"#UniqueGifts",
		"#STEM",
		"#Educational",
	}

	categoryTag := fmt.Sprintf("#%s", strings.ReplaceAll(product.CategoryName, " ", ""))
	base = append(base, categoryTag)

	return base
}

func GenerateShareURL(platform Platform, productURL, imageURL, postText string) string {
	switch platform {
	case PlatformFacebook:
		return fmt.Sprintf("https://www.facebook.com/sharer/sharer.php?u=%s&quote=%s",
			url.QueryEscape(productURL),
			url.QueryEscape(postText),
		)
	case PlatformInstagram:
		return ""
	case PlatformTwitter:
		tweetText := fmt.Sprintf("%s\n\n%s", postText, productURL)
		return fmt.Sprintf("https://twitter.com/intent/tweet?text=%s",
			url.QueryEscape(tweetText),
		)
	case PlatformPinterest:
		return fmt.Sprintf("https://pinterest.com/pin/create/button/?url=%s&media=%s&description=%s",
			url.QueryEscape(productURL),
			url.QueryEscape(imageURL),
			url.QueryEscape(postText),
		)
	}
	return ""
}

func formatPrice(priceCents int64) string {
	dollars := float64(priceCents) / 100.0
	return fmt.Sprintf("$%.2f", dollars)
}

func getProductEmoji(category string) string {
	categoryLower := strings.ToLower(category)

	switch {
	case strings.Contains(categoryLower, "dinosaur"):
		return "🦖"
	case strings.Contains(categoryLower, "custom"):
		return "✨"
	case strings.Contains(categoryLower, "educational"):
		return "📚"
	case strings.Contains(categoryLower, "event"):
		return "🎉"
	default:
		return "🎨"
	}
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	truncated := text[:maxLength-3]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
