# Authentication Rebuild Status
**Last Updated:** 2025-10-08

## ❌ NEEDS COMPLETE REWRITE - Current Approach is Wrong

### What We Built (Incorrect)
The current implementation uses **stateless JWT validation** on every request:
- ✅ Clerk OAuth login/sign-up pages work
- ✅ Clerk middleware validates JWT tokens
- ✅ Client-side Clerk SDK manages session state
- ❌ **NO server-side session storage**
- ❌ **Client/server state sync issues**
- ❌ **Expensive JWT validation on EVERY request**
- ❌ **Templates can't reliably access user data**

### Why This is Wrong
1. **Performance:** JWT validation hits Clerk's JWK endpoint on every page load
2. **Complexity:** Client-side and server-side have different session states
3. **Reliability:** Race conditions between client/server auth states
4. **User Experience:** Navbar shows "Sign In" even when logged in

### What We Need Instead (Server-Side Sessions)
- ✅ **Server stores session data** (name, email, avatar)
- ✅ **OAuth callback creates session**
- ✅ **Middleware loads user from session** (no API calls)
- ✅ **Echo context always populated** with user
- ✅ **Templates render from server context**
- ✅ **ZERO client-side JavaScript** for auth state

---

## Implementation Status

### Phase 1: Session Infrastructure ❌ NOT STARTED
**Goal:** Create server-side session management

#### Task 1.1: Install Session Library
```bash
go get github.com/gorilla/sessions
```

#### Task 1.2: Create Session Manager
**File:** `internal/session/manager.go`
- [ ] Create Manager struct with session store
- [ ] Implement `CreateSession(c echo.Context, user *UserData) error`
- [ ] Implement `GetSession(c echo.Context) (*UserData, error)`
- [ ] Implement `DestroySession(c echo.Context) error`
- [ ] Use in-memory store for development
- [ ] Plan Redis store for production

#### Task 1.3: Define Session Data Types
**File:** `internal/session/types.go`
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

### Phase 2: OAuth Callback Handler ❌ NOT STARTED
**Goal:** Handle Clerk OAuth redirect and create session

#### Task 2.1: Create Callback Route
**Route:** `GET /auth/callback`

**Handler Logic:**
1. Extract authorization code/token from Clerk redirect
2. Validate token with Clerk API
3. Fetch user info from Clerk
4. Create server session with user data
5. Set session cookie
6. Redirect to homepage

**File:** `internal/handlers/auth.go`
```go
func (h *AuthHandler) HandleCallback(c echo.Context) error {
    // Get token from Clerk
    // Validate with Clerk
    // Fetch user info
    // Create session
    // Redirect home
}
```

#### Task 2.2: Error Handling
- [ ] Invalid token → redirect to `/login?error=invalid_token`
- [ ] Clerk API error → show error page
- [ ] User cancels → redirect to `/login`

---

### Phase 3: Session Middleware ❌ NOT STARTED
**Goal:** Load session into Echo context on every request

#### Task 3.1: Create Middleware
**File:** `internal/middleware/session.go`

```go
func LoadSession(sm *session.Manager) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Read session cookie
            userData, err := sm.GetSession(c)

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

#### Task 3.2: Apply Globally
**File:** `service/service.go`
```go
func (s *Service) RegisterRoutes(e *echo.Echo) {
    // Apply session middleware FIRST
    e.Use(middleware.LoadSession(s.sessionManager))

    // Now all handlers have user in context
}
```

---

### Phase 4: Auth Context Helper ❌ NOT STARTED
**Goal:** Provide helper to extract auth context for templates

#### Task 4.1: Update Auth Context
**File:** `internal/auth/context.go`

```go
// Change from JWT validation to session reading
func GetAuthContext(c echo.Context) *Context {
    user, _ := c.Get("user").(*session.UserData)
    isAuth, _ := c.Get("is_authenticated").(bool)

    return &Context{
        IsAuthenticated: isAuth,
        User:            mapSessionUserToAuthUser(user),
    }
}
```

#### Task 4.2: Verify Handlers
**All handlers already use this pattern:**
```go
func (s *Service) handleHome(c echo.Context) error {
    authCtx := auth.GetAuthContext(c)
    return Render(c, home.Index(authCtx))
}
```

**No changes needed!** Just update GetAuthContext() implementation.

---

### Phase 5: Templates ✅ ALREADY CORRECT
**Templates already check server context:**
```templ
if authCtx.IsAuthenticated {
    <span>{ authCtx.User.FullName }</span>
} else {
    <a href="/login">Sign In</a>
}
```

**No changes needed!** Will work automatically once session provides data.

---

### Phase 6: Logout Handler ❌ NOT STARTED
**Goal:** Clear session and redirect to Clerk logout

#### Task 6.1: Create Logout Route
**Route:** `GET /logout`

**Handler:**
```go
func (s *Service) handleLogout(c echo.Context) error {
    // Clear server session
    s.sessionManager.DestroySession(c)

    // Redirect to Clerk logout
    clerkLogoutURL := fmt.Sprintf(
        "https://%s.clerk.accounts.dev/sign-out?redirect_url=%s",
        clerkInstanceName,
        url.QueryEscape("http://localhost:8000"),
    )

    return c.Redirect(http.StatusFound, clerkLogoutURL)
}
```

#### Task 6.2: Update Sign-Out Buttons
**File:** `public/js/clerk-init.js`

Simplify to just:
```javascript
// Sign out buttons
document.getElementById('desktop-sign-out-btn')?.addEventListener('click', () => {
    window.location.href = '/logout';
});

document.getElementById('mobile-sign-out-btn')?.addEventListener('click', () => {
    window.location.href = '/logout';
});
```

**Delete everything else!**

---

### Phase 7: Update Login Page ❌ NOT STARTED
**Goal:** Configure Clerk to redirect to our callback

**File:** `service/service.go` - `handleLoginPlaceholder`

Update Clerk SignIn config:
```javascript
window.Clerk.mountSignIn(document.getElementById('clerk-signin'), {
    signUpUrl: '/sign-up',
    afterSignInUrl: '/auth/callback',  // ← KEY CHANGE
    fallbackRedirectUrl: '/',
});
```

---

### Phase 8: Protected Routes ⚠️ UPDATE NEEDED
**Goal:** Check session instead of JWT

#### Task 8.1: Update RequireAuth Middleware
**File:** `internal/middleware/auth.go`

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

**Delete JWT validation code!**

---

## Files to Create

1. `internal/session/manager.go` - Session storage/retrieval
2. `internal/session/types.go` - UserData struct
3. `internal/middleware/session.go` - Session loading middleware
4. `internal/handlers/auth.go` - OAuth callback + logout

## Files to Modify

1. `service/service.go`:
   - Add `sessionManager *session.Manager` to Service struct
   - Initialize in NewService()
   - Register `/auth/callback` and `/logout` routes
   - Apply session middleware globally

2. `internal/auth/context.go`:
   - Change GetAuthContext() to read from Echo context
   - Remove Clerk API calls

3. `internal/middleware/auth.go`:
   - Simplify RequireAuth() to check session
   - Remove JWT validation

4. `service/service.go` - `handleLoginPlaceholder`:
   - Update `afterSignInUrl` to `/auth/callback`

5. `public/js/clerk-init.js`:
   - Delete session management code
   - Keep only logout button handlers
   - Reduce from 70+ lines to ~15 lines

## Files to Delete

- `internal/middleware/auth.go` (most of it - keep RequireAuth simplified)
- `internal/auth/service.go` (don't need user caching anymore)

---

## Benefits of Correct Approach

### Performance
```
OLD: Request → Clerk JWK fetch → JWT validation → User API call → Render
NEW: Request → Read session (in-memory) → Render
```
**100x faster** - No network calls!

### Simplicity
```
OLD: 70+ lines of client JS + JWT middleware + caching + sync
NEW: Session manager + 15 lines of JS
```

### Reliability
```
OLD: Client state ≠ Server state = Bugs
NEW: Server owns state = No sync issues
```

---

## Estimated Time

- Session Manager: 1 hour
- OAuth Callback: 1 hour
- Middleware Updates: 1 hour
- Logout Handler: 30 minutes
- Testing: 2 hours

**Total: ~5.5 hours**

---

## Success Criteria

✅ Login with Google → Session created
✅ Username appears in navbar (server-rendered)
✅ Refresh page → Still logged in (from session)
✅ No Clerk API calls on page loads
✅ Logout clears session
✅ `/account` redirects to `/login` when not logged in
✅ clerk-init.js is ~15 lines, not 70+
✅ View page source shows username (server-rendered)

---

**Current Status:** Awaiting implementation of correct server-side session approach

**Next Step:** Create session manager infrastructure
