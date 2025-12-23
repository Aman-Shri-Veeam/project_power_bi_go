// API Base URL
const API_BASE = window.location.origin + '/api';

// State
let workspaces = [];
let backups = [];

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeTabs();
    checkServerStatus();
    setupEventListeners();
});

// Tab Navigation
function initializeTabs() {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tabName = button.getAttribute('data-tab');
            
            // Update active states
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));
            
            button.classList.add('active');
            document.getElementById(`${tabName}-tab`).classList.add('active');
        });
    });
}

// Event Listeners
function setupEventListeners() {
    // Backup form
    document.getElementById('loadWorkspacesBtn').addEventListener('click', loadWorkspaces);
    document.getElementById('backupForm').addEventListener('submit', handleBackup);
    document.getElementById('workspaceSelect').addEventListener('change', handleWorkspaceSelect);
    
    // Restore form
    document.getElementById('loadBackupsBtn').addEventListener('click', loadBackups);
    document.getElementById('restoreForm').addEventListener('submit', handleRestore);
    
    // History
    document.getElementById('refreshHistoryBtn').addEventListener('click', loadHistory);
}

// Server Status Check
async function checkServerStatus() {
    const statusDot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');
    
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        
        if (data.success) {
            statusDot.className = 'status-dot status-online';
            statusText.textContent = 'Online';
        } else {
            statusDot.className = 'status-dot status-offline';
            statusText.textContent = 'Offline';
        }
    } catch (error) {
        statusDot.className = 'status-dot status-offline';
        statusText.textContent = 'Offline';
    }
}

// Load Workspaces
async function loadWorkspaces() {
    const button = document.getElementById('loadWorkspacesBtn');
    const spinner = document.getElementById('loadingSpinner');
    const select = document.getElementById('workspaceSelect');
    
    button.disabled = true;
    spinner.style.display = 'inline';
    
    try {
        const response = await fetch(`${API_BASE}/workspaces`);
        const data = await response.json();
        
        if (data.success) {
            workspaces = data.data;
            
            // Populate select
            select.innerHTML = '<option value="">-- Select a workspace --</option>';
            workspaces.forEach(ws => {
                const option = document.createElement('option');
                option.value = ws.id;
                option.textContent = `${ws.name} ${ws.is_on_premium ? '⭐ Premium' : ''}`;
                select.appendChild(option);
            });
            
            showNotification('✅ Workspaces loaded successfully', 'success');
        } else {
            showNotification('❌ Failed to load workspaces', 'error');
        }
    } catch (error) {
        console.error('Error loading workspaces:', error);
        showNotification('❌ Error loading workspaces', 'error');
    } finally {
        button.disabled = false;
        spinner.style.display = 'none';
    }
}

// Handle Workspace Selection
function handleWorkspaceSelect(event) {
    const workspaceId = event.target.value;
    if (workspaceId) {
        document.getElementById('workspaceId').value = workspaceId;
    }
}

// Handle Backup
async function handleBackup(event) {
    event.preventDefault();
    
    const workspaceId = document.getElementById('workspaceId').value;
    const backupAll = document.getElementById('backupAll').checked;
    
    if (!backupAll && !workspaceId) {
        showNotification('⚠️ Please enter a workspace ID or select "Backup All"', 'warning');
        return;
    }
    
    // Hide previous results
    document.getElementById('backupResult').style.display = 'none';
    document.getElementById('backupError').style.display = 'none';
    
    try {
        const response = await fetch(`${API_BASE}/backup`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                workspace_id: workspaceId,
                all: backupAll
            })
        });
        
        const data = await response.json();
        
        if (data.success) {
            document.getElementById('resultWorkspaceId').textContent = backupAll ? 'All Workspaces' : workspaceId;
            document.getElementById('resultStatus').textContent = 'Running';
            document.getElementById('resultTimestamp').textContent = new Date().toLocaleString();
            document.getElementById('backupResult').style.display = 'block';
            
            showNotification('✅ Backup started successfully', 'success');
        } else {
            document.getElementById('errorMessage').textContent = data.error || 'Unknown error';
            document.getElementById('backupError').style.display = 'block';
            
            showNotification('❌ Backup failed', 'error');
        }
    } catch (error) {
        console.error('Error starting backup:', error);
        document.getElementById('errorMessage').textContent = error.message;
        document.getElementById('backupError').style.display = 'block';
        
        showNotification('❌ Error starting backup', 'error');
    }
}

// Load Backups
async function loadBackups() {
    const button = document.getElementById('loadBackupsBtn');
    const select = document.getElementById('restoreBackupSelect');
    
    button.disabled = true;
    
    try {
        const response = await fetch(`${API_BASE}/backups`);
        const data = await response.json();
        
        if (data.success) {
            backups = data.data || [];
            
            // Populate select
            select.innerHTML = '<option value="">-- Select a backup --</option>';
            
            if (backups.length === 0) {
                const option = document.createElement('option');
                option.value = '';
                option.textContent = '-- No backups found --';
                select.appendChild(option);
                showNotification('ℹ️ No backups found', 'info');
            } else {
                backups.forEach(backup => {
                    const option = document.createElement('option');
                    option.value = backup.path;
                    const date = new Date(backup.timestamp).toLocaleString();
                    option.textContent = `${backup.workspace_name} - ${date}`;
                    select.appendChild(option);
                });
                showNotification('✅ Backups loaded successfully', 'success');
            }
        } else {
            showNotification('❌ Failed to load backups', 'error');
        }
    } catch (error) {
        console.error('Error loading backups:', error);
        showNotification('❌ Error loading backups', 'error');
    } finally {
        button.disabled = false;
    }
}

// Handle Restore
async function handleRestore(event) {
    event.preventDefault();
    
    const workspaceId = document.getElementById('restoreWorkspaceId').value;
    const backupPath = document.getElementById('restoreBackupSelect').value;
    
    if (!workspaceId || !backupPath) {
        showNotification('⚠️ Please select a backup and enter a workspace ID', 'warning');
        return;
    }
    
    // Hide previous results
    document.getElementById('restoreResult').style.display = 'none';
    document.getElementById('restoreError').style.display = 'none';
    
    try {
        const response = await fetch(`${API_BASE}/restore`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                workspace_id: workspaceId,
                backup_path: backupPath
            })
        });
        
        const data = await response.json();
        
        if (data.success) {
            document.getElementById('restoreResultWorkspaceId').textContent = workspaceId;
            document.getElementById('restoreResultBackupPath').textContent = backupPath;
            document.getElementById('restoreResultStatus').textContent = 'Running';
            document.getElementById('restoreResultTimestamp').textContent = new Date().toLocaleString();
            document.getElementById('restoreResult').style.display = 'block';
            
            showNotification('✅ Restore started successfully', 'success');
        } else {
            document.getElementById('restoreErrorMessage').textContent = data.error || 'Unknown error';
            document.getElementById('restoreError').style.display = 'block';
            
            showNotification('❌ Restore failed', 'error');
        }
    } catch (error) {
        console.error('Error starting restore:', error);
        document.getElementById('restoreErrorMessage').textContent = error.message;
        document.getElementById('restoreError').style.display = 'block';
        
        showNotification('❌ Error starting restore', 'error');
    }
}

// Load History
async function loadHistory() {
    const container = document.getElementById('historyContainer');
    container.innerHTML = '<p class="loading">Loading backups...</p>';
    
    try {
        const response = await fetch(`${API_BASE}/backups`);
        const data = await response.json();
        
        if (data.success) {
            const backupsList = data.data || [];
            
            if (backupsList.length === 0) {
                container.innerHTML = '<p class="no-data">No backups found</p>';
                return;
            }
            
            let html = '<div class="history-list">';
            backupsList.forEach(backup => {
                const date = new Date(backup.timestamp).toLocaleString();
                const workspaceName = backup.workspace_name || 'Unknown Workspace';
                html += `
                    <div class="history-item">
                        <div class="history-header">
                            <h4>${workspaceName}</h4>
                            <span class="history-date">${date}</span>
                        </div>
                        <div class="history-details">
                            <span>📄 Reports: ${backup.reports || 0}</span>
                            <span>📊 Datasets: ${backup.datasets || 0}</span>
                            <span>📈 Dashboards: ${backup.dashboards || 0}</span>
                            <span>🌊 Dataflows: ${backup.dataflows || 0}</span>
                            <span>📱 Apps: ${backup.apps || 0}</span>
                        </div>
                        <div class="history-path">
                            <small>Path: ${backup.path}</small>
                        </div>
                    </div>
                `;
            });
            html += '</div>';
            
            container.innerHTML = html;
        } else {
            container.innerHTML = '<p class="error">Failed to load backups</p>';
        }
    } catch (error) {
        console.error('Error loading history:', error);
        container.innerHTML = '<p class="error">Error loading backups</p>';
    }
}

// Show Notification
function showNotification(message, type = 'info') {
    const notification = document.getElementById('notification');
    const notificationText = document.getElementById('notificationText');
    
    notificationText.textContent = message;
    notification.className = `toast toast-${type}`;
    notification.style.display = 'block';
    
    setTimeout(() => {
        notification.style.display = 'none';
    }, 3000);
}
