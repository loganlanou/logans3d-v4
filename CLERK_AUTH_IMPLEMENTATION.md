# Clerk Authentication - Server-Side Implementation

## Overview

Complete server-side Clerk authentication system with **ZERO frontend JavaScript authentication logic**. All auth happens in middleware using the latest Clerk Go SDK v2.

## Architecture

### Middleware Chain
```
Request → ClerkAuthMiddleware → Handler
```

**ClerkAuthMiddleware** handles:
1. Extracts `__session` cookie (set by Clerk's hosted pages)
2. Validates JWT token using Clerk Go SDK
3. Fetches full user from Clerk API
4. Syncs user to local database (upsert)
5. Populates Echo context with auth data

### Context Keys

Every handler has access to these via Echo context:
- `clerk_user` - Full Clerk user object (\*clerk.User)
- `clerk_claims` - JWT session claims (\*clerk.SessionClaims)
- `db_user` - Local database user record (\*db.User)
- `is_authenticated` - Boolean authentication status

## Files Created/Modified

### New Files
1. `/storage/migrations/003_clerk_auth_update.sql` - Database schema for Clerk integration
2. `/internal/middleware/clerk.go` - Main auth middleware with database sync
3. `/internal/auth/helpers.go` - Template helper functions for auth
4. `/internal/handlers/auth.go` - Auth route handlers (login/signup/logout)

### Modified Files
1. `/storage/queries/users.sql` - Added Clerk-specific queries (UpsertUserByClerkID, GetUserByClerkID)
2. `/internal/auth/context.go` - Simplified to just type definitions
3. `/public/js/clerk-init.js` - **DELETED** (no frontend auth!)

## Database Schema Changes

### Users Table Updates
```sql
ALTER TABLE users ADD COLUMN clerk_id TEXT UNIQUE;
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name TEXT;
ALTER TABLE users ADD COLUMN username TEXT;
ALTER TABLE users ADD COLUMN profile_image_url TEXT;
ALTER TABLE users ADD COLUMN last_synced_at DATETIME;
```

## Integration Steps

### Step 1: Run Database Migration

```bash
# Navigate to project root
cd /home/loganlanou/projects/loganlanou/logans3d-v4

# Run the migration (using your migration tool)
# If using goose:
goose -dir storage/migrations sqlite3 ./data/database.db up

# Regenerate SQLC code
go generate ./...
```

### Step 2: Update service.go RegisterRoutes

Replace the middleware setup and auth routes:

```go
func (s *Service) RegisterRoutes(e *echo.Echo) {
    // Initialize Clerk SDK
    clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))

    // Apply global middleware - NEW SYSTEM
    e.Use(middleware.ClerkAuthMiddleware(s.storage))  // ← Replace old middleware

    // Static files
    e.Static("/public", "public")

    // Auth routes - NEW HANDLERS
    authHandler := handlers.NewAuthHandler()
    e.GET("/login", authHandler.HandleLogin)
    e.GET("/sign-up", authHandler.HandleSignUp)
    e.GET("/auth/callback", authHandler.HandleAuthCallback)
    e.GET("/account", authHandler.HandleAccount, middleware.RequireClerkAuth())
    e.GET("/logout", authHandler.HandleLogout)  // Changed from POST to GET

    // ... rest of your routes

    // Protected routes use RequireClerkAuth() middleware
    admin := e.Group("/admin", middleware.RequireClerkAuth())
    // ... admin routes
}
```

### Step 3: Remove Old Auth Handlers

Delete or comment out these old handlers in service.go:
- `handleLoginPlaceholder`
- `handleSignUp`
- `handleSSOCallback`
- `handleAccount`
- `handleLogout`

### Step 4: Update Template (base.templ)

Remove Clerk JavaScript and use server-side auth:

```go
// In base.templ <head>, REMOVE these lines:
<script async crossorigin="anonymous" data-clerk-publishable-key={ os.Getenv("CLERK_PUBLISHABLE_KEY") } src="https://cdn.jsdelivr.net/npm/@clerk/clerk-js@latest/dist/clerk.browser.js"></script>
<script src="/public/js/clerk-init.js"></script>

// REMOVE the logout function script block - not needed!
```

Update the logout button to use a simple link:
```go
<a href="/logout" class="block w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-gray-100">Sign Out</a>
```

### Step 5: Environment Variables

Ensure these are set in `.envrc`:
```bash
export CLERK_SECRET_KEY="sk_test_..."
export CLERK_PUBLISHABLE_KEY="pk_test_..."  # Still needed for hosted pages
```

## How It Works

### Sign In Flow

1. User visits `/login`
2. Server redirects to Clerk's hosted sign-in page
3. User authenticates with Clerk
4. Clerk redirects to `/auth/callback` with `__session` cookie set
5. Middleware extracts cookie, validates JWT, syncs user to DB
6. User redirected to intended destination (fully authenticated)

### Sign Out Flow

1. User clicks "Sign Out" link (`/logout`)
2. Server clears `__session` cookie
3. Server redirects to Clerk sign-out URL
4. Clerk clears its session
5. Clerk redirects back to home page

### Accessing User in Templates

```go
// In any handler
authCtx := auth.GetAuthContext(c)
return Render(c, yourTemplate.Index(authCtx))

// Or access directly
if dbUser, ok := auth.GetDBUser(c); ok {
    // Use database user
}

if clerkUser, ok := auth.GetClerkUser(c); ok {
    // Use Clerk user with full profile
}

// Check authentication
if auth.IsAuthenticated(c) {
    // User is logged in
}
```

## Helper Functions

### In Handlers

```go
// Get database user
dbUser, ok := auth.GetDBUser(c)

// Get Clerk user (full profile)
clerkUser, ok := auth.GetClerkUser(c)

// Get JWT claims
claims, ok := auth.GetClerkClaims(c)

// Check if authenticated
isAuth := auth.IsAuthenticated(c)

// Get user ID (tries DB first, falls back to Clerk)
userID, ok := auth.GetUserID(c)

// Require auth in handler
if err := auth.RequireAuth(c); err != nil {
    return err
}
```

### In Templates

```go
// Standard way - get full context
authCtx := auth.GetAuthContext(c)
// authCtx.IsAuthenticated → bool
// authCtx.User.ID → string
// authCtx.User.Email → string
// authCtx.User.FullName → string
// authCtx.User.ImageURL → string
```

## Benefits

✅ **Zero frontend auth logic** - All authentication happens server-side
✅ **Automatic database sync** - User data always up-to-date
✅ **Context-based access** - No need to pass user to every template
✅ **Type-safe** - Full TypeScript-like benefits in Go
✅ **Clerk Go SDK best practices** - Uses official middleware with JWK caching
✅ **Session management** - Token validation on every request
✅ **Security** - HttpOnly cookies, CSRF protection built-in

## Testing

1. Start your dev server: `make dev`
2. Visit `http://localhost:8000/login`
3. Should redirect to Clerk hosted page
4. Sign in with test account
5. Should redirect back and show user menu in header
6. Check logs to see middleware in action

## Troubleshooting

### "No Clerk session token found"
- User isn't logged in
- `__session` cookie not set
- Check Clerk dashboard for session settings

### "Failed to extract Clerk claims"
- Invalid JWT token
- Check `CLERK_SECRET_KEY` is correct
- Token may be expired

### "Failed to sync user to database"
- Database migration not run
- SQLC code not regenerated
- Check database connectivity

### Redirect loops
- Check that `/auth/callback` is publicly accessible
- Ensure middleware doesn't protect auth routes

## Next Steps

1. Run migration and regenerate code
2. Update service.go with new routes
3. Update base.templ to remove JS
4. Test the complete flow
5. Deploy to production with HTTPS (required for secure cookies)
