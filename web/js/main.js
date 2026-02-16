// Mycelium Landing Page - Interactive Features

// Tab Switching for Install Instructions
function showTab(tabName, event) {
    const contents = document.querySelectorAll('.tab-content');
    contents.forEach(content => {
        content.classList.remove('active');
    });

    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(button => {
        button.classList.remove('active');
    });

    const selectedTab = document.getElementById(tabName);
    if (selectedTab) {
        selectedTab.classList.add('active');
    }

    const clickedButton = event?.target;
    if (clickedButton && clickedButton.classList) {
        clickedButton.classList.add('active');
    }
}

// Smooth Scrolling for Anchor Links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// Add Loading Animation to External Links
document.querySelectorAll('a[target="_blank"]').forEach(link => {
    link.addEventListener('click', function() {
        this.style.opacity = '0.7';
        setTimeout(() => {
            this.style.opacity = '1';
        }, 300);
    });
});

// Detect OS and Set Default Tab
function setDefaultInstallTab() {
    const userAgent = navigator.userAgent.toLowerCase();
    let defaultTab = 'homebrew';

    if (userAgent.includes('win')) {
        defaultTab = 'scoop';
    }

    showTab(defaultTab);

    const buttons = document.querySelectorAll('.tab-btn');
    buttons.forEach(button => {
        button.classList.remove('active');
    });

    const activeButton = Array.from(buttons).find(btn =>
        (defaultTab === 'scoop' && btn.textContent.includes('Windows')) ||
        (defaultTab === 'homebrew' && btn.textContent.includes('macOS'))
    );

    if (activeButton) {
        activeButton.classList.add('active');
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    setDefaultInstallTab();

    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '0';
                entry.target.style.transform = 'translateY(20px)';
                entry.target.style.transition = 'all 0.6s ease';

                setTimeout(() => {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                }, 100);

                observer.unobserve(entry.target);
            }
        });
    }, observerOptions);

    document.querySelectorAll('.feature-card, .example, .preview-card').forEach(el => {
        observer.observe(el);
    });
});

// Copy to Clipboard for Code Blocks
function addCopyButtons() {
    const codeBlocks = document.querySelectorAll('pre code');
    codeBlocks.forEach((block, index) => {
        const button = document.createElement('button');
        button.textContent = 'Copy';
        button.className = 'copy-btn';
        button.style.cssText = `
            position: absolute;
            top: 0.5rem;
            right: 0.5rem;
            padding: 0.3rem 0.8rem;
            background: var(--accent-blue);
            color: var(--bg-dark);
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.85rem;
            opacity: 0;
            transition: opacity 0.3s ease;
        `;

        const pre = block.parentElement;
        pre.style.position = 'relative';
        pre.appendChild(button);

        pre.addEventListener('mouseenter', () => {
            button.style.opacity = '1';
        });

        pre.addEventListener('mouseleave', () => {
            button.style.opacity = '0';
        });

        button.addEventListener('click', () => {
            navigator.clipboard.writeText(block.textContent).then(() => {
                button.textContent = 'Copied!';
                setTimeout(() => {
                    button.textContent = 'Copy';
                }, 2000);
            });
        });
    });
}

if (navigator.clipboard) {
    document.addEventListener('DOMContentLoaded', addCopyButtons);
}

// ═══════════════════════════════════════════════════════════════
// Dashboard - Client-Side EC2 Queries
// ═══════════════════════════════════════════════════════════════

// AWS regions to query
const AWS_REGIONS = [
    'us-east-1', 'us-east-2', 'us-west-1', 'us-west-2',
    'eu-west-1', 'eu-west-2', 'eu-central-1',
    'ap-southeast-1', 'ap-southeast-2', 'ap-northeast-1'
];

// Dashboard API - Client-Side EC2 queries using user's AWS credentials
const DashboardAPI = {
    // Cross-account role ARN (development account where instances live)
    crossAccountRoleArn: 'arn:aws:iam::435415984226:role/SpawnDashboardCrossAccountReadRole',
    crossAccountCredentials: null,

    // Assume cross-account role to access EC2 instances in development account
    async assumeCrossAccountRole() {
        if (this.crossAccountCredentials && this.crossAccountCredentials.expiration > Date.now()) {
            return this.crossAccountCredentials;
        }

        const sts = new AWS.STS();
        const data = await sts.assumeRole({
            RoleArn: this.crossAccountRoleArn,
            RoleSessionName: 'spawn-dashboard-session',
            DurationSeconds: 3600
        }).promise();

        this.crossAccountCredentials = {
            accessKeyId: data.Credentials.AccessKeyId,
            secretAccessKey: data.Credentials.SecretAccessKey,
            sessionToken: data.Credentials.SessionToken,
            expiration: data.Credentials.Expiration.getTime()
        };

        return this.crossAccountCredentials;
    },

    // List instances across all regions (parallel queries)
    async listInstances() {
        if (!AWS.config.credentials) {
            throw new Error('AWS credentials not configured');
        }

        // Check if user authenticated via GitHub (credentials already have cross-account access)
        const user = authManager.getUser();
        const isGitHubAuth = user && user.provider === 'github';

        // Only assume cross-account role for non-GitHub auth (Google, Globus)
        // GitHub Lambda already provides cross-account credentials
        if (!isGitHubAuth) {
            await this.assumeCrossAccountRole();
        }

        const results = await Promise.allSettled(
            AWS_REGIONS.map(region => this.listInstancesInRegion(region))
        );

        // Combine all successful results
        const allInstances = [];
        results.forEach(result => {
            if (result.status === 'fulfilled' && result.value) {
                allInstances.push(...result.value);
            }
        });

        // Sort by launch time (newest first)
        allInstances.sort((a, b) => new Date(b.launch_time) - new Date(a.launch_time));

        return {
            success: true,
            regions_queried: AWS_REGIONS,
            total_instances: allInstances.length,
            instances: allInstances
        };
    },

    // List instances in a specific region
    async listInstancesInRegion(region) {
        // Check if user authenticated via GitHub
        const user = authManager.getUser();
        const isGitHubAuth = user && user.provider === 'github';

        // GitHub: use existing credentials (already cross-account)
        // Others: use cross-account credentials from STS AssumeRole
        const credentials = isGitHubAuth
            ? AWS.config.credentials
            : new AWS.Credentials({
                accessKeyId: this.crossAccountCredentials.accessKeyId,
                secretAccessKey: this.crossAccountCredentials.secretAccessKey,
                sessionToken: this.crossAccountCredentials.sessionToken
            });

        const ec2 = new AWS.EC2({
            region: region,
            credentials: credentials
        });

        // Query EC2 with filters (only show instances launched via spawn CLI)
        const params = {
            Filters: [
                { Name: 'tag:spawn:created-by', Values: ['spawn'] }
            ]
        };

        const data = await ec2.describeInstances(params).promise();

        // Convert to instance list
        const instances = [];
        data.Reservations.forEach(reservation => {
            reservation.Instances.forEach(instance => {
                instances.push(this.convertInstance(instance, region));
            });
        });

        return instances;
    },

    // Convert EC2 instance to dashboard format
    convertInstance(instance, region) {
        const tags = {};
        (instance.Tags || []).forEach(tag => {
            tags[tag.Key] = tag.Value;
        });

        // Parse state transition reason to get termination time
        let terminationTime = null;
        if (instance.State.Name === 'terminated' && instance.StateTransitionReason) {
            // Format: "User initiated (2026-01-15 01:30:45 GMT)"
            const match = instance.StateTransitionReason.match(/\(([^)]+)\)/);
            if (match) {
                try {
                    terminationTime = new Date(match[1]);
                } catch (e) {
                    // Ignore parsing errors
                }
            }
        }

        return {
            instance_id: instance.InstanceId,
            name: tags['Name'] || instance.InstanceId,
            instance_type: instance.InstanceType,
            state: instance.State.Name,
            region: region,
            availability_zone: instance.Placement.AvailabilityZone,
            public_ip: instance.PublicIpAddress || null,
            private_ip: instance.PrivateIpAddress || null,
            launch_time: instance.LaunchTime,
            termination_time: terminationTime,
            ttl: tags['spawn:ttl'] || null,
            dns_name: tags['spawn:dns-name'] || null,
            spot_instance: instance.InstanceLifecycle === 'spot',
            key_name: instance.KeyName || null,
            tags: tags
        };
    },

    // Get user account info
    async getUserProfile() {
        if (!AWS.config.credentials) {
            throw new Error('AWS credentials not configured');
        }

        const sts = new AWS.STS();
        const identity = await sts.getCallerIdentity().promise();

        // Also get cross-account identity
        let devAccountIdentity = null;
        try {
            await this.assumeCrossAccountRole();
            const devSts = new AWS.STS({
                credentials: new AWS.Credentials({
                    accessKeyId: this.crossAccountCredentials.accessKeyId,
                    secretAccessKey: this.crossAccountCredentials.secretAccessKey,
                    sessionToken: this.crossAccountCredentials.sessionToken
                })
            });
            devAccountIdentity = await devSts.getCallerIdentity().promise();
        } catch (error) {
            console.warn('Could not get dev account identity:', error);
        }

        return {
            success: true,
            user: {
                user_id: identity.Arn,
                aws_account_id: identity.Account,
                user_arn: identity.Arn,
                dev_account_id: devAccountIdentity?.Account || null
            }
        };
    }
};

// Dashboard UI Functions
async function loadDashboard() {
    const dashboardSection = document.getElementById('dashboard');
    if (!dashboardSection) return;

    try {
        const tbody = document.getElementById('instances-tbody');
        const errorDiv = document.getElementById('dashboard-error');
        const loadingDiv = document.getElementById('dashboard-loading');

        // Save expansion state BEFORE clearing tbody
        const expandedInstanceIds = new Set();
        if (tbody) {
            tbody.querySelectorAll('.instance-detail').forEach(detailRow => {
                if (detailRow.style.display === 'table-row') {
                    const instanceRow = detailRow.previousElementSibling;
                    if (instanceRow) {
                        const instanceId = instanceRow.getAttribute('data-instance-id');
                        if (instanceId) {
                            expandedInstanceIds.add(instanceId);
                        }
                    }
                }
            });
        }

        if (loadingDiv) loadingDiv.style.display = 'block';
        if (errorDiv) errorDiv.style.display = 'none';
        if (tbody) tbody.innerHTML = '';

        // Check AWS SDK
        if (typeof AWS === 'undefined') {
            throw new Error('AWS SDK not loaded. Please refresh the page.');
        }

        // Load instances
        const response = await DashboardAPI.listInstances();

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (response.success && response.instances && response.instances.length > 0) {
            // Cache instances for filtering
            allInstancesCache = response.instances;

            // Populate region filter
            populateRegionFilter(response.instances);

            // Apply current filters
            applyTableFilters();
        } else {
            allInstancesCache = [];
            if (tbody) {
                tbody.innerHTML = `
                    <tr>
                        <td colspan="7" style="text-align: center; padding: 3rem;">
                            <div style="max-width: 600px; margin: 0 auto;">
                                <div style="font-size: 3rem; margin-bottom: 1rem;">🍄</div>
                                <h3 style="color: var(--accent-blue); margin-bottom: 1rem;">No Spores Yet</h3>
                                <p style="color: var(--text-secondary); margin-bottom: 2rem; line-height: 1.8;">
                                    Spores are EC2 instances launched via the Spawn CLI. They'll appear here automatically once created.
                                </p>
                                <div style="background: rgba(79, 195, 247, 0.08); border: 1px solid rgba(79, 195, 247, 0.3); border-radius: 8px; padding: 1.5rem; text-align: left;">
                                    <h4 style="color: var(--accent-blue); margin-bottom: 1rem; text-align: center;">🚀 Quick Start</h4>
                                    <ol style="margin-left: 1.5rem; line-height: 2;">
                                        <li><strong>Install the CLI:</strong> <code style="background: var(--bg-dark); padding: 0.2rem 0.5rem; border-radius: 4px;">brew install scttfrdmn/tap/spawn</code></li>
                                        <li><strong>Launch your first spore:</strong> <code style="background: var(--bg-dark); padding: 0.2rem 0.5rem; border-radius: 4px;">spawn launch</code></li>
                                        <li><strong>Watch it appear here:</strong> Refresh this page in ~30 seconds</li>
                                    </ol>
                                    <p style="text-align: center; margin-top: 1.5rem; color: var(--text-muted); font-size: 0.9rem;">
                                        <a href="#install" style="color: var(--accent-blue); text-decoration: none;">View full installation guide ↓</a>
                                    </p>
                                </div>
                            </div>
                        </td>
                    </tr>
                `;
            }
        }
    } catch (error) {
        console.error('Failed to load dashboard:', error);

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (errorDiv) {
            errorDiv.style.display = 'block';
            const errorMessage = error.message || 'Unknown error';

            if (errorMessage.includes('credentials') || errorMessage.includes('not authorized')) {
                errorDiv.innerHTML = `
                    <strong>⚠️ Authentication Required</strong><br>
                    Please configure your AWS credentials to view your instances.<br>
                    <small>Make sure your IAM user has EC2 read permissions.</small>
                `;
            } else {
                errorDiv.innerHTML = `<strong>Error:</strong> ${errorMessage}`;
            }
        }
    }
}

// Auto-refresh interval (30 seconds)
let dashboardRefreshInterval = null;

// Manual refresh function
async function refreshDashboard() {
    const btn = document.getElementById('refresh-btn');
    if (btn) {
        btn.disabled = true;
        btn.style.opacity = '0.5';
        btn.textContent = '⏳ Refreshing...';
    }

    try {
        await refreshCurrentDashboardView();
        updateLastRefreshedTime();
    } finally {
        if (btn) {
            btn.disabled = false;
            btn.style.opacity = '1';
            btn.textContent = '🔄 Refresh';
        }
    }
}

// Update last refreshed timestamp
function updateLastRefreshedTime() {
    const lastUpdated = document.getElementById('last-updated');
    if (lastUpdated) {
        const now = new Date();
        lastUpdated.textContent = `Last updated: ${now.toLocaleTimeString()}`;
    }
}

// Start auto-refresh
function startAutoRefresh() {
    // Clear any existing interval
    if (dashboardRefreshInterval) {
        clearInterval(dashboardRefreshInterval);
    }

    // Refresh every 30 seconds
    dashboardRefreshInterval = setInterval(async () => {
        console.log('Auto-refreshing dashboard...');
        await refreshCurrentDashboardView();
        updateLastRefreshedTime();
    }, 30000);

    console.log('Auto-refresh enabled (30s interval)');
}

// Stop auto-refresh (e.g., when user logs out)
function stopAutoRefresh() {
    if (dashboardRefreshInterval) {
        clearInterval(dashboardRefreshInterval);
        dashboardRefreshInterval = null;
        console.log('Auto-refresh disabled');
    }
}

function displayInstances(instances, expandedInstanceIds = new Set()) {
    const tbody = document.getElementById('instances-tbody');
    if (!tbody) return;

    // Group instances by job array ID
    const jobArrays = {};
    const standaloneInstances = [];

    instances.forEach(instance => {
        const jobArrayId = instance.tags['spawn:job-array-id'];
        if (jobArrayId) {
            if (!jobArrays[jobArrayId]) {
                jobArrays[jobArrayId] = {
                    id: jobArrayId,
                    name: instance.tags['spawn:job-array-name'] || jobArrayId,
                    size: parseInt(instance.tags['spawn:job-array-size']) || 0,
                    instances: []
                };
            }
            jobArrays[jobArrayId].instances.push(instance);
        } else {
            standaloneInstances.push(instance);
        }
    });

    // Sort job array instances by index
    Object.values(jobArrays).forEach(jobArray => {
        jobArray.instances.sort((a, b) => {
            const indexA = parseInt(a.tags['spawn:job-array-index']) || 0;
            const indexB = parseInt(b.tags['spawn:job-array-index']) || 0;
            return indexA - indexB;
        });
    });

    // Build list of instance IDs in order (for diffing)
    const newInstanceOrder = [];
    Object.values(jobArrays).forEach(jobArray => {
        jobArray.instances.forEach(instance => {
            newInstanceOrder.push(instance.instance_id);
        });
    });
    standaloneInstances.forEach(instance => {
        newInstanceOrder.push(instance.instance_id);
    });

    // Get current instance IDs
    const existingRows = Array.from(tbody.querySelectorAll('tr.instance-row'));
    const existingInstanceIds = existingRows.map(row => row.getAttribute('data-instance-id')).filter(id => id);

    // Check if structure changed (new/removed instances or reordering)
    const structureChanged =
        newInstanceOrder.length !== existingInstanceIds.length ||
        newInstanceOrder.some((id, i) => id !== existingInstanceIds[i]);

    // If structure changed, do full rebuild
    if (structureChanged) {
        // Build HTML
        let html = '';
        let globalIndex = 0;

        // Display job arrays first
        Object.values(jobArrays).forEach(jobArray => {
            const runningCount = jobArray.instances.filter(i => i.state === 'running').length;
            const terminatedCount = jobArray.instances.filter(i => i.state === 'terminated').length;
            const pendingCount = jobArray.instances.filter(i => i.state === 'pending').length;

            html += `
                <tr class="job-array-header" style="background: rgba(79, 195, 247, 0.08); border-left: 3px solid var(--accent-blue);">
                    <td colspan="7" style="padding: 0.8rem 1rem;">
                        <div style="display: flex; justify-content: space-between; align-items: center;">
                            <div>
                                <strong style="color: var(--accent-blue);">🔗 Job Array:</strong>
                                <strong>${escapeHtml(jobArray.name)}</strong>
                                <span style="color: var(--text-muted); margin-left: 1rem;">
                                    ${jobArray.instances.length} of ${jobArray.size} instances
                                    ${runningCount > 0 ? `• ${runningCount} running` : ''}
                                    ${pendingCount > 0 ? `• ${pendingCount} pending` : ''}
                                    ${terminatedCount > 0 ? `• ${terminatedCount} terminated` : ''}
                                </span>
                            </div>
                        </div>
                    </td>
                </tr>
            `;

            jobArray.instances.forEach(instance => {
                html += renderInstanceRow(instance, globalIndex++, expandedInstanceIds, true);
            });

            html += `
                <tr class="job-array-spacer" style="height: 1rem; background: transparent;">
                    <td colspan="7" style="padding: 0; border: none;"></td>
                </tr>
            `;
        });

        standaloneInstances.forEach(instance => {
            html += renderInstanceRow(instance, globalIndex++, expandedInstanceIds, false);
        });

        tbody.style.opacity = '0';
        setTimeout(() => {
            tbody.innerHTML = html;
            tbody.style.opacity = '1';
        }, 150);
    } else {
        // Structure unchanged, update cells in place
        updateInstanceRows(instances, jobArrays);
    }
}

function updateInstanceRows(instances, jobArrays) {
    const tbody = document.getElementById('instances-tbody');
    if (!tbody) return;

    // Create map of instance data
    const instanceMap = new Map();
    instances.forEach(instance => {
        instanceMap.set(instance.instance_id, instance);
    });

    // Update each instance row
    tbody.querySelectorAll('tr.instance-row').forEach(row => {
        const instanceId = row.getAttribute('data-instance-id');
        const instance = instanceMap.get(instanceId);
        if (!instance) return;

        const launchTime = new Date(instance.launch_time);
        const stateClass = getStateClass(instance.state);

        // Calculate TTL remaining first
        let ttlRemaining = null;
        if (instance.ttl) {
            const ttlMinutes = parseTTL(instance.ttl);
            const elapsed = Math.floor((Date.now() - launchTime.getTime()) / 60000);
            const remaining = ttlMinutes - elapsed;
            if (remaining > 0) {
                ttlRemaining = formatDuration(remaining);
            } else {
                ttlRemaining = 'Expired';
            }
        }

        // Update state badge
        const stateTd = row.children[2];
        if (stateTd) {
            const newState = `<span class="badge badge-${stateClass}">${escapeHtml(instance.state)}</span>`;
            if (stateTd.innerHTML !== newState) {
                stateTd.style.transition = 'background-color 0.3s ease';
                stateTd.innerHTML = newState;
            }
        }

        // Update public IP
        const ipTd = row.children[4];
        if (ipTd) {
            const newIp = instance.public_ip ? `<code>${escapeHtml(instance.public_ip)}</code>` : '<span style="color: var(--text-muted);">—</span>';
            if (ipTd.innerHTML !== newIp) {
                ipTd.style.transition = 'opacity 0.3s ease';
                ipTd.style.opacity = '0.5';
                setTimeout(() => {
                    ipTd.innerHTML = newIp;
                    ipTd.style.opacity = '1';
                }, 150);
            }
        }

        // Update DNS name
        const dnsTd = row.children[5];
        if (dnsTd) {
            const newDns = instance.dns_name ? `<code>${escapeHtml(instance.dns_name)}</code>` : '<span style="color: var(--text-muted);">—</span>';
            if (dnsTd.innerHTML !== newDns) {
                dnsTd.style.transition = 'opacity 0.3s ease';
                dnsTd.style.opacity = '0.5';
                setTimeout(() => {
                    dnsTd.innerHTML = newDns;
                    dnsTd.style.opacity = '1';
                }, 150);
            }
        }

        // Update TTL display (ttlRemaining already calculated above)
        const ttlTd = row.children[6];
        if (ttlTd) {
            const ttlColor = ttlRemaining === 'Expired' ? 'var(--accent-red)' : ttlRemaining ? 'var(--accent-green)' : 'var(--text-muted)';

            let ttlDisplay;
            if (instance.state === 'terminated') {
                ttlDisplay = '<span style="color: var(--text-muted);">Terminated</span>';
            } else if (ttlRemaining) {
                ttlDisplay = `<span style="color: ${ttlColor};">${ttlRemaining}</span>`;
            } else {
                ttlDisplay = '<span style="color: var(--text-muted);">No TTL</span>';
            }

            if (ttlTd.innerHTML !== ttlDisplay) {
                ttlTd.style.transition = 'color 0.3s ease';
                ttlTd.innerHTML = ttlDisplay;
            }
        }
    });

    // Update job array headers
    Object.values(jobArrays).forEach(jobArray => {
        const runningCount = jobArray.instances.filter(i => i.state === 'running').length;
        const terminatedCount = jobArray.instances.filter(i => i.state === 'terminated').length;
        const pendingCount = jobArray.instances.filter(i => i.state === 'pending').length;

        // Find header row by checking for job array name
        const headers = tbody.querySelectorAll('tr.job-array-header');
        headers.forEach(header => {
            const headerText = header.textContent;
            if (headerText.includes(jobArray.name)) {
                const statusSpan = header.querySelector('span[style*="color: var(--text-muted)"]');
                if (statusSpan) {
                    const newStatus = `${jobArray.instances.length} of ${jobArray.size} instances
                                ${runningCount > 0 ? `• ${runningCount} running` : ''}
                                ${pendingCount > 0 ? `• ${pendingCount} pending` : ''}
                                ${terminatedCount > 0 ? `• ${terminatedCount} terminated` : ''}`;
                    if (statusSpan.textContent.trim() !== newStatus.trim()) {
                        statusSpan.style.transition = 'opacity 0.3s ease';
                        statusSpan.style.opacity = '0.5';
                        setTimeout(() => {
                            statusSpan.textContent = newStatus;
                            statusSpan.style.opacity = '1';
                        }, 150);
                    }
                }
            }
        });
    });
}

function renderInstanceRow(instance, index, expandedInstanceIds = new Set(), isJobArray = false) {
        const launchTime = new Date(instance.launch_time);
        const stateClass = getStateClass(instance.state);
        const rowId = `instance-${index}`;
        const detailId = `detail-${index}`;

        // Check if this instance should be expanded
        const isExpanded = expandedInstanceIds.has(instance.instance_id);
        const displayStyle = isExpanded ? 'table-row' : 'none';
        const arrowIcon = isExpanded ? '▲' : '▼';

        // Add indentation for job array instances
        const nameCellStyle = isJobArray ? 'padding-left: 2.5rem;' : '';

        // For terminated instances, show runtime. For others, show age.
        let ageDisplay;
        if (instance.state === 'terminated' && instance.termination_time) {
            const runtime = Math.floor((instance.termination_time - launchTime) / 1000 / 60); // minutes
            ageDisplay = `Ran ${formatDuration(runtime)}`;
        } else {
            ageDisplay = formatAge(launchTime);
        }

        // Calculate TTL remaining if present
        let ttlRemaining = null;
        let ttlColor = 'var(--text-muted)';
        if (instance.ttl) {
            const ttlMinutes = parseTTL(instance.ttl);
            const elapsed = Math.floor((Date.now() - launchTime.getTime()) / 60000);
            const remaining = ttlMinutes - elapsed;
            if (remaining > 0) {
                ttlRemaining = formatDuration(remaining);
                ttlColor = 'var(--accent-green)';
            } else {
                ttlRemaining = 'Expired';
                ttlColor = 'var(--accent-red)';
            }
        }

        // TTL display for table column
        let ttlDisplay;
        if (instance.state === 'terminated') {
            ttlDisplay = '<span style="color: var(--text-muted);">Terminated</span>';
        } else if (ttlRemaining) {
            ttlDisplay = `<span style="color: ${ttlColor};">${ttlRemaining}</span>`;
        } else {
            ttlDisplay = '<span style="color: var(--text-muted);">No TTL</span>';
        }

        return `
            <tr id="${rowId}" class="instance-row" data-instance-id="${escapeHtml(instance.instance_id)}" onclick="toggleInstanceDetails('${detailId}', '${rowId}')" style="cursor: pointer;">
                <td data-label="Name" style="${nameCellStyle}"><strong>${escapeHtml(instance.name)}</strong> <span style="color: var(--text-muted); font-size: 0.85rem;">${arrowIcon}</span></td>
                <td data-label="Type"><code>${escapeHtml(instance.instance_type)}</code></td>
                <td data-label="State"><span class="badge badge-${stateClass}">${escapeHtml(instance.state)}</span></td>
                <td data-label="Region"><code>${escapeHtml(instance.region)}</code></td>
                <td data-label="Public IP">${instance.public_ip ? `<code>${escapeHtml(instance.public_ip)}</code>` : '<span style="color: var(--text-muted);">—</span>'}</td>
                <td data-label="DNS Name">${instance.dns_name ? `<code>${escapeHtml(instance.dns_name)}</code>` : '<span style="color: var(--text-muted);">—</span>'}</td>
                <td data-label="TTL">${ttlDisplay}</td>
            </tr>
            <tr id="${detailId}" class="instance-detail" style="display: ${displayStyle};">
                <td colspan="7" style="padding: 0; background: rgba(79, 195, 247, 0.03);">
                    <div style="padding: 1.5rem; border-top: 1px solid var(--border);">
                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem;">
                            <div>
                                <h4 style="margin: 0 0 0.5rem 0; color: var(--accent-blue); font-size: 0.9rem;">Instance Details</h4>
                                <div style="font-size: 0.9rem; line-height: 1.8;">
                                    <div><strong>Instance ID:</strong> <code>${escapeHtml(instance.instance_id)}</code></div>
                                    <div><strong>AZ:</strong> <code>${escapeHtml(instance.availability_zone)}</code></div>
                                    <div><strong>Private IP:</strong> ${instance.private_ip ? `<code>${escapeHtml(instance.private_ip)}</code>` : '<span style="color: var(--text-muted);">—</span>'}</div>
                                    <div><strong>Spot:</strong> ${instance.spot_instance ? '<span style="color: var(--accent-green);">Yes</span>' : '<span style="color: var(--text-muted);">No</span>'}</div>
                                    ${instance.key_name ? `<div><strong>Key Pair:</strong> <code>${escapeHtml(instance.key_name)}</code></div>` : ''}
                                </div>
                            </div>
                            <div>
                                <h4 style="margin: 0 0 0.5rem 0; color: var(--accent-blue); font-size: 0.9rem;">Lifecycle</h4>
                                <div style="font-size: 0.9rem; line-height: 1.8;">
                                    <div><strong>Launched:</strong> <span style="color: var(--text-secondary);">${launchTime.toLocaleString()}</span></div>
                                    ${instance.termination_time ? `<div><strong>Terminated:</strong> <span style="color: var(--text-secondary);">${instance.termination_time.toLocaleString()}</span></div>` : ''}
                                    ${instance.termination_time ? `<div><strong>Runtime:</strong> <span style="color: var(--text-secondary);">${ageDisplay}</span></div>` : ''}
                                    ${instance.ttl ? `<div><strong>TTL:</strong> <code>${escapeHtml(instance.ttl)}</code></div>` : ''}
                                    ${ttlRemaining && !instance.termination_time ? `<div><strong>TTL Remaining:</strong> <span style="color: ${ttlRemaining === 'Expired' ? 'var(--accent-red)' : 'var(--accent-green)'};">${ttlRemaining}</span></div>` : ''}
                                    ${instance.tags['spawn:idle-timeout'] ? `<div><strong>Idle Timeout:</strong> <code>${escapeHtml(instance.tags['spawn:idle-timeout'])}</code></div>` : ''}
                                    ${instance.tags['spawn:session-timeout'] ? `<div><strong>Session Timeout:</strong> <code>${escapeHtml(instance.tags['spawn:session-timeout'])}</code></div>` : ''}
                                </div>
                            </div>
                        </div>
                        <div style="margin-top: 1.5rem;">
                            <h4 style="margin: 0 0 0.5rem 0; color: var(--accent-blue); font-size: 0.9rem;">Tags</h4>
                            <div style="font-size: 0.85rem; line-height: 1.6;">
                                ${Object.entries(instance.tags)
                                    .filter(([key]) => !key.startsWith('aws:') && key !== 'Name')
                                    .map(([key, value]) => `<div><code style="color: var(--accent-blue);">${escapeHtml(key)}</code>: <span style="color: var(--text-secondary);">${escapeHtml(value)}</span></div>`)
                                    .join('') || '<span style="color: var(--text-muted);">No custom tags</span>'}
                            </div>
                        </div>
                    </div>
                </td>
            </tr>
        `;
}

function toggleInstanceDetails(detailId, rowId) {
    const detailRow = document.getElementById(detailId);
    const instanceRow = document.getElementById(rowId);

    if (!detailRow || !instanceRow) return;

    const isVisible = detailRow.style.display !== 'none';

    // Toggle display
    detailRow.style.display = isVisible ? 'none' : 'table-row';

    // Update arrow indicator
    const arrow = instanceRow.querySelector('span');
    if (arrow) {
        arrow.textContent = isVisible ? '▼' : '▲';
    }
}

function parseTTL(ttlStr) {
    // Parse TTL string like "1h", "30m", "2h30m" to minutes
    const hours = ttlStr.match(/(\d+)h/);
    const minutes = ttlStr.match(/(\d+)m/);

    let total = 0;
    if (hours) total += parseInt(hours[1]) * 60;
    if (minutes) total += parseInt(minutes[1]);

    return total;
}

function formatDuration(minutes) {
    if (minutes < 60) {
        return `${minutes}m`;
    }
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
}

function formatAge(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m`;
    return `${seconds}s`;
}

function getStateClass(state) {
    const stateMap = {
        'running': 'success',
        'stopped': 'warning',
        'terminated': 'danger',
        'pending': 'info',
        'stopping': 'warning',
        'shutting-down': 'danger'
    };
    return stateMap[state] || 'default';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Table filtering, searching, and sorting
let allInstancesCache = [];
let currentSort = { column: null, direction: 'asc' };

function applyTableFilters() {
    if (allInstancesCache.length === 0) return;

    const searchTerm = document.getElementById('search-input')?.value.toLowerCase() || '';
    const stateFilter = document.getElementById('state-filter')?.value || '';
    const regionFilter = document.getElementById('region-filter')?.value || '';

    let filteredInstances = allInstancesCache.filter(instance => {
        // Search filter
        const searchMatch = !searchTerm ||
            instance.name.toLowerCase().includes(searchTerm) ||
            instance.instance_id.toLowerCase().includes(searchTerm) ||
            instance.instance_type.toLowerCase().includes(searchTerm) ||
            Object.values(instance.tags).some(v => String(v).toLowerCase().includes(searchTerm));

        // State filter
        const stateMatch = !stateFilter || instance.state === stateFilter;

        // Region filter
        const regionMatch = !regionFilter || instance.region === regionFilter;

        return searchMatch && stateMatch && regionMatch;
    });

    // Apply current sort
    if (currentSort.column) {
        filteredInstances = sortInstances(filteredInstances, currentSort.column, currentSort.direction);
    }

    // Preserve expansion state
    const tbody = document.getElementById('instances-tbody');
    const expandedInstanceIds = new Set();
    if (tbody) {
        tbody.querySelectorAll('.instance-detail').forEach(detailRow => {
            if (detailRow.style.display === 'table-row') {
                const instanceRow = detailRow.previousElementSibling;
                if (instanceRow) {
                    const instanceId = instanceRow.getAttribute('data-instance-id');
                    if (instanceId) {
                        expandedInstanceIds.add(instanceId);
                    }
                }
            }
        });
    }

    displayInstances(filteredInstances, expandedInstanceIds);
}

function clearTableFilters() {
    const searchInput = document.getElementById('search-input');
    const stateFilter = document.getElementById('state-filter');
    const regionFilter = document.getElementById('region-filter');

    if (searchInput) searchInput.value = '';
    if (stateFilter) stateFilter.value = '';
    if (regionFilter) regionFilter.value = '';

    applyTableFilters();
}

function sortTable(column) {
    // Toggle direction if same column, otherwise default to ascending
    if (currentSort.column === column) {
        currentSort.direction = currentSort.direction === 'asc' ? 'desc' : 'asc';
    } else {
        currentSort.column = column;
        currentSort.direction = 'asc';
    }

    // Update sort indicators
    ['name', 'type', 'state', 'region', 'ip', 'dns', 'ttl'].forEach(col => {
        const indicator = document.getElementById(`sort-${col}`);
        if (indicator) {
            if (col === column) {
                indicator.textContent = currentSort.direction === 'asc' ? '▲' : '▼';
                indicator.style.color = 'var(--accent-blue)';
            } else {
                indicator.textContent = '';
            }
        }
    });

    applyTableFilters();
}

function sortInstances(instances, column, direction) {
    return [...instances].sort((a, b) => {
        let aVal, bVal;

        switch (column) {
            case 'name':
                aVal = a.name.toLowerCase();
                bVal = b.name.toLowerCase();
                break;
            case 'type':
                aVal = a.instance_type;
                bVal = b.instance_type;
                break;
            case 'state':
                aVal = a.state;
                bVal = b.state;
                break;
            case 'region':
                aVal = a.region;
                bVal = b.region;
                break;
            case 'ip':
                aVal = a.public_ip || '';
                bVal = b.public_ip || '';
                break;
            case 'dns':
                aVal = a.dns_name || '';
                bVal = b.dns_name || '';
                break;
            case 'ttl':
                aVal = a.ttl ? parseTTL(a.ttl) : 0;
                bVal = b.ttl ? parseTTL(b.ttl) : 0;
                break;
            default:
                return 0;
        }

        if (aVal < bVal) return direction === 'asc' ? -1 : 1;
        if (aVal > bVal) return direction === 'asc' ? 1 : -1;
        return 0;
    });
}

function populateRegionFilter(instances) {
    const regionFilter = document.getElementById('region-filter');
    if (!regionFilter) return;

    // Get unique regions
    const regions = [...new Set(instances.map(i => i.region))].sort();

    // Preserve current selection
    const currentValue = regionFilter.value;

    // Clear and repopulate
    regionFilter.innerHTML = '<option value="">All Regions</option>';
    regions.forEach(region => {
        const option = document.createElement('option');
        option.value = region;
        option.textContent = region;
        regionFilter.appendChild(option);
    });

    // Restore selection
    regionFilter.value = currentValue;
}

// ==================== SWEEP MANAGEMENT ====================

// Current dashboard state
let currentDashboardTab = 'instances';
let allSweepsCache = [];
let sweepSortState = { column: null, direction: 'asc' };

// Tab switching
function switchDashboardTab(tab) {
    currentDashboardTab = tab;

    // Update tab buttons
    const instancesTab = document.getElementById('tab-instances');
    const sweepsTab = document.getElementById('tab-sweeps');
    const autoscaleTab = document.getElementById('tab-autoscale');
    const instancesContent = document.getElementById('instances-tab-content');
    const sweepsContent = document.getElementById('sweeps-tab-content');
    const autoscaleContent = document.getElementById('autoscale-tab-content');

    // Reset all tabs
    [instancesTab, sweepsTab, autoscaleTab].forEach(t => {
        if (t) {
            t.classList.remove('active');
            t.style.borderBottom = '3px solid transparent';
            t.style.color = 'var(--text-muted)';
        }
    });

    [instancesContent, sweepsContent, autoscaleContent].forEach(c => {
        if (c) c.style.display = 'none';
    });

    if (tab === 'instances') {
        instancesTab.classList.add('active');
        instancesTab.style.borderBottom = '3px solid var(--accent-blue)';
        instancesTab.style.color = 'var(--accent-blue)';
        instancesContent.style.display = 'block';
        loadDashboard();
    } else if (tab === 'sweeps') {
        sweepsTab.classList.add('active');
        sweepsTab.style.borderBottom = '3px solid var(--accent-blue)';
        sweepsTab.style.color = 'var(--accent-blue)';
        sweepsContent.style.display = 'block';
        loadSweeps();
    } else if (tab === 'autoscale') {
        autoscaleTab.classList.add('active');
        autoscaleTab.style.borderBottom = '3px solid var(--accent-blue)';
        autoscaleTab.style.color = 'var(--accent-blue)';
        autoscaleContent.style.display = 'block';
        loadAutoscaleGroups();
        loadCostSummary();
    }
}

// Refresh current view
async function refreshCurrentDashboardView() {
    if (currentDashboardTab === 'instances') {
        await loadDashboard();
    } else if (currentDashboardTab === 'sweeps') {
        await loadSweeps();
    } else if (currentDashboardTab === 'autoscale') {
        await loadAutoscaleGroups();
        await loadCostSummary();
    }
}

// Load sweeps from API
async function loadSweeps() {
    const tbody = document.getElementById('sweeps-tbody');
    const errorDiv = document.getElementById('sweeps-error');
    const loadingDiv = document.getElementById('sweeps-loading');

    try {
        if (loadingDiv) loadingDiv.style.display = 'block';
        if (errorDiv) errorDiv.style.display = 'none';
        if (tbody) tbody.innerHTML = '';

        // Call Lambda API endpoint
        const apiEndpoint = 'https://api.spore.host/api/sweeps';

        const response = await fetch(apiEndpoint, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                // Authentication will be added by auth.js interceptor
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`API returned ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (data.success && data.sweeps && data.sweeps.length > 0) {
            allSweepsCache = data.sweeps;
            applySweepFilters();
            updateLastRefreshedTime();
        } else {
            allSweepsCache = [];
            if (tbody) {
                tbody.innerHTML = `
                    <tr>
                        <td colspan="7" style="text-align: center; padding: 3rem;">
                            <div style="max-width: 600px; margin: 0 auto;">
                                <div style="font-size: 3rem; margin-bottom: 1rem;">📊</div>
                                <h3 style="color: var(--accent-blue); margin-bottom: 1rem;">No Sweeps Yet</h3>
                                <p style="color: var(--text-secondary); margin-bottom: 2rem; line-height: 1.8;">
                                    Parameter sweeps let you launch multiple instances with different configurations. They'll appear here automatically.
                                </p>
                                <div style="background: rgba(79, 195, 247, 0.08); border: 1px solid rgba(79, 195, 247, 0.3); border-radius: 8px; padding: 1.5rem; text-align: left;">
                                    <h4 style="color: var(--accent-blue); margin-bottom: 1rem; text-align: center;">🚀 Launch Your First Sweep</h4>
                                    <pre style="background: var(--bg-dark); padding: 1rem; border-radius: 4px; overflow-x: auto;">spawn sweep --file params.json --detach</pre>
                                    <p style="text-align: center; margin-top: 1rem; color: var(--text-muted); font-size: 0.9rem;">
                                        The <code>--detach</code> flag runs the sweep in Lambda so you can close your terminal.
                                    </p>
                                </div>
                            </div>
                        </td>
                    </tr>
                `;
            }
        }
    } catch (error) {
        console.error('Failed to load sweeps:', error);

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (errorDiv) {
            errorDiv.style.display = 'block';
            const errorMessage = error.message || 'Unknown error';
            errorDiv.innerHTML = `<strong>Error:</strong> ${errorMessage}`;
        }

        if (tbody) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="7" style="text-align: center; padding: 2rem; color: var(--accent-red);">
                        Failed to load sweeps. Please try again.
                    </td>
                </tr>
            `;
        }
    }
}

// Render sweeps table
function renderSweepsTable(sweeps) {
    const tbody = document.getElementById('sweeps-tbody');
    if (!tbody) return;

    if (sweeps.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" style="text-align: center; padding: 2rem; color: var(--text-muted);">
                    No sweeps match your filters.
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = sweeps.map(sweep => {
        const statusIcon = getSweepStatusIcon(sweep.status);
        const progress = `${sweep.launched}/${sweep.total_params}`;
        const progressPercent = sweep.total_params > 0 ? (sweep.launched / sweep.total_params * 100) : 0;
        const failedText = sweep.failed > 0 ? ` (${sweep.failed} failed)` : '';
        const createdTime = formatRelativeTime(new Date(sweep.created_at));
        const costText = sweep.estimated_cost > 0 ? `$${sweep.estimated_cost.toFixed(2)}` : 'N/A';

        // Region display: show count for multi-region, single region otherwise
        const regionInfo = sweep.multi_region && sweep.region_status
            ? `${Object.keys(sweep.region_status).length} regions`
            : sweep.region;

        // Action buttons
        let actionButtons = '';
        if (sweep.status === 'RUNNING') {
            actionButtons = `
                <button onclick="cancelSweep('${sweep.sweep_id}')"
                        style="padding: 0.4rem 0.8rem; background: var(--accent-red); color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; transition: opacity 0.2s;"
                        onmouseover="this.style.opacity='0.8'"
                        onmouseout="this.style.opacity='1'">
                    ❌ Cancel
                </button>
            `;
        } else {
            actionButtons = `<span style="color: var(--text-muted); font-size: 0.85rem;">—</span>`;
        }

        return `
            <tr data-sweep-id="${sweep.sweep_id}">
                <td>
                    <div style="font-weight: 500;">${sweep.sweep_name || sweep.sweep_id}</div>
                    <div style="font-size: 0.85rem; color: var(--text-muted); font-family: monospace; margin-top: 0.2rem;">${sweep.sweep_id}</div>
                </td>
                <td>
                    <span style="display: inline-flex; align-items: center; gap: 0.4rem;">
                        <span style="font-size: 1.2rem;">${statusIcon}</span>
                        <span>${sweep.status}</span>
                    </span>
                </td>
                <td>
                    <div style="margin-bottom: 0.3rem;">${progress}${failedText}</div>
                    <div style="width: 100%; background: var(--bg-dark); border-radius: 4px; height: 6px; overflow: hidden;">
                        <div style="width: ${progressPercent}%; background: var(--accent-blue); height: 100%; transition: width 0.3s;"></div>
                    </div>
                </td>
                <td>${regionInfo}</td>
                <td title="${new Date(sweep.created_at).toLocaleString()}">${createdTime}</td>
                <td>${costText}</td>
                <td>${actionButtons}</td>
            </tr>
        `;
    }).join('');
}

// Get status icon
function getSweepStatusIcon(status) {
    switch (status.toUpperCase()) {
        case 'RUNNING':
        case 'INITIALIZING':
            return '🚀';
        case 'COMPLETED':
            return '✅';
        case 'FAILED':
            return '❌';
        case 'CANCELLED':
            return '⚠️';
        default:
            return '❓';
    }
}

// Format relative time
function formatRelativeTime(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (seconds < 60) return 'just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
}

// Apply sweep filters
function applySweepFilters() {
    const searchInput = document.getElementById('sweep-search-input');
    const statusFilter = document.getElementById('sweep-status-filter');

    const searchTerm = searchInput ? searchInput.value.toLowerCase() : '';
    const statusValue = statusFilter ? statusFilter.value : '';

    let filtered = allSweepsCache.filter(sweep => {
        // Search filter
        if (searchTerm) {
            const name = (sweep.sweep_name || '').toLowerCase();
            const id = sweep.sweep_id.toLowerCase();
            if (!name.includes(searchTerm) && !id.includes(searchTerm)) {
                return false;
            }
        }

        // Status filter
        if (statusValue && sweep.status !== statusValue) {
            return false;
        }

        return true;
    });

    // Apply sorting
    if (sweepSortState.column) {
        filtered = sortSweeps(filtered, sweepSortState.column, sweepSortState.direction);
    }

    renderSweepsTable(filtered);
}

// Sort sweeps
function sortSweeps(sweeps, column, direction) {
    return sweeps.sort((a, b) => {
        let aVal, bVal;

        switch (column) {
            case 'name':
                aVal = (a.sweep_name || a.sweep_id).toLowerCase();
                bVal = (b.sweep_name || b.sweep_id).toLowerCase();
                break;
            case 'status':
                aVal = a.status;
                bVal = b.status;
                break;
            case 'progress':
                aVal = a.total_params > 0 ? (a.launched / a.total_params) : 0;
                bVal = b.total_params > 0 ? (b.launched / b.total_params) : 0;
                break;
            case 'region':
                aVal = a.region;
                bVal = b.region;
                break;
            case 'created':
                aVal = new Date(a.created_at).getTime();
                bVal = new Date(b.created_at).getTime();
                break;
            case 'cost':
                aVal = a.estimated_cost || 0;
                bVal = b.estimated_cost || 0;
                break;
            default:
                return 0;
        }

        if (aVal < bVal) return direction === 'asc' ? -1 : 1;
        if (aVal > bVal) return direction === 'asc' ? 1 : -1;
        return 0;
    });
}

// Clear sweep filters
function clearSweepFilters() {
    const searchInput = document.getElementById('sweep-search-input');
    const statusFilter = document.getElementById('sweep-status-filter');

    if (searchInput) searchInput.value = '';
    if (statusFilter) statusFilter.value = '';

    applySweepFilters();
}

// Sort sweeps table
function sortSweepsTable(column) {
    // Update sort state
    if (sweepSortState.column === column) {
        sweepSortState.direction = sweepSortState.direction === 'asc' ? 'desc' : 'asc';
    } else {
        sweepSortState.column = column;
        sweepSortState.direction = 'asc';
    }

    // Update sort indicators
    ['name', 'status', 'progress', 'region', 'created', 'cost'].forEach(col => {
        const indicator = document.getElementById(`sweep-sort-${col}`);
        if (indicator) {
            if (col === column) {
                indicator.textContent = sweepSortState.direction === 'asc' ? '▲' : '▼';
            } else {
                indicator.textContent = '';
            }
        }
    });

    // Sort filtered results
    applySweepFilters();
}

// Cancel sweep
async function cancelSweep(sweepId) {
    if (!confirm('Are you sure you want to cancel this sweep? Running instances will be terminated.')) {
        return;
    }

    try {
        const apiEndpoint = `https://api.spore.host/api/sweeps/${sweepId}/cancel`;

        const response = await fetch(apiEndpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`API returned ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();

        if (data.success) {
            alert(`Sweep cancelled successfully. Terminated ${data.instances_terminated} instance(s).`);
            await loadSweeps();
        } else {
            throw new Error(data.error || 'Failed to cancel sweep');
        }
    } catch (error) {
        console.error('Failed to cancel sweep:', error);
        alert(`Failed to cancel sweep: ${error.message}`);
    }
}

// ═══════════════════════════════════════════════════════════════
// Autoscale Groups Functions
// ═══════════════════════════════════════════════════════════════

let allAutoscaleGroupsCache = [];

// Load autoscale groups from API
async function loadAutoscaleGroups() {
    const tbody = document.getElementById('autoscale-tbody');
    const errorDiv = document.getElementById('autoscale-error');
    const loadingDiv = document.getElementById('autoscale-loading');

    try {
        if (loadingDiv) loadingDiv.style.display = 'block';
        if (errorDiv) errorDiv.style.display = 'none';
        if (tbody) tbody.innerHTML = '';

        const apiEndpoint = 'https://api.spore.host/api/autoscale-groups';

        const response = await fetch(apiEndpoint, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`API returned ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (data.success && data.autoscale_groups && data.autoscale_groups.length > 0) {
            allAutoscaleGroupsCache = data.autoscale_groups;
            applyAutoscaleFilters();
        } else {
            allAutoscaleGroupsCache = [];
            if (tbody) {
                tbody.innerHTML = `
                    <tr>
                        <td colspan="6" style="text-align: center; padding: 3rem;">
                            <div style="max-width: 600px; margin: 0 auto;">
                                <div style="font-size: 3rem; margin-bottom: 1rem;">⚙️</div>
                                <h3 style="color: var(--accent-blue); margin-bottom: 1rem;">No Autoscale Groups Yet</h3>
                                <p style="color: var(--text-secondary); margin-bottom: 2rem; line-height: 1.8;">
                                    Create autoscale groups to automatically manage capacity based on queues, metrics, or schedules.
                                </p>
                                <div style="background: rgba(79, 195, 247, 0.08); border: 1px solid rgba(79, 195, 247, 0.3); border-radius: 8px; padding: 1.5rem; text-align: left;">
                                    <h4 style="color: var(--accent-blue); margin-bottom: 1rem; text-align: center;">🚀 Quick Start</h4>
                                    <pre style="background: var(--bg-dark); padding: 1rem; border-radius: 4px; overflow-x: auto;"><code>spawn autoscale launch \\
  --name my-workers \\
  --desired 5 --min 2 --max 10 \\
  --instance-type t3.micro</code></pre>
                                </div>
                            </div>
                        </td>
                    </tr>
                `;
            }
        }
    } catch (error) {
        console.error('Failed to load autoscale groups:', error);

        if (loadingDiv) loadingDiv.style.display = 'none';

        if (errorDiv) {
            errorDiv.style.display = 'block';
            errorDiv.innerHTML = `<strong>Error:</strong> ${error.message || 'Unknown error'}`;
        }
    }
}

// Load cost summary
async function loadCostSummary() {
    try {
        const apiEndpoint = 'https://api.spore.host/api/cost-summary';

        const response = await fetch(apiEndpoint, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`API returned ${response.status}`);
        }

        const data = await response.json();

        if (data.success && data.cost) {
            const cost = data.cost;

            // Update cost widgets
            const hourlyElem = document.getElementById('cost-hourly');
            const monthlyElem = document.getElementById('cost-monthly');
            const countElem = document.getElementById('cost-instance-count');
            const breakdownElem = document.getElementById('cost-breakdown');

            if (hourlyElem) hourlyElem.textContent = `$${cost.total_hourly_cost.toFixed(2)}`;
            if (monthlyElem) monthlyElem.textContent = `$${cost.estimated_monthly_cost.toFixed(2)}`;
            if (countElem) countElem.textContent = cost.instance_count;

            // Format breakdown
            if (breakdownElem && cost.breakdown_by_type) {
                const breakdownHTML = Object.entries(cost.breakdown_by_type)
                    .sort((a, b) => b[1].count - a[1].count)
                    .map(([type, info]) =>
                        `${type}: ${info.count}x ($${info.hourly_cost.toFixed(2)}/hr)`
                    )
                    .join(' • ');

                breakdownElem.innerHTML = breakdownHTML || 'No running instances';
            }
        }
    } catch (error) {
        console.error('Failed to load cost summary:', error);
    }
}

// Render autoscale groups table
function renderAutoscaleGroupsTable(groups) {
    const tbody = document.getElementById('autoscale-tbody');
    if (!tbody) return;

    tbody.innerHTML = '';

    groups.forEach(group => {
        const row = document.createElement('tr');
        row.className = 'autoscale-group-row';
        row.style.cursor = 'pointer';
        row.onclick = () => toggleGroupDetails(group.autoscale_group_id);

        // Name
        const nameCell = document.createElement('td');
        nameCell.innerHTML = `
            <div style="font-weight: 500;">${group.group_name || group.autoscale_group_id}</div>
            <div style="font-size: 0.8rem; color: var(--text-muted);">${group.autoscale_group_id.substring(0, 12)}...</div>
        `;
        row.appendChild(nameCell);

        // Status
        const statusCell = document.createElement('td');
        statusCell.innerHTML = getGroupStatusBadge(group.status);
        row.appendChild(statusCell);

        // Capacity
        const capacityCell = document.createElement('td');
        capacityCell.innerHTML = `
            <div>${group.current_capacity} / ${group.desired_capacity}</div>
            <div style="font-size: 0.8rem; color: var(--text-muted);">
                ${group.current_capacity === group.desired_capacity ? '✅ At target' : '⏳ Scaling'}
            </div>
        `;
        row.appendChild(capacityCell);

        // Min/Max
        const rangeCell = document.createElement('td');
        rangeCell.textContent = `${group.min_capacity} / ${group.max_capacity}`;
        row.appendChild(rangeCell);

        // Policy
        const policyCell = document.createElement('td');
        policyCell.innerHTML = formatPolicyType(group.policy_type);
        row.appendChild(policyCell);

        // Last Event
        const eventCell = document.createElement('td');
        eventCell.textContent = formatRelativeTime(new Date(group.last_scale_event));
        eventCell.style.color = 'var(--text-muted)';
        row.appendChild(eventCell);

        tbody.appendChild(row);

        // Add details row (hidden by default)
        const detailRow = document.createElement('tr');
        detailRow.id = `detail-${group.autoscale_group_id}`;
        detailRow.className = 'autoscale-group-detail';
        detailRow.style.display = 'none';

        const detailCell = document.createElement('td');
        detailCell.colSpan = 6;
        detailCell.innerHTML = '<div style="padding: 1rem; text-align: center; color: var(--text-muted);">Loading details...</div>';
        detailRow.appendChild(detailCell);

        tbody.appendChild(detailRow);
    });
}

// Toggle group details
async function toggleGroupDetails(groupId) {
    const detailRow = document.getElementById(`detail-${groupId}`);
    if (!detailRow) return;

    if (detailRow.style.display === 'table-row') {
        detailRow.style.display = 'none';
        return;
    }

    // Show loading
    detailRow.style.display = 'table-row';
    detailRow.querySelector('td').innerHTML = '<div style="padding: 1rem; text-align: center; color: var(--text-muted);">Loading details...</div>';

    try {
        const apiEndpoint = `https://api.spore.host/api/autoscale-groups/${groupId}`;

        const response = await fetch(apiEndpoint, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'include'
        });

        if (!response.ok) {
            throw new Error(`API returned ${response.status}`);
        }

        const data = await response.json();

        if (data.success && data.group) {
            renderGroupDetails(detailRow, data.group);
        } else {
            throw new Error('Failed to load group details');
        }
    } catch (error) {
        console.error('Failed to load group details:', error);
        detailRow.querySelector('td').innerHTML = `<div style="padding: 1rem; text-align: center; color: var(--accent-red);">Error: ${error.message}</div>`;
    }
}

// Render group details
function renderGroupDetails(detailRow, group) {
    const cell = detailRow.querySelector('td');

    let detailHTML = `
        <div style="padding: 1.5rem; background: rgba(0, 0, 0, 0.2); border-radius: 8px;">
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 1.5rem;">
                <div>
                    <div style="font-size: 0.85rem; color: var(--text-muted); margin-bottom: 0.5rem;">Healthy Instances</div>
                    <div style="font-size: 1.5rem; font-weight: bold; color: var(--accent-green);">${group.healthy_count}</div>
                </div>
                <div>
                    <div style="font-size: 0.85rem; color: var(--text-muted); margin-bottom: 0.5rem;">Unhealthy Instances</div>
                    <div style="font-size: 1.5rem; font-weight: bold; color: var(--accent-red);">${group.unhealthy_count}</div>
                </div>
                <div>
                    <div style="font-size: 0.85rem; color: var(--text-muted); margin-bottom: 0.5rem;">Pending Instances</div>
                    <div style="font-size: 1.5rem; font-weight: bold; color: var(--accent-blue);">${group.pending_count}</div>
                </div>
            </div>
    `;

    if (group.instances && group.instances.length > 0) {
        detailHTML += `
            <h4 style="margin-bottom: 1rem; color: var(--text-primary);">Instances</h4>
            <div style="overflow-x: auto;">
                <table style="width: 100%; border-collapse: collapse;">
                    <thead>
                        <tr style="background: rgba(0, 0, 0, 0.3);">
                            <th style="padding: 0.5rem; text-align: left; border-bottom: 1px solid var(--border);">Instance ID</th>
                            <th style="padding: 0.5rem; text-align: left; border-bottom: 1px solid var(--border);">State</th>
                            <th style="padding: 0.5rem; text-align: left; border-bottom: 1px solid var(--border);">Health</th>
                            <th style="padding: 0.5rem; text-align: left; border-bottom: 1px solid var(--border);">Launched</th>
                        </tr>
                    </thead>
                    <tbody>
        `;

        group.instances.forEach(inst => {
            const healthBadge = inst.health_status === 'healthy'
                ? '<span style="color: var(--accent-green);">✅ Healthy</span>'
                : inst.health_status === 'pending'
                ? '<span style="color: var(--accent-blue);">⏳ Pending</span>'
                : '<span style="color: var(--accent-red);">❌ Unhealthy</span>';

            detailHTML += `
                <tr style="border-bottom: 1px solid rgba(255, 255, 255, 0.05);">
                    <td style="padding: 0.5rem; font-family: monospace; font-size: 0.9rem;">${inst.instance_id}</td>
                    <td style="padding: 0.5rem;">${inst.state}</td>
                    <td style="padding: 0.5rem;">${healthBadge}</td>
                    <td style="padding: 0.5rem; color: var(--text-muted); font-size: 0.9rem;">${formatRelativeTime(new Date(inst.launched_at))}</td>
                </tr>
            `;
        });

        detailHTML += `
                    </tbody>
                </table>
            </div>
        `;
    } else {
        detailHTML += '<p style="color: var(--text-muted); text-align: center;">No instances</p>';
    }

    detailHTML += '</div>';

    cell.innerHTML = detailHTML;
}

// Get group status badge
function getGroupStatusBadge(status) {
    switch (status) {
        case 'active':
            return '<span style="background: rgba(102, 187, 106, 0.2); color: var(--accent-green); padding: 0.25rem 0.75rem; border-radius: 12px; font-size: 0.85rem; font-weight: 500;">✅ Active</span>';
        case 'paused':
            return '<span style="background: rgba(255, 193, 7, 0.2); color: #FFC107; padding: 0.25rem 0.75rem; border-radius: 12px; font-size: 0.85rem; font-weight: 500;">⏸️ Paused</span>';
        case 'terminated':
            return '<span style="background: rgba(244, 67, 54, 0.2); color: var(--accent-red); padding: 0.25rem 0.75rem; border-radius: 12px; font-size: 0.85rem; font-weight: 500;">❌ Terminated</span>';
        default:
            return '<span style="background: rgba(158, 158, 158, 0.2); color: var(--text-muted); padding: 0.25rem 0.75rem; border-radius: 12px; font-size: 0.85rem; font-weight: 500;">❓ Unknown</span>';
    }
}

// Format policy type
function formatPolicyType(type) {
    switch (type) {
        case 'queue':
            return '<span style="color: var(--accent-blue);">📋 Queue-based</span>';
        case 'metric':
            return '<span style="color: var(--accent-green);">📊 Metric-based</span>';
        case 'schedule':
            return '<span style="color: #FFC107;">⏰ Scheduled</span>';
        case 'none':
            return '<span style="color: var(--text-muted);">⚙️ Manual</span>';
        default:
            return '<span style="color: var(--text-muted);">❓ Unknown</span>';
    }
}

// Apply autoscale filters
function applyAutoscaleFilters() {
    const searchInput = document.getElementById('autoscale-search-input');
    const statusFilter = document.getElementById('autoscale-status-filter');

    const searchTerm = searchInput ? searchInput.value.toLowerCase() : '';
    const statusValue = statusFilter ? statusFilter.value : '';

    let filtered = allAutoscaleGroupsCache.filter(group => {
        // Search filter
        if (searchTerm) {
            const name = (group.group_name || '').toLowerCase();
            const id = group.autoscale_group_id.toLowerCase();
            if (!name.includes(searchTerm) && !id.includes(searchTerm)) {
                return false;
            }
        }

        // Status filter
        if (statusValue && group.status !== statusValue) {
            return false;
        }

        return true;
    });

    renderAutoscaleGroupsTable(filtered);
}

// Clear autoscale filters
function clearAutoscaleFilters() {
    const searchInput = document.getElementById('autoscale-search-input');
    const statusFilter = document.getElementById('autoscale-status-filter');

    if (searchInput) searchInput.value = '';
    if (statusFilter) statusFilter.value = '';

    applyAutoscaleFilters();
}

// Auto-refresh sweeps (every 10 seconds if on sweeps tab)
setInterval(() => {
    if (currentDashboardTab === 'sweeps' && document.getElementById('sweeps-tab-content').style.display !== 'none') {
        loadSweeps();
    } else if (currentDashboardTab === 'autoscale' && document.getElementById('autoscale-tab-content').style.display !== 'none') {
        loadAutoscaleGroups();
        loadCostSummary();
    }
}, 10000);

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { DashboardAPI, showTab };
}
