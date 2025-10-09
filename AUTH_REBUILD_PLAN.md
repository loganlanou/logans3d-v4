# Clerk Authentication Rebuild Plan - Server-Side Sessions
**Date:** 2025-10-08
**Goal:** Implement production-ready authentication with **server-side session management**

---

## Architecture Overview

### The Problem with Current Approach
- ❌ Client-side Clerk manages auth state
- ❌ Server validates JWT on every request (expensive)
- ❌ No server-side session storage
- ❌ Session state not synced between client/server
- ❌ Templates can't access user data server-side

### Correct Architecture - Server-Side Sessions
- ✅ **Server maintains session state**
- ✅ **Session stores user data (name, email, avatar)**
- ✅ **No database/Clerk API calls per request**
- ✅ **Echo context always populated with user**
- ✅ **Templates render based on server context**
- ✅ **NO client-side JavaScript for auth state**

---

## How It Should Work

### 1. Login Flow
```
User → Clerk OAuth → Callback → Backend:
  1. Validate Clerk token
  2. Get user info from Clerk
  3. Create server session
  4. Store user data in session (name, email, avatar, etc.)
  5. Set session cookie
  6. Redirect to homepage
```

### 2. Every Subsequent Request
```
Request → Middleware:
  1. Read session cookie
  2. Load user data from session (in-memory/Redis)
  3. Populate Echo context with user
  4. Handler/template accesses user from context

NO Clerk API calls
NO database queries
NO JWT validation
```

### 3. Logout Flow
```
User clicks logout → Backend:
  1. Clear server session
  2. Delete session cookie
  3. Redirect to Clerk logout URL
  4. Clerk redirects back to homepage
```

### 4. Template Rendering
```go
// In handler
authCtx := auth.GetContextFromSession(c)

// In template
if authCtx.IsAuthenticated {
    // Show user name, avatar
} else {
    // Show "Sign In" button
}
```

**Zero JavaScript** for determining auth state!

---

## Phase 1: Session Management Infrastructure

### Task 1.1: Choose Session Store
**Options:**
- In-memory map (development only)
- Redis (production)
- Database-backed sessions
- Echo sessions middleware

**Decision:** Use **gorilla/sessions** or **Echo session middleware** with in-memory store for dev, Redis for production.

### Task 1.2: Create Session Manager (`internal/session/manager.go`)
```go
type Manager struct {
    store sessions.Store
}

// CreateSession - Called after OAuth callback
func (m *Manager) CreateSession(c echo.Context, user *UserData) error

// GetSession - Called on every request
func (m *Manager) GetSession(c echo.Context) (*UserData, error)

// DestroySession - Called on logout
func (m *Manager) DestroySession(c echo.Context) error
```

### Task 1.3: Define User Session Data Structure
```go
type UserData struct {
    ID        string
    Email     string
    FirstName string
    LastName  string
    FullName  string
    ImageURL  string
    Username  string
}
```

---

## Phase 2: OAuth Callback Handler

### Task 2.1: Create OAuth Callback Route
Route: `GET /auth/callback`

Handler logic:
1. Extract authorization code from query params
2. Exchange code for Clerk session token
3. Validate token with Clerk
4. Get user info from Clerk API
5. Create session with user data
6. Redirect to homepage

### Task 2.2: Handle OAuth Errors
- Invalid code → redirect to login with error
- Clerk API error → show error page
- User cancels → redirect to login

---

## Phase 3: Session Middleware

### Task 3.1: Create Session Middleware (`internal/middleware/session.go`)
```go
func LoadSession(sessionMgr *session.Manager) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Read session cookie
            userData, err := sessionMgr.GetSession(c)

            // Populate Echo context
            if err == nil && userData != nil {
                c.Set("user", userData)
                c.Set("is_authenticated", true)
            } else {
                c.Set("user", nil)
                c.Set("is_authenticated", false)
            }

            return next(c)
        }
    }
}
```

### Task 3.2: Apply Middleware Globally
```go
e.Use(middleware.LoadSession(sessionManager))
```

Now **every handler** has user data in context automatically!

---

## Phase 4: Auth Context for Templates

### Task 4.1: Create Auth Context Helper
```go
func GetAuthContext(c echo.Context) *Context {
    user, _ := c.Get("user").(*UserData)
    isAuth, _ := c.Get("is_authenticated").(bool)

    return &Context{
        IsAuthenticated: isAuth,
        User:            user,
    }
}
```

### Task 4.2: Update All Handlers
```go
func (s *Service) handleHome(c echo.Context) error {
    authCtx := auth.GetAuthContext(c)
    return Render(c, home.Index(authCtx))
}
```

### Task 4.3: Templates Use Server Context
```templ
// base.templ
if authCtx.IsAuthenticated {
    <span>{ authCtx.User.FullName }</span>
} else {
    <a href="/login">Sign In</a>
}
```

**NO JavaScript needed!**

---

## Phase 5: Login & Logout Pages

### Task 5.1: Login Page
Mount Clerk's `SignIn` component:
- Configured with OAuth providers
- `afterSignInUrl` → `/auth/callback`

### Task 5.2: Logout Handler
```go
func (s *Service) handleLogout(c echo.Context) error {
    // Clear server session
    sessionMgr.DestroySession(c)

    // Build Clerk logout URL
    logoutURL := "https://your-instance.clerk.accounts.dev/sign-out?redirect_url=http://localhost:8000"

    return c.Redirect(http.StatusFound, logoutURL)
}
```

Client-side logout button just sends request to `/logout`.

---

## Phase 6: Remove Client-Side Auth Logic

### Task 6.1: Simplify clerk-init.js
**Before:** 100+ lines managing session state
**After:** ~20 lines for sign-out button only

```javascript
// Only handle sign-out button clicks
document.getElementById('sign-out-btn')?.addEventListener('click', () => {
    window.location.href = '/logout';
});
```

### Task 6.2: Remove Session Detection
Delete all client-side session checking code.

### Task 6.3: Remove Session Sync Logic
Delete session listeners, polling, etc.

---

## Phase 7: Protected Routes

### Task 7.1: Create Auth-Required Middleware
```go
func RequireAuth() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            isAuth, _ := c.Get("is_authenticated").(bool)
            if !isAuth {
                return c.Redirect(http.StatusFound, "/login")
            }
            return next(c)
        }
    }
}
```

### Task 7.2: Apply to Protected Routes
```go
e.GET("/account", s.handleAccount, middleware.RequireAuth())
api.Use(middleware.RequireAuth()) // Entire API group
```

---

## Implementation Order

1. ✅ **Session Manager** - Create session storage
2. ✅ **OAuth Callback** - Handle Clerk redirect, create session
3. ✅ **Session Middleware** - Load user into Echo context
4. ✅ **Update Handlers** - Pass auth context to templates
5. ✅ **Update Templates** - Render based on server context
6. ✅ **Logout Handler** - Clear session + redirect to Clerk
7. ✅ **Remove JavaScript** - Delete client-side auth code
8. ✅ **Test** - Verify login/logout/session persistence

---

## Key Benefits

✅ **Performance** - No JWT validation on every request
✅ **Simplicity** - Server owns auth state
✅ **Reliability** - No client/server sync issues
✅ **Security** - Session data server-side only
✅ **Maintainability** - No complex client-side logic

---

## Files to Create

- `internal/session/manager.go` - Session storage
- `internal/session/store.go` - Session data structures
- `internal/middleware/session.go` - Session loading middleware
- `internal/handlers/auth.go` - OAuth callback + logout

## Files to Modify

- `service/service.go` - Add session manager, register routes
- `views/layout/base.templ` - Use server auth context
- `public/js/clerk-init.js` - Simplify to just logout button
- All handlers - Add `auth.GetAuthContext(c)`

## Files to Delete

- None (simplify existing code)

---

## Success Criteria

✅ Login redirects to Clerk, returns to callback
✅ Session stores user data
✅ Username appears in navbar (server-rendered)
✅ Logout clears session
✅ Protected routes redirect to login
✅ **Zero client-side JavaScript** for auth state
✅ Session persists across page reloads
✅ No Clerk API calls on regular page loads

---

**End of Correct Plan**
