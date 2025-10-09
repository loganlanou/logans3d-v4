package middleware

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/labstack/echo/v4"
	"github.com/loganlanou/logans3d-v4/internal/session"
)

// LoadSession is middleware that loads user session into Echo context
// If Clerk session exists but server session doesn't, automatically creates server session
func LoadSession(sessionMgr *session.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Try to get user data from server session
			userData, err := sessionMgr.GetSession(c)

			if err == nil && userData != nil {
				// Server session exists, use it
				fmt.Printf("[SESSION] %s - Server session found for user: %s (%s)\n", path, userData.Email, userData.ID)
				c.Set("user", userData)
				c.Set("is_authenticated", true)
			} else {
				fmt.Printf("[SESSION] %s - No server session (err: %v)\n", path, err)

				// No server session, check if Clerk session exists
				claims, clerkOk := GetSessionClaims(c)
				fmt.Printf("[SESSION] %s - Clerk claims check: ok=%v, claims=%v\n", path, clerkOk, claims != nil)

				if clerkOk && claims != nil && claims.Subject != "" {
					fmt.Printf("[SESSION] %s - Clerk session found for user ID: %s\n", path, claims.Subject)

					// Clerk session exists, create server session
					clerkUser, fetchErr := user.Get(context.Background(), claims.Subject)
					if fetchErr == nil && clerkUser != nil {
						fmt.Printf("[SESSION] %s - Fetched Clerk user: %s\n", path, getFirstEmail(clerkUser))

						// Create session data
						newUserData := &session.UserData{
							ID:        clerkUser.ID,
							Email:     getFirstEmail(clerkUser),
							FirstName: getStringValue(clerkUser.FirstName),
							LastName:  getStringValue(clerkUser.LastName),
							FullName:  getFullName(clerkUser),
							ImageURL:  getStringValue(clerkUser.ImageURL),
							Username:  getStringValue(clerkUser.Username),
							HasImage:  clerkUser.ImageURL != nil && *clerkUser.ImageURL != "",
						}

						// Save server session
						if createErr := sessionMgr.CreateSession(c, newUserData); createErr == nil {
							fmt.Printf("[SESSION] %s - Created server session for: %s\n", path, newUserData.Email)
							c.Set("user", newUserData)
							c.Set("is_authenticated", true)
						} else {
							fmt.Printf("[SESSION] %s - Failed to create server session: %v\n", path, createErr)
							c.Set("user", nil)
							c.Set("is_authenticated", false)
						}
					} else {
						fmt.Printf("[SESSION] %s - Failed to fetch Clerk user: %v\n", path, fetchErr)
						c.Set("user", nil)
						c.Set("is_authenticated", false)
					}
				} else {
					fmt.Printf("[SESSION] %s - No Clerk session\n", path)
					// No Clerk session either
					c.Set("user", nil)
					c.Set("is_authenticated", false)
				}
			}

			return next(c)
		}
	}
}

// Helper functions
func getFirstEmail(clerkUser *clerk.User) string {
	if len(clerkUser.EmailAddresses) == 0 {
		return ""
	}
	return clerkUser.EmailAddresses[0].EmailAddress
}

func getStringValue(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}

func getFullName(clerkUser *clerk.User) string {
	firstName := getStringValue(clerkUser.FirstName)
	lastName := getStringValue(clerkUser.LastName)

	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	if username := getStringValue(clerkUser.Username); username != "" {
		return username
	}
	if email := getFirstEmail(clerkUser); email != "" {
		return email
	}
	return "User"
}
