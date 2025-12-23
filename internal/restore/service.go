package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/veeam/powerbi-backup-go/internal/api"
	"github.com/veeam/powerbi-backup-go/internal/logger"
	"github.com/veeam/powerbi-backup-go/internal/models"
	"github.com/veeam/powerbi-backup-go/internal/storage"
)

// Service orchestrates the restore of Power BI components
type Service struct {
	apiClient      *api.Client
	storageService *storage.StorageService
}

// NewService creates a new restore service
func NewService(apiClient *api.Client, storageService *storage.StorageService) *Service {
	return &Service{
		apiClient:      apiClient,
		storageService: storageService,
	}
}

// RestoreWorkspace restores a workspace from backup
func (s *Service) RestoreWorkspace(ctx context.Context, targetWorkspaceID, backupPath string) error {
	logger.LogInfo(fmt.Sprintf("Starting restore for workspace: %s", targetWorkspaceID))
	logger.LogInfo(fmt.Sprintf("Restoring from backup: %s", backupPath))

	// Load backup
	backup, err := s.storageService.LoadBackup(backupPath)
	if err != nil {
		logger.LogError("Failed to load backup", err)
		return err
	}

	logger.LogInfo(fmt.Sprintf("Loaded backup from: %s", backup.Timestamp.Format("2006-01-02 15:04:05")))
	logger.LogInfo(fmt.Sprintf("Original workspace: %s (%s)", backup.WorkspaceName, backup.WorkspaceID))

	// Restore reports via PBIX files
	if err := s.restoreReportsPBIX(ctx, targetWorkspaceID, backupPath); err != nil {
		logger.LogError("Failed to restore reports", err)
		return err
	}

	// Restore refresh schedules
	if err := s.restoreRefreshSchedules(ctx, targetWorkspaceID, backup.RefreshSchedules); err != nil {
		logger.LogError("Failed to restore refresh schedules", err)
		// Don't fail entire restore if schedules fail
		logger.LogWarn("Continuing without refresh schedules")
	}

	logger.LogInfo("✅ Workspace restore completed successfully")
	return nil
}

// restoreReportsPBIX restores reports by importing PBIX files
func (s *Service) restoreReportsPBIX(ctx context.Context, workspaceID, backupPath string) error {
	logger.LogInfo("📄 Starting PBIX restoration...")

	// Find PBIX directory
	pbixDir := filepath.Join(backupPath, "pbix")
	if _, err := os.Stat(pbixDir); os.IsNotExist(err) {
		logger.LogWarn("No PBIX directory found in backup")
		return nil
	}

	// Get all PBIX files
	files, err := filepath.Glob(filepath.Join(pbixDir, "*.pbix"))
	if err != nil {
		logger.LogError("Failed to find PBIX files", err)
		return err
	}

	if len(files) == 0 {
		logger.LogWarn("No PBIX files found to restore")
		return nil
	}

	logger.LogInfo(fmt.Sprintf("Found %d PBIX files to restore", len(files)))

	// Get existing datasets to detect duplicates
	existingDatasets := make(map[string]bool)
	datasetsResp, err := s.apiClient.GetDatasets(ctx, workspaceID)
	if err == nil {
		if value, ok := datasetsResp["value"].([]interface{}); ok {
			for _, item := range value {
				if ds, ok := item.(map[string]interface{}); ok {
					if name, ok := ds["name"].(string); ok {
						existingDatasets[name] = true
					}
				}
			}
		}
	}

	// Import each PBIX file
	imported := 0
	failed := 0

	for _, pbixFile := range files {
		fileName := filepath.Base(pbixFile)
		datasetName := fileName[:len(fileName)-5] // Remove .pbix extension

		logger.LogInfo(fmt.Sprintf("📥 Importing: %s", fileName))

		// Check for duplicates
		finalName := datasetName
		if existingDatasets[datasetName] {
			counter := 1
			for existingDatasets[fmt.Sprintf("%s_%d", datasetName, counter)] {
				counter++
			}
			finalName = fmt.Sprintf("%s_%d", datasetName, counter)
			logger.LogInfo(fmt.Sprintf("⚠️  Duplicate detected - renaming to: %s", finalName))
		}

		// Import PBIX
		success, err := s.apiClient.ImportPBIX(ctx, workspaceID, pbixFile, finalName)
		if err != nil || !success {
			logger.LogError(fmt.Sprintf("❌ Failed to import: %s", fileName), err)
			failed++
			continue
		}

		logger.LogInfo(fmt.Sprintf("✅ Import queued successfully: %s", finalName))
		existingDatasets[finalName] = true
		imported++
	}

	logger.LogInfo(fmt.Sprintf("PBIX restoration complete: %d imported, %d failed", imported, failed))
	return nil
}

// restoreRefreshSchedules restores refresh schedules for datasets
func (s *Service) restoreRefreshSchedules(ctx context.Context, workspaceID string, schedules []models.RefreshSchedule) error {
	if len(schedules) == 0 {
		logger.LogInfo("No refresh schedules to restore")
		return nil
	}

	logger.LogInfo(fmt.Sprintf("Restoring %d refresh schedules...", len(schedules)))

	// Get current datasets in workspace
	datasetsResp, err := s.apiClient.GetDatasets(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to get datasets", err)
		return err
	}

	// Map dataset names to IDs
	datasetNameToID := make(map[string]string)
	if value, ok := datasetsResp["value"].([]interface{}); ok {
		for _, item := range value {
			if ds, ok := item.(map[string]interface{}); ok {
				name, _ := ds["name"].(string)
				id, _ := ds["id"].(string)
				if name != "" && id != "" {
					datasetNameToID[name] = id
				}
			}
		}
	}

	restored := 0
	failed := 0

	// Restore schedules
	for _, schedule := range schedules {
		logger.LogInfo(fmt.Sprintf("Restoring schedule for dataset: %s", schedule.DatasetName))

		// Find new dataset ID by name
		newDatasetID, exists := datasetNameToID[schedule.DatasetName]
		if !exists {
			logger.LogWarn(fmt.Sprintf("Dataset not found in target workspace: %s", schedule.DatasetName))
			failed++
			continue
		}

		// Update refresh schedule
		if err := s.apiClient.UpdateRefreshSchedule(ctx, workspaceID, newDatasetID, schedule.Schedule); err != nil {
			logger.LogError(fmt.Sprintf("Failed to restore schedule for: %s", schedule.DatasetName), err)
			failed++
			continue
		}

		logger.LogInfo(fmt.Sprintf("✅ Schedule restored: %s", schedule.DatasetName))
		restored++
	}

	logger.LogInfo(fmt.Sprintf("Refresh schedule restoration complete: %d restored, %d failed", restored, failed))
	return nil
}
