// O Dan Go - Minimal Functional JavaScript with Credential Storage

const app = {
    currentView: 'welcome',
    
    init() {
        // Check and restore saved credentials (if not expired)
        this.restoreCredentials();
        
        // Set up navigation
        document.querySelectorAll('.nav a').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const view = link.getAttribute('href').substring(1);
                this.showView(view);
            });
        });
        
        // Set up form submission
        const searchForm = document.getElementById('searchForm');
        if (searchForm) {
            searchForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleSearch();
            });
        }
        
        // Set up phone number field interactions
        this.setupPhoneFields();
        
        // Show initial view based on URL hash
        const hash = window.location.hash.substring(1) || 'welcome';
        this.showView(hash);
        
        // Set up 4-hour expiry check
        this.startExpiryCheck();
    },
    
    restoreCredentials() {
        const savedData = localStorage.getItem('odango_credentials');
        if (savedData) {
            try {
                const data = JSON.parse(savedData);
                const now = new Date().getTime();
                
                // Check if expired (4 hours = 14400000 ms)
                if (now - data.timestamp < 14400000) {
                    // Restore credentials
                    if (document.getElementById('api_url')) {
                        document.getElementById('api_url').value = data.api_url || '';
                    }
                    if (document.getElementById('api_token')) {
                        document.getElementById('api_token').value = data.api_token || '';
                    }
                } else {
                    // Expired, clear storage
                    localStorage.removeItem('odango_credentials');
                }
            } catch (e) {
                // Invalid data, clear it
                localStorage.removeItem('odango_credentials');
            }
        }
    },
    
    saveCredentials() {
        const apiUrl = document.getElementById('api_url').value;
        const apiToken = document.getElementById('api_token').value;
        
        if (apiUrl && apiToken) {
            const data = {
                api_url: apiUrl,
                api_token: apiToken,
                timestamp: new Date().getTime()
            };
            localStorage.setItem('odango_credentials', JSON.stringify(data));
        }
    },
    
    clearCredentials() {
        localStorage.removeItem('odango_credentials');
        this.showMessage('Credentials cleared', 'success');
    },
    
    startExpiryCheck() {
        // Check every minute if credentials have expired
        setInterval(() => {
            const savedData = localStorage.getItem('odango_credentials');
            if (savedData) {
                const data = JSON.parse(savedData);
                const now = new Date().getTime();
                
                if (now - data.timestamp >= 14400000) {
                    localStorage.removeItem('odango_credentials');
                    if (this.currentView === 'search') {
                        this.showMessage('Your API credentials have expired. Please re-enter them.', 'error');
                        document.getElementById('api_url').value = '';
                        document.getElementById('api_token').value = '';
                    }
                }
            }
        }, 60000); // Check every minute
    },
    
    showView(viewName) {
        // Hide all views
        document.querySelectorAll('.view').forEach(view => {
            view.classList.remove('active');
        });
        
        // Show selected view
        const selectedView = document.getElementById(viewName + 'View');
        if (selectedView) {
            selectedView.classList.add('active');
            this.currentView = viewName;
            
            // Update navigation
            document.querySelectorAll('.nav a').forEach(link => {
                link.classList.remove('active');
                if (link.getAttribute('href') === '#' + viewName) {
                    link.classList.add('active');
                }
            });
            
            // Update URL
            window.location.hash = viewName;
        }
    },
    
    setupPhoneFields() {
        const originating = document.getElementById('originating_number');
        const terminating = document.getElementById('terminating_number');
        const anyPhone = document.getElementById('any_phone_number');
        
        if (!originating || !terminating || !anyPhone) return;
        
        // Disable specific fields when "any" is used
        anyPhone.addEventListener('input', (e) => {
            const hasValue = e.target.value.trim() !== '';
            originating.disabled = hasValue;
            terminating.disabled = hasValue;
            if (hasValue) {
                originating.value = '';
                terminating.value = '';
            }
        });
        
        // Disable "any" field when specific fields are used
        [originating, terminating].forEach(field => {
            field.addEventListener('input', () => {
                const hasSpecific = originating.value.trim() !== '' || 
                                  terminating.value.trim() !== '';
                anyPhone.disabled = hasSpecific;
                if (hasSpecific) {
                    anyPhone.value = '';
                }
            });
        });
    },
    
    handleSearch() {
        // Get form data
        const formData = new FormData(document.getElementById('searchForm'));
        
        // Validate API credentials
        const apiUrl = formData.get('api_url');
        const apiToken = formData.get('api_token');
        
        if (!apiUrl || !apiToken) {
            this.showMessage('Please enter your API URL and Bearer Token', 'error');
            return;
        }
        
        // Save credentials for 4 hours
        this.saveCredentials();
        
        // Basic validation for search criteria
        const hasSearchCriteria = Array.from(formData.entries())
            .filter(([key]) => !['api_url', 'api_token', 'limit'].includes(key))
            .some(([_, value]) => value.trim() !== '');
            
        if (!hasSearchCriteria) {
            this.showMessage('Please enter at least one search criterion', 'error');
            return;
        }
        
        // Show loading
        this.showLoading(true);
        
        // Submit form
        document.getElementById('searchForm').submit();
    },
    
    showMessage(text, type = 'info') {
        // Remove existing messages
        document.querySelectorAll('.message').forEach(msg => msg.remove());
        
        // Create new message
        const message = document.createElement('div');
        message.className = `message ${type}`;
        message.textContent = text;
        
        // Insert at top of current view
        const currentViewEl = document.getElementById(this.currentView + 'View');
        if (currentViewEl) {
            currentViewEl.insertBefore(message, currentViewEl.firstChild);
            
            // Auto-remove after 5 seconds
            setTimeout(() => {
                message.remove();
            }, 5000);
        }
    },
    
    showLoading(show) {
        const overlay = document.getElementById('loadingOverlay');
        if (overlay) {
            overlay.classList.toggle('active', show);
        }
    },
    
    // Utility function to display results (called from results page)
    displayResults(sessionId, results) {
        const resultsContainer = document.getElementById('resultsContainer');
        if (!resultsContainer) return;
        
        // Clear existing content
        resultsContainer.innerHTML = '';
        
        // Add summary
        const summary = document.createElement('div');
        summary.className = 'results-summary';
        summary.innerHTML = `
            <h3>Search Results - Session: ${sessionId}</h3>
            <p>Found ${results.uniqueCDRs || 0} unique CDRs from ${results.totalCDRs || 0} total records</p>
            <p>Query completed in ${results.queryTime || '0.00'} seconds across ${results.endpointCount || 0} endpoints</p>
        `;
        resultsContainer.appendChild(summary);
        
        // Add export buttons
        const exportDiv = document.createElement('div');
        exportDiv.style.marginBottom = '20px';
        exportDiv.innerHTML = `
            <button class="btn" onclick="app.export('csv', '${sessionId}')">Export CSV</button>
            <button class="btn" onclick="app.export('json', '${sessionId}')">Export JSON</button>
        `;
        resultsContainer.appendChild(exportDiv);
        
        // Add endpoint details
        if (results.endpoints && results.endpoints.length > 0) {
            const grid = document.createElement('div');
            grid.className = 'results-grid';
            
            results.endpoints.forEach(endpoint => {
                const card = document.createElement('div');
                card.className = 'result-card';
                card.innerHTML = `
                    <h4>${endpoint.URL || 'Unknown Endpoint'}</h4>
                    <p>CDRs Found: ${endpoint.CDRCount || 0}</p>
                    <p>Query Time: ${endpoint.QueryTime || '0.00'}s</p>
                    ${endpoint.Error ? `<p style="color: red;">Error: ${endpoint.Error}</p>` : ''}
                `;
                grid.appendChild(card);
            });
            
            resultsContainer.appendChild(grid);
        }
    },
    
    export(format, sessionId) {
        // In production, this would trigger a download
        // For now, just show a message
        this.showMessage(`Export to ${format.toUpperCase()} will be implemented soon`, 'info');
        console.log(`Export ${format} for session: ${sessionId}`);
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    app.init();
});