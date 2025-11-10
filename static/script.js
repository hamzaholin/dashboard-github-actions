let allJobs = [];
let filteredJobs = [];
let currentPage = 1;
let itemsPerPage = 25;
let autoRefreshInterval = null;
let isAutoRefreshEnabled = false;
let organizations = [];

// Fetch data from API
async function fetchDashboardData() {
    try {
        // Show loading state
        document.getElementById('jobsTableBody').innerHTML = 
            '<tr><td colspan="8" class="loading">Loading...</td></tr>';
        
        // Get selected period
        const period = document.getElementById('periodFilter').value;
        
        // Fetch with period parameter
        const response = await fetch(`/api/dashboard?period=${period}`);
        if (!response.ok) {
            throw new Error('Failed to fetch dashboard data');
        }
        const data = await response.json();
        
        allJobs = data.jobs || [];
        
        // Extract unique organizations and populate filter
        organizations = [...new Set(allJobs.map(job => job.organization).filter(org => org))].sort();
        populateOrgFilter();
        
        // Sort by CreatedAt (newest first) - already sorted in backend, but ensure it
        allJobs.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
        
        updateStats(data.stats);
        updateRateLimit(data.rate_limit);
        applyFilters();
    } catch (error) {
        console.error('Error fetching dashboard data:', error);
        document.getElementById('jobsTableBody').innerHTML = 
            '<tr><td colspan="9" class="loading">Error loading data. Please try again.</td></tr>';
    }
}

// Populate organization filter dropdown
function populateOrgFilter() {
    const orgFilter = document.getElementById('orgFilter');
    const currentValue = orgFilter.value;
    
    // Clear existing options except "All Organizations"
    orgFilter.innerHTML = '<option value="all">All Organizations</option>';
    
    // Add organization options
    organizations.forEach(org => {
        const option = document.createElement('option');
        option.value = org;
        option.textContent = org;
        orgFilter.appendChild(option);
    });
    
    // Restore previous selection if it still exists
    if (currentValue && organizations.includes(currentValue)) {
        orgFilter.value = currentValue;
    }
}

// Update stats cards
function updateStats(stats) {
    document.getElementById('successCount').textContent = stats.success || 0;
    document.getElementById('failedCount').textContent = stats.failed || 0;
    document.getElementById('runningCount').textContent = stats.running || 0;
    document.getElementById('pendingCount').textContent = stats.pending || 0;
    document.getElementById('totalCount').textContent = stats.total || 0;
}

// Update rate limit info
function updateRateLimit(rateLimit) {
    if (!rateLimit) {
        return;
    }
    
    const remaining = rateLimit.remaining || 0;
    const limit = rateLimit.limit || 5000;
    const resetAt = rateLimit.reset_at;
    
    // Update rate limit value
    document.getElementById('rateLimitValue').textContent = `${remaining}/${limit}`;
    
    // Calculate percentage
    const percentage = (remaining / limit) * 100;
    
    // Add warning class if below 20%
    const rateLimitValue = document.getElementById('rateLimitValue');
    rateLimitValue.classList.remove('warning', 'critical');
    if (percentage < 10) {
        rateLimitValue.classList.add('critical');
    } else if (percentage < 20) {
        rateLimitValue.classList.add('warning');
    }
    
    // Update reset time
    if (resetAt) {
        const resetDate = new Date(resetAt);
        const now = new Date();
        const diffMs = resetDate - now;
        const diffMins = Math.floor(diffMs / 60000);
        
        let resetText = '';
        if (diffMins < 1) {
            resetText = 'Resets soon';
        } else if (diffMins < 60) {
            resetText = `Resets in ${diffMins}m`;
        } else {
            const diffHours = Math.floor(diffMins / 60);
            const remainingMins = diffMins % 60;
            resetText = `Resets in ${diffHours}h ${remainingMins}m`;
        }
        
        document.getElementById('rateLimitReset').textContent = resetText;
    } else {
        document.getElementById('rateLimitReset').textContent = '';
    }
}

// Apply filters and search
function applyFilters() {
    const orgFilter = document.getElementById('orgFilter').value;
    const statusFilter = document.getElementById('statusFilter').value;
    const searchQuery = document.getElementById('searchInput').value.toLowerCase();
    
    filteredJobs = allJobs.filter(job => {
        const matchesOrg = orgFilter === 'all' || job.organization === orgFilter;
        const matchesStatus = statusFilter === 'all' || job.status === statusFilter;
        const matchesSearch = job.name.toLowerCase().includes(searchQuery) || 
                             job.id.toLowerCase().includes(searchQuery) ||
                             job.pipeline.toLowerCase().includes(searchQuery) ||
                             (job.organization && job.organization.toLowerCase().includes(searchQuery));
        return matchesOrg && matchesStatus && matchesSearch;
    });
    
    // Ensure sorted by newest first (by CreatedAt)
    filteredJobs.sort((a, b) => {
        const dateA = new Date(a.created_at || 0);
        const dateB = new Date(b.created_at || 0);
        return dateB - dateA; // Newest first
    });
    
    // Update stats based on filtered jobs
    updateFilteredStats(filteredJobs);
    
    currentPage = 1;
    renderTable();
    renderPagination();
}

// Update stats based on filtered jobs
function updateFilteredStats(jobs) {
    const stats = {
        success: 0,
        failed: 0,
        running: 0,
        pending: 0,
        total: jobs.length
    };
    
    jobs.forEach(job => {
        switch(job.status) {
            case 'success':
                stats.success++;
                break;
            case 'failed':
                stats.failed++;
                break;
            case 'running':
                stats.running++;
                break;
            case 'pending':
                stats.pending++;
                break;
        }
    });
    
    updateStats(stats);
}

// Render jobs table
function renderTable() {
    const tbody = document.getElementById('jobsTableBody');
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    const jobsToShow = filteredJobs.slice(startIndex, endIndex);
    
    if (jobsToShow.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" class="loading">No jobs found</td></tr>';
        return;
    }
    
    tbody.innerHTML = jobsToShow.map(job => `
        <tr>
            <td>${job.id}</td>
            <td>${escapeHtml(job.name)}</td>
            <td><span class="status-badge ${job.status}">${job.status}</span></td>
            <td>${escapeHtml(job.pipeline)}</td>
            <td>${escapeHtml(job.branch)}</td>
            <td>${job.duration}</td>
            <td>${job.started}</td>
            <td>${escapeHtml(job.organization || 'N/A')}</td>
            <td>
                <a href="${job.html_url || '#'}" target="_blank" class="btn-link" title="View on GitHub Actions">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                        <polyline points="15 3 21 3 21 9"></polyline>
                        <line x1="10" y1="14" x2="21" y2="3"></line>
                    </svg>
                </a>
            </td>
        </tr>
    `).join('');
}

// Render pagination
function renderPagination() {
    const totalPages = Math.ceil(filteredJobs.length / itemsPerPage);
    const pagination = document.getElementById('pagination');
    
    if (totalPages <= 1) {
        pagination.innerHTML = '';
        return;
    }
    
    let html = '';
    
    // Previous button
    html += `<button onclick="changePage(${currentPage - 1})" ${currentPage === 1 ? 'disabled' : ''}>Previous</button>`;
    
    // Page numbers
    for (let i = 1; i <= totalPages; i++) {
        if (i === 1 || i === totalPages || (i >= currentPage - 2 && i <= currentPage + 2)) {
            html += `<button onclick="changePage(${i})" class="${i === currentPage ? 'active' : ''}">${i}</button>`;
        } else if (i === currentPage - 3 || i === currentPage + 3) {
            html += `<button disabled>...</button>`;
        }
    }
    
    // Next button
    html += `<button onclick="changePage(${currentPage + 1})" ${currentPage === totalPages ? 'disabled' : ''}>Next</button>`;
    
    pagination.innerHTML = html;
}

// Change page
function changePage(page) {
    const totalPages = Math.ceil(filteredJobs.length / itemsPerPage);
    if (page >= 1 && page <= totalPages) {
        currentPage = page;
        renderTable();
        renderPagination();
        window.scrollTo({ top: 0, behavior: 'smooth' });
    }
}

// View job details (placeholder - can be extended)
function viewJob(runId, jobId, organization) {
    alert(`Viewing job: ${jobId}\nRun ID: ${runId}\nOrganization: ${organization}\n\nThis can be extended to show detailed job information or link to GitHub Actions.`);
    // You can extend this to open a modal or navigate to GitHub Actions page
    // window.open(`https://github.com/${organization}/actions/runs/${runId}`, '_blank');
}

// Toggle auto refresh
function toggleAutoRefresh() {
    const btn = document.getElementById('autoRefreshBtn');
    
    if (isAutoRefreshEnabled) {
        clearInterval(autoRefreshInterval);
        isAutoRefreshEnabled = false;
        btn.textContent = 'Auto Refresh: OFF';
        btn.classList.remove('active');
    } else {
        autoRefreshInterval = setInterval(fetchDashboardData, 30000); // Refresh every 30 seconds
        isAutoRefreshEnabled = true;
        btn.textContent = 'Auto Refresh: ON';
        btn.classList.add('active');
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Event listeners
document.addEventListener('DOMContentLoaded', () => {
    fetchDashboardData();
    
    document.getElementById('refreshBtn').addEventListener('click', fetchDashboardData);
    document.getElementById('autoRefreshBtn').addEventListener('click', toggleAutoRefresh);
    document.getElementById('periodFilter').addEventListener('change', fetchDashboardData);
    document.getElementById('orgFilter').addEventListener('change', applyFilters);
    document.getElementById('statusFilter').addEventListener('change', applyFilters);
    document.getElementById('searchInput').addEventListener('input', applyFilters);
    document.getElementById('itemsPerPage').addEventListener('change', (e) => {
        itemsPerPage = parseInt(e.target.value);
        currentPage = 1;
        applyFilters();
    });
});

