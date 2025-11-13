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
	highlight := getProductHighlight(product.CategoryName, product.Name)
	cta := getCTA(PlatformFacebook, product.CategoryName, product.Name)

	template := fmt.Sprintf(`%s Just finished printing this incredible %s!
💰 %s

%s Printed in high-quality PLA+ with carefully selected colors.

Great for collectors, display shelves, gifts, or anyone who loves %s%s!

🇺🇸 100%% Made in Wisconsin, USA - Never imported, always handcrafted here

%s`,
		emoji,
		product.Name,
		priceText,
		highlight,
		strings.ToLower(product.CategoryName),
		bestSellerText,
		cta,
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

	highlight := getProductHighlight(product.CategoryName, product.Name)
	cta := getCTA(PlatformInstagram, product.CategoryName, product.Name)

	template := fmt.Sprintf(`%s %s%s
💵 %s

%s

🇺🇸 100%% Made in Wisconsin, USA - Never imported!
📦 Ships nationwide

✨ %s

%s`,
		emoji,
		product.Name,
		bestSellerBadge,
		priceText,
		description,
		highlight,
		cta,
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

	cta := getCTA(PlatformTwitter, product.CategoryName, product.Name)

	template := fmt.Sprintf(`%s New: %s%s

Hand-finished 3D print | %s
100%% Made in Wisconsin 🇺🇸

%s`,
		emoji,
		product.Name,
		bestSellerBadge,
		priceText,
		cta,
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

	cta := getCTA(PlatformPinterest, product.CategoryName, product.Name)

	template := fmt.Sprintf(`%s %s - %s

%s

Amazing gift for %s fans and collectors! Hand-finished with incredible detail.

🎁 Great for: Collectors, gifts, home decor, display shelves, STEM enthusiasts
📐 Precision 3D printing with hand-selected colors
🇺🇸 100%% Made in Wisconsin - Never imported, always handcrafted here

%s`,
		emoji,
		product.Name,
		priceText,
		truncateText(description, 150),
		strings.ToLower(product.CategoryName),
		cta,
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
		"#MadeInUSA",
		"#DeskDecor",
		"#GiftIdeas",
		"#UniqueGifts",
		"#HomeDecor",
		"#LogansCreations",
	}

	categoryTag := fmt.Sprintf("#%s", strings.ReplaceAll(product.CategoryName, " ", ""))
	base = append(base, categoryTag)

	if strings.Contains(strings.ToLower(product.CategoryName), "dinosaur") {
		base = append(base, "#DinosaurCollector", "#DinosaurCollectibles", "#DinosaurLover")
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
		// Facebook blocks pre-filling post text via URL parameters (policy restriction)
		// Only Open Graph tags from the product page are used for preview
		return fmt.Sprintf("https://www.facebook.com/sharer/sharer.php?u=%s",
			url.QueryEscape(productURL),
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

func getProductHighlight(category, name string) string {
	categoryLower := strings.ToLower(category)
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "articulated") || strings.Contains(nameLower, "flexi"):
		return "Fully articulated - pose it however you want!"
	case strings.Contains(categoryLower, "dragon"):
		return "Perfect desk companion with incredible detail"
	case strings.Contains(categoryLower, "dinosaur"):
		return "Stunning detail that collectors love"
	case strings.Contains(categoryLower, "fidget") || strings.Contains(nameLower, "fidget"):
		return "Satisfying to hold and fun to display"
	default:
		return "Hand-finished with amazing detail"
	}
}

func getCTA(platform Platform, category, name string) string {
	categoryLower := strings.ToLower(category)
	nameLower := strings.ToLower(name)

	// Category-specific CTAs
	hasArticulated := strings.Contains(nameLower, "articulated") || strings.Contains(nameLower, "flexi")
	isDinosaur := strings.Contains(categoryLower, "dinosaur")
	isCustom := strings.Contains(categoryLower, "custom")

	switch platform {
	case PlatformFacebook:
		if hasArticulated {
			return "Want to pose your own? Tap to choose your colors and customize!"
		}
		if isDinosaur {
			return "Ready to start your collection? Choose your colors now!"
		}
		if isCustom {
			return "Pick your favorite colors and make it yours!"
		}
		return "Choose your colors and customize yours today!"

	case PlatformInstagram:
		if hasArticulated {
			return "Pose it your way! Link in bio to customize 👆"
		}
		if isDinosaur {
			return "Start your collection! Link in bio to choose colors 👆"
		}
		return "Customize yours! Link in bio to pick your colors 👆"

	case PlatformTwitter:
		return "Choose your colors →"

	case PlatformPinterest:
		if hasArticulated {
			return "Click to pose your own!"
		}
		if isDinosaur {
			return "Click to start your collection!"
		}
		return "Click to customize with your favorite colors!"
	}

	return "Shop now!"
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
