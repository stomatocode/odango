// O Dan Go - Minimal Functional JavaScript

const app = {
    currentView: 'welcome',
    
    init() {
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
        
        // Basic validation
        const hasSearchCriteria = Array.from(formData.values()).some(value => value.trim() !== '');
        if (!hasSearchCriteria) {
            this.showMessage('Please enter at least one search criterion', 'error');
            return;
        }
        
        // Show loading
        this.showLoading(true);
        
        // Submit form (using traditional form submission for now)
        // In production, this would be an AJAX call
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