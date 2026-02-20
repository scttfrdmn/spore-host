// Multi-Provider OIDC Authentication for Spawn Dashboard
// Client-Side Architecture: Users query their own AWS accounts directly

// Configuration (credentials from config/google-oauth-credentials.json)
const AUTH_CONFIG = {
    region: 'us-east-1',
    identityPoolId: 'us-east-1:f51165b6-05f7-46c7-a8d3-947080c17876',

    providers: {
        globus: {
            name: 'Globus Auth',
            clientId: '8b578341-b7b5-4e0b-b29f-14a8b3d9011a',
            authEndpoint: 'https://auth.globus.org/v2/oauth2/authorize',
            scope: 'openid profile email',
            responseType: 'token id_token', // Globus requires 'token id_token' (not 'id_token token')
            cognitoKey: 'auth.globus.org'
        },
        google: {
            name: 'Google',
            clientId: '721954523328-d2ark4gifse2j3g763opso2isgqmp4em.apps.googleusercontent.com',
            authEndpoint: 'https://accounts.google.com/o/oauth2/v2/auth',
            scope: 'openid profile email',
            responseType: 'id_token token', // Google supports both tokens
            cognitoKey: 'accounts.google.com'
        },
        github: {
            name: 'GitHub',
            clientId: 'Ov23liOPNcrWFpDvtWrX',
            authEndpoint: 'https://github.com/login/oauth/authorize',
            scope: 'read:user user:email',
            responseType: 'code', // GitHub uses authorization code flow
            useCustomCallback: true, // Redirect to Lambda, not frontend
            customCallbackUrl: 'https://1yr1kjdm5j.execute-api.us-east-1.amazonaws.com/github/callback',
            cognitoKey: 'api.spore.host', // Custom OIDC provider (Lambda bridge)
            disabled: false
        }
    }
};

// Authentication Manager
class AuthManager {
    constructor() {
        this.currentUser = null;
        this.credentials = null;
        this.loadFromStorage();
    }

    // Initialize authentication (check for OAuth callback)
    async init() {
        // Check if we're returning from OAuth callback
        const hash = window.location.hash.substring(1);
        const params = new URLSearchParams(hash);

        console.log('Auth init - hash:', hash);
        console.log('Auth init - has github_auth:', params.has('github_auth'));
        console.log('Auth init - has id_token:', params.has('id_token'));
        console.log('Auth init - has error:', params.has('error'));

        // Check for OAuth errors
        if (params.has('error')) {
            const error = params.get('error');
            const errorDesc = params.get('error_description') || 'Authentication failed';
            console.error('OAuth error:', error, errorDesc);
            throw new Error(`${error}: ${errorDesc}`);
        }

        // Handle GitHub OAuth (custom flow with direct credentials)
        if (params.has('github_auth')) {
            console.log('Handling GitHub callback');
            await this.handleGitHubCallback(params);
            window.history.replaceState({}, document.title, window.location.pathname);
            // Redirect to dashboard if we're on the home page
            if (window.location.pathname === '/' || window.location.pathname === '/index.html') {
                window.location.href = '/dashboard.html';
            }
            return true;
        }

        // Handle standard OIDC callback (Google, Globus)
        if (params.has('id_token')) {
            await this.handleOAuthCallback(params);
            window.history.replaceState({}, document.title, window.location.pathname);
            // Redirect to dashboard if we're on the home page
            if (window.location.pathname === '/' || window.location.pathname === '/index.html') {
                window.location.href = '/dashboard.html';
            }
            return true;
        }

        // Check if we have stored credentials
        if (this.credentials && this.credentials.expiration > Date.now()) {
            await this.configureAWS();
            return true;
        }

        return false;
    }

    // Start OAuth login flow
    login(provider) {
        const config = AUTH_CONFIG.providers[provider];
        if (!config) {
            throw new Error(`Unknown provider: ${provider}`);
        }

        if (config.disabled) {
            alert(`${config.name} is coming soon! It requires additional backend infrastructure (OAuth 2.0 → OIDC token exchange). For now, please use Google or Globus Auth.`);
            return;
        }

        if (!config.clientId) {
            alert(`${config.name} is not configured yet. Please set up OAuth apps first.`);
            return;
        }

        if (!AUTH_CONFIG.identityPoolId) {
            alert('Cognito Identity Pool not configured. Run setup-dashboard-cognito.sh first.');
            return;
        }

        // Build OAuth authorization URL
        // GitHub uses custom callback (Lambda bridge), others use frontend callback
        const redirectUri = config.useCustomCallback
            ? config.customCallbackUrl
            : window.location.origin;
        const state = this.generateState(provider);

        const params = new URLSearchParams({
            client_id: config.clientId,
            redirect_uri: redirectUri,
            response_type: config.responseType || 'id_token token', // Use provider-specific response type
            scope: config.scope,
            state: state,
            nonce: this.generateNonce()
        });

        // Store state for verification
        sessionStorage.setItem('oauth_state', state);
        sessionStorage.setItem('oauth_provider', provider);

        // Redirect to OAuth provider
        window.location.href = `${config.authEndpoint}?${params.toString()}`;
    }

    // Handle OAuth callback
    async handleOAuthCallback(params) {
        const storedState = sessionStorage.getItem('oauth_state');
        const returnedState = params.get('state');

        if (storedState !== returnedState) {
            throw new Error('OAuth state mismatch - possible CSRF attack');
        }

        const provider = sessionStorage.getItem('oauth_provider');
        const config = AUTH_CONFIG.providers[provider];

        // Get ID token
        const idToken = params.get('id_token');
        if (!idToken) {
            throw new Error('No ID token received from OAuth provider');
        }

        // Parse ID token to get user info
        const userInfo = this.parseJWT(idToken);
        this.currentUser = {
            provider: provider,
            id: userInfo.sub,
            email: userInfo.email || userInfo.preferred_username || 'User',
            name: userInfo.name || userInfo.email || 'User'
        };

        // Exchange OIDC token for AWS credentials via Cognito Identity Pool
        await this.exchangeTokenForCredentials(provider, idToken);

        // Save to storage
        this.saveToStorage();

        // Clean up
        sessionStorage.removeItem('oauth_state');
        sessionStorage.removeItem('oauth_provider');

        console.log('Logged in as:', this.currentUser.email);
    }

    // Handle GitHub OAuth callback (direct credentials)
    async handleGitHubCallback(params) {
        try {
            console.log('Processing GitHub OAuth callback...');

            // Decode the base64-encoded auth data
            const authDataB64 = params.get('github_auth');
            console.log('GitHub auth data received:', authDataB64 ? 'yes' : 'no');

            if (!authDataB64) {
                throw new Error('No github_auth parameter in callback');
            }

            const authDataJson = atob(authDataB64.replace(/-/g, '+').replace(/_/g, '/'));
            const authData = JSON.parse(authDataJson);
            console.log('Decoded auth data:', authData);

            // Verify state
            const storedState = sessionStorage.getItem('oauth_state');
            if (storedState !== authData.state) {
                throw new Error('OAuth state mismatch - possible CSRF attack');
            }

            // Set current user
            this.currentUser = {
                provider: 'github',
                id: authData.user.id,
                email: authData.user.email,
                name: authData.user.name,
                picture: authData.user.picture
            };

            // Set credentials (already from STS AssumeRole)
            this.credentials = authData.credentials;

            // Configure AWS SDK
            await this.configureAWS();

            // Save to storage
            this.saveToStorage();

            // Clean up
            sessionStorage.removeItem('oauth_state');
            sessionStorage.removeItem('oauth_provider');

            console.log('Logged in as:', this.currentUser.email, '(via GitHub)');

        } catch (error) {
            console.error('Failed to process GitHub auth:', error);
            throw new Error(`GitHub authentication failed: ${error.message}`);
        }
    }

    // Exchange OIDC token for AWS credentials
    async exchangeTokenForCredentials(provider, idToken) {
        if (!AUTH_CONFIG.identityPoolId) {
            throw new Error('Cognito Identity Pool ID not configured');
        }

        const config = AUTH_CONFIG.providers[provider];

        // Configure AWS SDK region
        AWS.config.region = AUTH_CONFIG.region;

        const cognitoIdentity = new AWS.CognitoIdentity();

        // Get Identity ID
        const logins = {};
        logins[config.cognitoKey] = idToken;

        try {
            const identityData = await cognitoIdentity.getId({
                IdentityPoolId: AUTH_CONFIG.identityPoolId,
                Logins: logins
            }).promise();

            const identityId = identityData.IdentityId;

            // Get temporary credentials
            const credentialsData = await cognitoIdentity.getCredentialsForIdentity({
                IdentityId: identityId,
                Logins: logins
            }).promise();

            this.credentials = {
                accessKeyId: credentialsData.Credentials.AccessKeyId,
                secretAccessKey: credentialsData.Credentials.SecretKey,
                sessionToken: credentialsData.Credentials.SessionToken,
                expiration: credentialsData.Credentials.Expiration.getTime(),
                identityId: identityId
            };

            // Configure AWS SDK with new credentials
            await this.configureAWS();

        } catch (error) {
            console.error('Failed to exchange token for credentials:', error);
            throw new Error(`Authentication failed: ${error.message}`);
        }
    }

    // Configure AWS SDK with credentials
    async configureAWS() {
        if (!this.credentials) {
            throw new Error('No credentials available');
        }

        AWS.config.update({
            region: AUTH_CONFIG.region,
            credentials: new AWS.Credentials({
                accessKeyId: this.credentials.accessKeyId,
                secretAccessKey: this.credentials.secretAccessKey,
                sessionToken: this.credentials.sessionToken
            })
        });

        // Verify credentials work
        try {
            const sts = new AWS.STS();
            const identity = await sts.getCallerIdentity().promise();
            console.log('✓ AWS credentials configured');
            console.log('  User ARN:', identity.Arn);
            console.log('  Account:', identity.Account);
        } catch (error) {
            console.error('Failed to verify AWS credentials:', error);
            throw error;
        }
    }

    // Logout
    logout() {
        // Stop auto-refresh
        if (typeof stopAutoRefresh === 'function') {
            stopAutoRefresh();
        }

        this.currentUser = null;
        this.credentials = null;
        localStorage.removeItem('spawn_auth');
        sessionStorage.clear();
        AWS.config.credentials = null;
        window.location.reload();
    }

    // Check if user is logged in
    isAuthenticated() {
        return this.currentUser !== null &&
               this.credentials !== null &&
               this.credentials.expiration > Date.now();
    }

    // Get current user
    getUser() {
        return this.currentUser;
    }

    // Save to localStorage
    saveToStorage() {
        const data = {
            user: this.currentUser,
            credentials: this.credentials
        };
        localStorage.setItem('spawn_auth', JSON.stringify(data));
    }

    // Load from localStorage
    loadFromStorage() {
        try {
            const data = localStorage.getItem('spawn_auth');
            if (data) {
                const parsed = JSON.parse(data);
                this.currentUser = parsed.user;
                this.credentials = parsed.credentials;

                if (this.credentials && this.credentials.expiration <= Date.now()) {
                    console.log('Stored credentials expired');
                    this.logout();
                }
            }
        } catch (error) {
            console.error('Failed to load stored auth:', error);
            localStorage.removeItem('spawn_auth');
        }
    }

    // Helper: Generate random state
    generateState(provider) {
        return provider + '_' + Math.random().toString(36).substring(2, 15);
    }

    // Helper: Generate nonce
    generateNonce() {
        return Math.random().toString(36).substring(2, 15) +
               Math.random().toString(36).substring(2, 15);
    }

    // Helper: Parse JWT token
    parseJWT(token) {
        try {
            const base64Url = token.split('.')[1];
            const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
            const jsonPayload = decodeURIComponent(atob(base64).split('').map(c => {
                return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
            }).join(''));

            return JSON.parse(jsonPayload);
        } catch (error) {
            console.error('Failed to parse JWT:', error);
            throw new Error('Invalid token format');
        }
    }
}

// Global auth manager instance
const authManager = new AuthManager();

// Initialize on page load
document.addEventListener('DOMContentLoaded', async function() {
    try {
        // Clear any expired credentials before init
        try {
            const data = localStorage.getItem('spawn_auth');
            if (data) {
                const parsed = JSON.parse(data);
                if (parsed.credentials && parsed.credentials.expiration <= Date.now()) {
                    console.log('Clearing expired credentials on page load');
                    localStorage.removeItem('spawn_auth');
                    authManager.currentUser = null;
                    authManager.credentials = null;
                }
            }
        } catch (e) {
            console.error('Error checking stored credentials:', e);
            localStorage.removeItem('spawn_auth');
        }

        const authenticated = await authManager.init();

        if (authenticated) {
            // User is logged in
            showDashboard();
            await loadDashboard(); // From main.js
            updateLastRefreshedTime(); // From main.js
            startAutoRefresh(); // From main.js

            // Initialize WebSocket for real-time updates
            if (typeof initWebSocket === 'function') {
                initWebSocket(); // From websocket.js
            }
        } else {
            // Show login page
            showLoginPage();
        }
    } catch (error) {
        console.error('Auth initialization failed:', error);
        showError('Authentication error: ' + error.message);
        showLoginPage();
    }
});

// UI Functions
function showLoginPage() {
    const dashboardSection = document.getElementById('dashboard');
    if (dashboardSection) {
        dashboardSection.style.display = 'none';
    }

    const loginSection = document.getElementById('login-section');
    if (loginSection) {
        loginSection.style.display = 'block';
    }
}

function showDashboard() {
    const loginSection = document.getElementById('login-section');
    if (loginSection) {
        loginSection.style.display = 'none';
    }

    const dashboardSection = document.getElementById('dashboard');
    if (dashboardSection) {
        dashboardSection.style.display = 'block';
    }

    // Update user info display
    const user = authManager.getUser();
    if (user) {
        const userDisplay = document.getElementById('user-display');
        if (userDisplay) {
            const providerName = user.provider === 'github' ? 'GitHub' :
                                 user.provider === 'google' ? 'Google' :
                                 user.provider === 'globus' ? 'Globus Auth' :
                                 user.provider;
            userDisplay.innerHTML = `
                <span>Logged in as <strong>${escapeHtml(user.email)}</strong> via ${providerName}</span>
                <button onclick="authManager.logout()" class="btn-logout">Logout</button>
            `;
        }
    }

    // Initialize team selector (non-blocking)
    if (typeof initTeamSelector === 'function') {
        initTeamSelector().catch(e => console.warn('Team selector init failed:', e));
    }
}

function showError(message) {
    const errorDiv = document.getElementById('auth-error');
    if (errorDiv) {
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { authManager, AuthManager };
}
