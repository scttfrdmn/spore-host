// WebSocket Manager for Real-Time Dashboard Updates
// Handles connection lifecycle, reconnection, and graceful fallback to polling

class DashboardWebSocket {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.baseReconnectDelay = 1000; // 1 second
        this.maxReconnectDelay = 30000; // 30 seconds
        this.reconnectTimeout = null;
        this.fallbackToPolling = false;
        this.isConnected = false;
        this.isConnecting = false;
        this.wsEndpoint = null;
        this.credentialsRefreshInterval = null;
    }

    // Initialize WebSocket connection
    async connect() {
        // Don't connect if already connected or connecting
        if (this.isConnected || this.isConnecting) {
            console.log('WebSocket: Already connected or connecting');
            return;
        }

        // Check if credentials are available
        if (!AWS.config.credentials) {
            console.error('WebSocket: AWS credentials not configured');
            this.enablePollingFallback();
            return;
        }

        try {
            this.isConnecting = true;
            this.updateConnectionStatus('connecting');

            // Get credentials
            const credentials = AWS.config.credentials;
            if (!credentials.accessKeyId || !credentials.secretAccessKey) {
                throw new Error('Invalid AWS credentials');
            }

            // Construct auth token
            const token = btoa(JSON.stringify({
                accessKeyId: credentials.accessKeyId,
                secretAccessKey: credentials.secretAccessKey,
                sessionToken: credentials.sessionToken || ''
            }));

            // WebSocket API Gateway endpoint
            this.wsEndpoint = 'wss://ir832sgfz2.execute-api.us-east-1.amazonaws.com/production';

            // Connect to WebSocket
            const url = `${this.wsEndpoint}?token=${encodeURIComponent(token)}`;
            console.log('WebSocket: Connecting to', this.wsEndpoint);

            this.ws = new WebSocket(url);

            // Connection opened
            this.ws.onopen = () => {
                console.log('WebSocket: Connected');
                this.isConnected = true;
                this.isConnecting = false;
                this.reconnectAttempts = 0;
                this.fallbackToPolling = false;
                this.updateConnectionStatus('connected');

                // Stop polling when WebSocket is connected
                this.disablePolling();
            };

            // Message received
            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('WebSocket: Failed to parse message:', error);
                }
            };

            // Connection closed
            this.ws.onclose = (event) => {
                console.log('WebSocket: Connection closed', event.code, event.reason);
                this.isConnected = false;
                this.isConnecting = false;
                this.ws = null;

                // Attempt to reconnect
                if (!this.fallbackToPolling) {
                    this.reconnect();
                }
            };

            // Connection error
            this.ws.onerror = (error) => {
                console.error('WebSocket: Connection error:', error);
                this.isConnected = false;
                this.isConnecting = false;
                this.updateConnectionStatus('error');
            };

            // Set up credentials refresh (55 minutes - before 1 hour expiry)
            this.setupCredentialsRefresh();

        } catch (error) {
            console.error('WebSocket: Failed to connect:', error);
            this.isConnecting = false;
            this.updateConnectionStatus('error');
            this.reconnect();
        }
    }

    // Handle incoming messages
    handleMessage(message) {
        console.log('WebSocket: Received message:', message);

        const { type, data } = message;

        switch (type) {
            case 'sweep:update':
                this.handleSweepUpdate(data);
                break;

            case 'sweep:delete':
                this.handleSweepDelete(data);
                break;

            case 'autoscale:update':
                this.handleAutoscaleUpdate(data);
                break;

            case 'autoscale:delete':
                this.handleAutoscaleDelete(data);
                break;

            default:
                console.warn('WebSocket: Unknown message type:', type);
        }
    }

    // Handle sweep update
    handleSweepUpdate(sweep) {
        if (typeof allSweepsCache === 'undefined') {
            console.warn('WebSocket: allSweepsCache not defined yet');
            return;
        }

        // Find and update existing sweep, or add new one
        const index = allSweepsCache.findIndex(s => s.sweep_id === sweep.sweep_id);
        if (index >= 0) {
            allSweepsCache[index] = sweep;
            console.log('WebSocket: Updated sweep:', sweep.sweep_id);
        } else {
            allSweepsCache.push(sweep);
            console.log('WebSocket: Added new sweep:', sweep.sweep_id);
        }

        // Refresh display if on sweeps tab
        if (typeof getCurrentDashboardTab === 'function' && getCurrentDashboardTab() === 'sweeps') {
            if (typeof applySweepFilters === 'function') {
                applySweepFilters();
            }
        }

        // Update last refreshed time
        if (typeof updateLastRefreshedTime === 'function') {
            updateLastRefreshedTime();
        }
    }

    // Handle sweep deletion
    handleSweepDelete(data) {
        if (typeof allSweepsCache === 'undefined') {
            console.warn('WebSocket: allSweepsCache not defined yet');
            return;
        }

        const sweepId = data.sweep_id;
        const index = allSweepsCache.findIndex(s => s.sweep_id === sweepId);
        if (index >= 0) {
            allSweepsCache.splice(index, 1);
            console.log('WebSocket: Deleted sweep:', sweepId);

            // Refresh display if on sweeps tab
            if (typeof getCurrentDashboardTab === 'function' && getCurrentDashboardTab() === 'sweeps') {
                if (typeof applySweepFilters === 'function') {
                    applySweepFilters();
                }
            }
        }
    }

    // Handle autoscale group update
    handleAutoscaleUpdate(group) {
        if (typeof allAutoscaleGroupsCache === 'undefined') {
            console.warn('WebSocket: allAutoscaleGroupsCache not defined yet');
            return;
        }

        // Find and update existing group, or add new one
        const index = allAutoscaleGroupsCache.findIndex(g => g.autoscale_group_id === group.autoscale_group_id);
        if (index >= 0) {
            allAutoscaleGroupsCache[index] = group;
            console.log('WebSocket: Updated autoscale group:', group.autoscale_group_id);
        } else {
            allAutoscaleGroupsCache.push(group);
            console.log('WebSocket: Added new autoscale group:', group.autoscale_group_id);
        }

        // Refresh display if on autoscale tab
        if (typeof getCurrentDashboardTab === 'function' && getCurrentDashboardTab() === 'autoscale') {
            if (typeof applyAutoscaleFilters === 'function') {
                applyAutoscaleFilters();
            }
        }

        // Update last refreshed time
        if (typeof updateLastRefreshedTime === 'function') {
            updateLastRefreshedTime();
        }
    }

    // Handle autoscale group deletion
    handleAutoscaleDelete(data) {
        if (typeof allAutoscaleGroupsCache === 'undefined') {
            console.warn('WebSocket: allAutoscaleGroupsCache not defined yet');
            return;
        }

        const groupId = data.autoscale_group_id;
        const index = allAutoscaleGroupsCache.findIndex(g => g.autoscale_group_id === groupId);
        if (index >= 0) {
            allAutoscaleGroupsCache.splice(index, 1);
            console.log('WebSocket: Deleted autoscale group:', groupId);

            // Refresh display if on autoscale tab
            if (typeof getCurrentDashboardTab === 'function' && getCurrentDashboardTab() === 'autoscale') {
                if (typeof applyAutoscaleFilters === 'function') {
                    applyAutoscaleFilters();
                }
            }
        }
    }

    // Reconnect with exponential backoff
    reconnect() {
        // Clear any existing reconnect timeout
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }

        // Check if max attempts reached
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('WebSocket: Max reconnection attempts reached, falling back to polling');
            this.enablePollingFallback();
            return;
        }

        // Calculate delay with exponential backoff
        const delay = Math.min(
            this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts),
            this.maxReconnectDelay
        );

        this.reconnectAttempts++;
        console.log(`WebSocket: Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

        this.updateConnectionStatus('reconnecting');

        this.reconnectTimeout = setTimeout(() => {
            this.connect();
        }, delay);
    }

    // Enable polling fallback
    enablePollingFallback() {
        console.log('WebSocket: Enabling polling fallback');
        this.fallbackToPolling = true;
        this.updateConnectionStatus('polling');

        // Re-enable polling intervals
        if (typeof startAutoRefresh === 'function') {
            startAutoRefresh();
        }
    }

    // Disable polling (when WebSocket is active)
    disablePolling() {
        console.log('WebSocket: Disabling polling (WebSocket active)');

        // Stop auto-refresh for sweeps and autoscale (keep instances polling)
        if (typeof window.dashboardRefreshInterval !== 'undefined' && window.dashboardRefreshInterval) {
            // Don't clear the interval completely, but we'll skip sweep/autoscale refreshes
            // This is handled in the refresh function by checking WebSocket connection status
        }
    }

    // Setup credentials refresh
    setupCredentialsRefresh() {
        // Clear any existing interval
        if (this.credentialsRefreshInterval) {
            clearInterval(this.credentialsRefreshInterval);
        }

        // Refresh every 55 minutes (before 1-hour credential expiry)
        this.credentialsRefreshInterval = setInterval(() => {
            console.log('WebSocket: Refreshing credentials and reconnecting...');
            this.disconnect();
            // Wait a bit for disconnect to complete, then reconnect with new credentials
            setTimeout(() => {
                this.connect();
            }, 1000);
        }, 55 * 60 * 1000);
    }

    // Disconnect
    disconnect() {
        console.log('WebSocket: Disconnecting');

        // Clear reconnect timeout
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }

        // Clear credentials refresh interval
        if (this.credentialsRefreshInterval) {
            clearInterval(this.credentialsRefreshInterval);
            this.credentialsRefreshInterval = null;
        }

        // Close WebSocket
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }

        this.isConnected = false;
        this.isConnecting = false;
        this.updateConnectionStatus('disconnected');
    }

    // Update connection status indicator
    updateConnectionStatus(status) {
        const indicator = document.getElementById('connection-status');
        if (!indicator) return;

        // Remove all status classes
        indicator.classList.remove('status-connected', 'status-polling', 'status-disconnected', 'status-error', 'status-connecting', 'status-reconnecting');

        // Add appropriate class and title
        switch (status) {
            case 'connected':
                indicator.classList.add('status-connected');
                indicator.title = 'Real-time updates active';
                break;
            case 'polling':
                indicator.classList.add('status-polling');
                indicator.title = 'Polling mode (WebSocket unavailable)';
                break;
            case 'disconnected':
                indicator.classList.add('status-disconnected');
                indicator.title = 'Disconnected';
                break;
            case 'error':
                indicator.classList.add('status-error');
                indicator.title = 'Connection error';
                break;
            case 'connecting':
                indicator.classList.add('status-connecting');
                indicator.title = 'Connecting...';
                break;
            case 'reconnecting':
                indicator.classList.add('status-reconnecting');
                indicator.title = 'Reconnecting...';
                break;
        }
    }

    // Get current connection state
    getState() {
        return {
            isConnected: this.isConnected,
            isConnecting: this.isConnecting,
            fallbackToPolling: this.fallbackToPolling,
            reconnectAttempts: this.reconnectAttempts
        };
    }
}

// Global WebSocket instance
let dashboardWebSocket = null;

// Initialize WebSocket connection (called after authentication)
function initWebSocket() {
    console.log('Initializing WebSocket connection...');

    // Create WebSocket instance if it doesn't exist
    if (!dashboardWebSocket) {
        dashboardWebSocket = new DashboardWebSocket();
    }

    // Connect
    dashboardWebSocket.connect();
}

// Get current dashboard tab (helper for WebSocket)
function getCurrentDashboardTab() {
    if (typeof currentDashboardTab !== 'undefined') {
        return currentDashboardTab;
    }
    return null;
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { DashboardWebSocket, initWebSocket };
}
