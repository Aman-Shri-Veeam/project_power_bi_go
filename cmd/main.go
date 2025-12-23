package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/veeam/powerbi-backup-go/internal/api"
	"github.com/veeam/powerbi-backup-go/internal/auth"
	"github.com/veeam/powerbi-backup-go/internal/backup"
	"github.com/veeam/powerbi-backup-go/internal/config"
	"github.com/veeam/powerbi-backup-go/internal/logger"
	"github.com/veeam/powerbi-backup-go/internal/restore"
	"github.com/veeam/powerbi-backup-go/internal/storage"
)

func main() {
	// Define command-line flags
	cmd := flag.String("cmd", "backup", "Command to execute: backup or restore")
	workspaceID := flag.String("workspace-id", "", "Power BI workspace ID")
	backupPathArg := flag.String("backup-path", "", "Path to backup for restore operation")
	allWorkspaces := flag.Bool("all", false, "Backup all workspaces")
	flag.Parse()

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

	logger.LogInfo("🚀 Power BI Backup & Restore Tool")
	logger.LogInfo("=" + string(make([]byte, 50)))

	// Create services
	authService := auth.NewAuthService(settings)
	apiClient := api.NewClient(authService, settings)
	storageService := storage.NewStorageService(settings.BackupPath)

	ctx := context.Background()

	// Execute command
	switch *cmd {
	case "backup":
		if *allWorkspaces {
			backupAllWorkspaces(ctx, apiClient, storageService)
		} else if *workspaceID != "" {
			backupWorkspace(ctx, *workspaceID, apiClient, storageService)
		} else {
			logger.LogError("Please provide --workspace-id or use --all flag", nil)
			flag.Usage()
			os.Exit(1)
		}

	case "restore":
		if *workspaceID == "" || *backupPathArg == "" {
			logger.LogError("Restore requires --workspace-id and --backup-path", nil)
			flag.Usage()
			os.Exit(1)
		}
		restoreWorkspace(ctx, *workspaceID, *backupPathArg, apiClient, storageService)

	default:
		logger.LogError(fmt.Sprintf("Unknown command: %s", *cmd), nil)
		flag.Usage()
		os.Exit(1)
	}
}

func backupWorkspace(ctx context.Context, workspaceID string, apiClient *api.Client, storageService *storage.StorageService) {
	logger.LogInfo(fmt.Sprintf("Starting backup for workspace: %s", workspaceID))

	backupService := backup.NewService(apiClient, storageService)

	startTime := time.Now()
	backupData, err := backupService.BackupWorkspace(ctx, workspaceID)
	if err != nil {
		logger.LogError("Backup failed", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	logger.LogInfo(fmt.Sprintf("✅ Backup completed in %v", duration))
	logger.LogInfo(fmt.Sprintf("📊 Summary:"))
	logger.LogInfo(fmt.Sprintf("   - Reports: %d", len(backupData.Reports)))
	logger.LogInfo(fmt.Sprintf("   - Datasets: %d", len(backupData.Datasets)))
	logger.LogInfo(fmt.Sprintf("   - Dataflows: %d", len(backupData.Dataflows)))
	logger.LogInfo(fmt.Sprintf("   - Dashboards: %d", len(backupData.Dashboards)))
	logger.LogInfo(fmt.Sprintf("   - Apps: %d", len(backupData.Apps)))
	logger.LogInfo(fmt.Sprintf("   - Refresh Schedules: %d", len(backupData.RefreshSchedules)))
}

func backupAllWorkspaces(ctx context.Context, apiClient *api.Client, storageService *storage.StorageService) {
	logger.LogInfo("Fetching all workspaces...")

	workspacesResp, err := apiClient.GetWorkspaces(ctx)
	if err != nil {
		logger.LogError("Failed to fetch workspaces", err)
		os.Exit(1)
	}

	value, ok := workspacesResp["value"].([]interface{})
	if !ok || len(value) == 0 {
		logger.LogWarn("No workspaces found")
		return
	}

	logger.LogInfo(fmt.Sprintf("Found %d workspaces", len(value)))

	backupService := backup.NewService(apiClient, storageService)
	successCount := 0
	failCount := 0

	for i, item := range value {
		workspaceMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		wsID, _ := workspaceMap["id"].(string)
		wsName, _ := workspaceMap["name"].(string)

		logger.LogInfo(fmt.Sprintf("[%d/%d] Backing up workspace: %s (%s)", i+1, len(value), wsName, wsID))

		_, err := backupService.BackupWorkspace(ctx, wsID)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to backup workspace: %s", wsName), err)
			failCount++
			continue
		}

		successCount++
	}

	logger.LogInfo(fmt.Sprintf("✅ All workspaces backup completed: %d succeeded, %d failed", successCount, failCount))
}

func restoreWorkspace(ctx context.Context, workspaceID, backupPath string, apiClient *api.Client, storageService *storage.StorageService) {
	logger.LogInfo(fmt.Sprintf("Starting restore for workspace: %s", workspaceID))
	logger.LogInfo(fmt.Sprintf("From backup: %s", backupPath))

	restoreService := restore.NewService(apiClient, storageService)

	startTime := time.Now()
	err := restoreService.RestoreWorkspace(ctx, workspaceID, backupPath)
	if err != nil {
		logger.LogError("Restore failed", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	logger.LogInfo(fmt.Sprintf("✅ Restore completed in %v", duration))
}
