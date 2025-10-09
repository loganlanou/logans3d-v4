/**
 * Global Clerk initialization
 * This script loads on every page to:
 * 1. Process OAuth handshakes (__clerk_handshake parameter)
 * 2. Manage the __session cookie
 * 3. Keep session state synchronized between frontend and backend
 */

window.addEventListener('load', async () => {
    // Wait for Clerk to be available
    if (!window.Clerk) {
        console.warn('[CLERK] Clerk.js not loaded');
        return;
    }

    try {
        // Initialize Clerk
        await window.Clerk.load();
        console.log('[CLERK] Frontend initialized successfully');

        // Check if user is signed in
        if (window.Clerk.user) {
            console.log('[CLERK] User session detected:', window.Clerk.user.id);

            // Get the session token
            const session = window.Clerk.session;
            if (session) {
                console.log('[CLERK] Session active, token should be in __session cookie');
            }
        } else {
            console.log('[CLERK] No active session');
        }

        // Listen for session changes
        window.Clerk.addListener((resources) => {
            if (resources.session) {
                console.log('[CLERK] Session changed, reloading page to sync with backend');
                // Reload the page to let the backend middleware pick up the new session
                window.location.reload();
            }
        });

    } catch (error) {
        console.error('[CLERK] Initialization error:', error);
    }
});
