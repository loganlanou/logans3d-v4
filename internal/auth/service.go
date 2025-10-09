package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/labstack/echo/v4"
)

// Service handles authentication-related operations
type Service struct {
	cache *userCache
}

// NewService creates a new auth service
func NewService() *Service {
	return &Service{
		cache: newUserCache(5 * time.Minute), // 5 minute cache
	}
}

// GetUser fetches a user by ID from Clerk API (with caching)
func (s *Service) GetUser(ctx context.Context, userID string) (*clerk.User, error) {
	// Check cache first
	if cachedUser := s.cache.Get(userID); cachedUser != nil {
		return cachedUser, nil
	}

	// Fetch from Clerk API using the global clerk client
	usr, err := user.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Cache the result
	s.cache.Set(userID, usr)

	return usr, nil
}

// GetCurrentUser gets the authenticated user from the Echo context
func (s *Service) GetCurrentUser(c echo.Context) (*clerk.User, error) {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	return s.GetUser(c.Request().Context(), userID)
}

// GetUserID gets the current user ID from the Echo context
func (s *Service) GetUserID(c echo.Context) (string, bool) {
	userID, ok := c.Get("user_id").(string)
	return userID, ok && userID != ""
}

// IsAuthenticated checks if the current request is authenticated
func (s *Service) IsAuthenticated(c echo.Context) bool {
	_, ok := s.GetUserID(c)
	return ok
}

// InvalidateCache removes a user from the cache (useful after profile updates)
func (s *Service) InvalidateCache(userID string) {
	s.cache.Delete(userID)
}

// userCache is a simple in-memory cache for user data
type userCache struct {
	mu      sync.RWMutex
	data    map[string]*cacheEntry
	ttl     time.Duration
	cleanup *time.Ticker
	done    chan bool
}

type cacheEntry struct {
	user      *clerk.User
	expiresAt time.Time
}

func newUserCache(ttl time.Duration) *userCache {
	cache := &userCache{
		data:    make(map[string]*cacheEntry),
		ttl:     ttl,
		cleanup: time.NewTicker(ttl),
		done:    make(chan bool),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

func (c *userCache) Get(userID string) *clerk.User {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[userID]
	if !exists {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.user
}

func (c *userCache) Set(userID string, user *clerk.User) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[userID] = &cacheEntry{
		user:      user,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *userCache) Delete(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, userID)
}

func (c *userCache) cleanupExpired() {
	for {
		select {
		case <-c.cleanup.C:
			c.mu.Lock()
			now := time.Now()
			for id, entry := range c.data {
				if now.After(entry.expiresAt) {
					delete(c.data, id)
				}
			}
			c.mu.Unlock()
		case <-c.done:
			return
		}
	}
}

func (c *userCache) Stop() {
	c.cleanup.Stop()
	c.done <- true
}
