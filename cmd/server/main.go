package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/veeam/powerbi-backup-go/internal/api"
	"github.com/veeam/powerbi-backup-go/internal/auth"
	"github.com/veeam/powerbi-backup-go/internal/backup"
	"github.com/veeam/powerbi-backup-go/internal/config"
	"github.com/veeam/powerbi-backup-go/internal/logger"
	"github.com/veeam/powerbi-backup-go/internal/restore"
	"github.com/veeam/powerbi-backup-go/internal/storage"
)

// Server represents the web server with all dependencies
type Server struct {
	apiClient      *api.Client
	storageService *storage.StorageService
	authService    *auth.AuthService
	settings       *config.Settings
}

// Response types
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type BackupRequest struct {
	WorkspaceID string `json:"workspace_id"`
	All         bool   `json:"all"`
}

type RestoreRequest struct {
	WorkspaceID string `json:"workspace_id"`
	BackupPath  string `json:"backup_path"`
}

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type WorkspaceInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	IsOnPremium bool      `json:"is_on_premium"`
	Reports     int       `json:"reports"`
	Datasets    int       `json:"datasets"`
	Dashboards  int       `json:"dashboards"`
	Dataflows   int       `json:"dataflows"`
}

type BackupInfo struct {
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	Timestamp     time.Time `json:"timestamp"`
	Path          string    `json:"path"`
	Reports       int       `json:"reports"`
	Datasets      int       `json:"datasets"`
	Dashboards    int       `json:"dashboards"`
	Dataflows     int       `json:"dataflows"`
	Apps          int       `json:"apps"`
}

func main() {
	// Load configuration
	settings, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.InitLogger(settings.Debug)

	// Validate credentials
	if settings.PowerBIClientID == "" || settings.PowerBIClientSecret == "" || settings.PowerBITenantID == "" {
		logger.LogError("Missing required credentials. Please configure .env file", nil)
		os.Exit(1)
	}

	logger.LogInfo("🚀 Power BI Backup & Restore Web Server")
	logger.LogInfo("========================================")

	// Create services
	authService := auth.NewAuthService(settings)
	apiClient := api.NewClient(authService, settings)
	storageService := storage.NewStorageService(settings.BackupPath)

	server := &Server{
		apiClient:      apiClient,
		storageService: storageService,
		authService:    authService,
		settings:       settings,
	}

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/workspaces", server.handleWorkspaces)
	mux.HandleFunc("/api/workspace/create", server.handleCreateWorkspace)
	mux.HandleFunc("/api/backup", server.handleBackup)
	mux.HandleFunc("/api/restore", server.handleRestore)
	mux.HandleFunc("/api/backups", server.handleListBackups)

	// Static files
	webDir := filepath.Join(".", "web", "static")
	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	// CORS middleware wrapper
	handler := corsMiddleware(mux)

	// Start server
	port := "8060"
	addr := fmt.Sprintf("0.0.0.0:%s", port)

	logger.LogInfo(fmt.Sprintf("🌐 Web UI:        http://localhost:%s", port))
	logger.LogInfo(fmt.Sprintf("🔗 API Endpoint:  http://localhost:%s/api", port))
	logger.LogInfo(fmt.Sprintf("📁 Static Files:  %s", webDir))
	logger.LogInfo("")
	logger.LogInfo("Press Ctrl+C to stop the server")
	logger.LogInfo("")

	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.LogError("Server failed", err)
		os.Exit(1)
	}
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Message: "Power BI Backup Service is running",
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	s.sendJSON(w, http.StatusOK, response)
}

// List workspaces handler
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx := context.Background()
	workspaces, err := s.apiClient.GetWorkspaces(ctx)
	if err != nil {
		logger.LogError("Failed to fetch workspaces", err)
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch workspaces: %v", err))
		return
	}

	// Extract workspace list from response
	value, ok := workspaces["value"].([]interface{})
	if !ok {
		s.sendError(w, http.StatusInternalServerError, "Invalid workspace response format")
		return
	}

	// Format workspace info
	var workspaceList []WorkspaceInfo
	for _, ws := range value {
		wsMap, ok := ws.(map[string]interface{})
		if !ok {
			continue
		}

		wsInfo := WorkspaceInfo{
			ID:   getString(wsMap, "id"),
			Name: getString(wsMap, "name"),
			Type: getString(wsMap, "type"),
		}

		if isOnDedicatedCapacity, ok := wsMap["isOnDedicatedCapacity"].(bool); ok {
			wsInfo.IsOnPremium = isOnDedicatedCapacity
		}

		workspaceList = append(workspaceList, wsInfo)
	}

	response := APIResponse{
		Success: true,
		Data:    workspaceList,
	}
	s.sendJSON(w, http.StatusOK, response)
}

// Create workspace handler
func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate workspace name
	if req.Name == "" {
		s.sendError(w, http.StatusBadRequest, "Workspace name is required")
		return
	}

	ctx := context.Background()

	// Create workspace using Power BI API
	// POST https://api.powerbi.com/v1.0/myorg/groups
	createPayload := map[string]interface{}{
		"name": req.Name,
	}
	if req.Description != "" {
		createPayload["description"] = req.Description
	}

	result, err := s.apiClient.CreateWorkspace(ctx, createPayload)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create workspace: %v", err))
		return
	}

	// Extract workspace info from result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		s.sendError(w, http.StatusInternalServerError, "Invalid response from Power BI API")
		return
	}

	workspaceInfo := WorkspaceInfo{
		ID:   getString(resultMap, "id"),
		Name: getString(resultMap, "name"),
		Type: getString(resultMap, "type"),
	}

	if isOnDedicatedCapacity, ok := resultMap["isOnDedicatedCapacity"].(bool); ok {
		workspaceInfo.IsOnPremium = isOnDedicatedCapacity
	}

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("Workspace '%s' created successfully", req.Name),
		Data:    workspaceInfo,
	}
	s.sendJSON(w, http.StatusCreated, response)
}

// Backup handler
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := context.Background()

	if req.All {
		// Backup all workspaces (async)
		go s.backupAllWorkspaces(ctx)
		
		response := APIResponse{
			Success: true,
			Message: "Backup of all workspaces started",
			Data: map[string]interface{}{
				"status":    "started",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}
		s.sendJSON(w, http.StatusAccepted, response)
	} else if req.WorkspaceID != "" {
		// Backup single workspace (async)
		go s.backupWorkspace(ctx, req.WorkspaceID)

		response := APIResponse{
			Success: true,
			Message: fmt.Sprintf("Backup started for workspace: %s", req.WorkspaceID),
			Data: map[string]interface{}{
				"workspace_id": req.WorkspaceID,
				"status":       "started",
				"timestamp":    time.Now().Format(time.RFC3339),
			},
		}
		s.sendJSON(w, http.StatusAccepted, response)
	} else {
		s.sendError(w, http.StatusBadRequest, "workspace_id or all flag required")
	}
}

// Restore handler
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.WorkspaceID == "" || req.BackupPath == "" {
		s.sendError(w, http.StatusBadRequest, "workspace_id and backup_path required")
		return
	}

	ctx := context.Background()

	// Restore workspace (async)
	go s.restoreWorkspace(ctx, req.WorkspaceID, req.BackupPath)

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("Restore started for workspace: %s", req.WorkspaceID),
		Data: map[string]interface{}{
			"workspace_id": req.WorkspaceID,
			"backup_path":  req.BackupPath,
			"status":       "started",
			"timestamp":    time.Now().Format(time.RFC3339),
		},
	}
	s.sendJSON(w, http.StatusAccepted, response)
}

// List backups handler
func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	backupDir := s.settings.BackupPath
	backups := []BackupInfo{} // Initialize as empty array instead of nil

	err := filepath.WalkDir(backupDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check if this is a backup directory (contains complete_backup.json)
		backupFile := filepath.Join(path, "complete_backup.json")
		if _, statErr := os.Stat(backupFile); statErr == nil {
			// Parse backup info
			data, err := os.ReadFile(backupFile)
			if err == nil {
				var backupData map[string]interface{}
				if err := json.Unmarshal(data, &backupData); err == nil {
					info := BackupInfo{
						Path: path,
					}

					// Handle both workspaceId and workspace_id formats
					if wsID, ok := backupData["workspaceId"].(string); ok {
						info.WorkspaceID = wsID
					} else if wsID, ok := backupData["workspace_id"].(string); ok {
						info.WorkspaceID = wsID
					}
					
					// Handle both workspaceName and workspace_name formats
					if wsName, ok := backupData["workspaceName"].(string); ok {
						info.WorkspaceName = wsName
					} else if wsName, ok := backupData["workspace_name"].(string); ok {
						info.WorkspaceName = wsName
					}
					
					if timestamp, ok := backupData["timestamp"].(string); ok {
						if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
							info.Timestamp = t
						}
					}
					if reports, ok := backupData["reports"].([]interface{}); ok {
						info.Reports = len(reports)
					}
					if datasets, ok := backupData["datasets"].([]interface{}); ok {
						info.Datasets = len(datasets)
					}
					if dashboards, ok := backupData["dashboards"].([]interface{}); ok {
						info.Dashboards = len(dashboards)
					}
					if dataflows, ok := backupData["dataflows"].([]interface{}); ok {
						info.Dataflows = len(dataflows)
					}
					if apps, ok := backupData["apps"].([]interface{}); ok {
						info.Apps = len(apps)
					}

					backups = append(backups, info)
				}
			}
		}
		return nil
	})

	if err != nil {
		logger.LogError("Failed to list backups", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	response := APIResponse{
		Success: true,
		Data:    backups,
	}
	s.sendJSON(w, http.StatusOK, response)
}

// Background backup operations
func (s *Server) backupWorkspace(ctx context.Context, workspaceID string) {
	logger.LogInfo(fmt.Sprintf("Starting backup for workspace: %s", workspaceID))
	start := time.Now()

	backupService := backup.NewService(s.apiClient, s.storageService)
	backupData, err := backupService.BackupWorkspace(ctx, workspaceID)
	if err != nil {
		logger.LogError(fmt.Sprintf("Backup failed for workspace %s", workspaceID), err)
		return
	}

	duration := time.Since(start)
	logger.LogInfo(fmt.Sprintf("✅ Backup completed in %v", duration))
	logger.LogInfo(fmt.Sprintf("📊 Summary: Reports: %d, Datasets: %d, Dashboards: %d, Dataflows: %d, Apps: %d",
		len(backupData.Reports), len(backupData.Datasets), len(backupData.Dashboards),
		len(backupData.Dataflows), len(backupData.Apps)))
}

func (s *Server) backupAllWorkspaces(ctx context.Context) {
	logger.LogInfo("Fetching all workspaces...")

	workspaces, err := s.apiClient.GetWorkspaces(ctx)
	if err != nil {
		logger.LogError("Failed to fetch workspaces", err)
		return
	}

	value, ok := workspaces["value"].([]interface{})
	if !ok {
		logger.LogError("Invalid workspace response format", nil)
		return
	}

	logger.LogInfo(fmt.Sprintf("Found %d workspaces", len(value)))

	backupService := backup.NewService(s.apiClient, s.storageService)
	successCount := 0
	failCount := 0

	for i, ws := range value {
		wsMap, ok := ws.(map[string]interface{})
		if !ok {
			continue
		}

		wsID := getString(wsMap, "id")
		wsName := getString(wsMap, "name")

		logger.LogInfo(fmt.Sprintf("[%d/%d] Backing up workspace: %s (%s)", i+1, len(value), wsName, wsID))

		_, err := backupService.BackupWorkspace(ctx, wsID)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to backup workspace: %s", wsName), err)
			failCount++
		} else {
			successCount++
		}
	}

	logger.LogInfo(fmt.Sprintf("✅ All workspaces backup completed: %d succeeded, %d failed", successCount, failCount))
}

func (s *Server) restoreWorkspace(ctx context.Context, workspaceID, backupPath string) {
	logger.LogInfo(fmt.Sprintf("Starting restore for workspace: %s from %s", workspaceID, backupPath))
	start := time.Now()

	restoreService := restore.NewService(s.apiClient, s.storageService)
	err := restoreService.RestoreWorkspace(ctx, workspaceID, backupPath)
	if err != nil {
		logger.LogError(fmt.Sprintf("Restore failed for workspace %s", workspaceID), err)
		return
	}

	duration := time.Since(start)
	logger.LogInfo(fmt.Sprintf("✅ Restore completed in %v", duration))
}

// Helper functions
func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	s.sendJSON(w, status, response)
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
