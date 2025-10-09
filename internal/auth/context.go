package auth

import (
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/session"
)

// Context holds authentication data to be passed to templates
type Context struct {
	IsAuthenticated bool
	User            *UserData
}

// UserData contains user information for templates
type UserData struct {
	ID            string
	Email         string
	FirstName     string
	LastName      string
	FullName      string
	ImageURL      string
	Username      string
	HasImage      bool
}

// GetAuthContext gets auth context from the session (loaded by middleware)
func GetAuthContext(c echo.Context) *Context {
	// Get authentication status from Echo context (set by session middleware)
	isAuth, _ := c.Get("is_authenticated").(bool)

	if !isAuth {
		return &Context{
			IsAuthenticated: false,
			User:            nil,
		}
	}

	// Get user data from Echo context (set by session middleware)
	sessionUser, ok := c.Get("user").(*session.UserData)
	if !ok || sessionUser == nil {
		return &Context{
			IsAuthenticated: false,
			User:            nil,
		}
	}

	return &Context{
		IsAuthenticated: true,
		User: &UserData{
			ID:        sessionUser.ID,
			Email:     sessionUser.Email,
			FirstName: sessionUser.FirstName,
			LastName:  sessionUser.LastName,
			FullName:  sessionUser.FullName,
			ImageURL:  sessionUser.ImageURL,
			Username:  sessionUser.Username,
			HasImage:  sessionUser.HasImage,
		},
	}
}

// NewContext creates an auth context from the current request (DEPRECATED: Use GetAuthContext instead)
func NewContext(c echo.Context, authService *Service) *Context {
	// For backward compatibility, delegate to GetAuthContext
	return GetAuthContext(c)
}

// mapUserToUserData converts Clerk user to template-friendly UserData
func mapUserToUserData(user *clerk.User) *UserData {
	if user == nil {
		return nil
	}

	userData := &UserData{
		ID:        user.ID,
		FirstName: stringValue(user.FirstName),
		LastName:  stringValue(user.LastName),
		Username:  stringValue(user.Username),
		ImageURL:  stringValue(user.ImageURL),
		HasImage:  stringValue(user.ImageURL) != "",
	}

	// Get primary email
	if len(user.EmailAddresses) > 0 {
		for _, email := range user.EmailAddresses {
			if email.ID == stringValue(user.PrimaryEmailAddressID) {
				userData.Email = email.EmailAddress
				break
			}
		}
		// Fallback to first email if primary not found
		if userData.Email == "" {
			userData.Email = user.EmailAddresses[0].EmailAddress
		}
	}

	// Build full name
	if userData.FirstName != "" && userData.LastName != "" {
		userData.FullName = userData.FirstName + " " + userData.LastName
	} else if userData.FirstName != "" {
		userData.FullName = userData.FirstName
	} else if userData.LastName != "" {
		userData.FullName = userData.LastName
	} else if userData.Username != "" {
		userData.FullName = userData.Username
	} else {
		userData.FullName = "User"
	}

	return userData
}

// stringValue safely converts a *string to string
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
