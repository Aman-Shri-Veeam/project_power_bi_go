package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/veeam/powerbi-backup-go/internal/logger"
	"github.com/veeam/powerbi-backup-go/internal/models"
)

// StorageService handles backup storage operations
type StorageService struct {
	backupPath string
}

// NewStorageService creates a new storage service
func NewStorageService(backupPath string) *StorageService {
	return &StorageService{
		backupPath: backupPath,
	}
}

// GetBackupPath returns the backup path
func (s *StorageService) GetBackupPath() string {
	return s.backupPath
}

// SaveBackup saves a complete backup to the file system
func (s *StorageService) SaveBackup(backup *models.CompleteBackup) (string, error) {
	// Create workspace directory
	timestamp := backup.Timestamp.Format("2006-01-02_15-04-05")
	backupDir := filepath.Join(s.backupPath, backup.WorkspaceID, timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.LogError(fmt.Sprintf("Failed to create backup directory: %s", backupDir), err)
		return "", err
	}

	// Note: PBIX files are already created in backupDir by BackupWorkspace()
	// This function just saves the JSON metadata files alongside them

	// Save complete backup as JSON
	backupFile := filepath.Join(backupDir, "complete_backup.json")
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		logger.LogError("Failed to marshal backup data", err)
		return "", err
	}

	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		logger.LogError(fmt.Sprintf("Failed to write backup file: %s", backupFile), err)
		return "", err
	}

	// Save individual components
	s.saveComponent(backupDir, "reports.json", backup.Reports)
	s.saveComponent(backupDir, "datasets.json", backup.Datasets)
	s.saveComponent(backupDir, "dataflows.json", backup.Dataflows)
	s.saveComponent(backupDir, "dashboards.json", backup.Dashboards)
	s.saveComponent(backupDir, "apps.json", backup.Apps)
	s.saveComponent(backupDir, "refresh_schedules.json", backup.RefreshSchedules)
	s.saveComponent(backupDir, "workspace_settings.json", backup.WorkspaceSettings)

	logger.LogInfo(fmt.Sprintf("Backup saved successfully: %s", backupDir))
	return backupDir, nil
}

// saveComponent saves a component to a JSON file
func (s *StorageService) saveComponent(dir, filename string, data interface{}) error {
	filePath := filepath.Join(dir, filename)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to marshal %s", filename), err)
		return err
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		logger.LogError(fmt.Sprintf("Failed to write %s", filename), err)
		return err
	}

	return nil
}

// LoadBackup loads a backup from the file system
func (s *StorageService) LoadBackup(backupPath string) (*models.CompleteBackup, error) {
	backupFile := filepath.Join(backupPath, "complete_backup.json")

	data, err := os.ReadFile(backupFile)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to read backup file: %s", backupFile), err)
		return nil, err
	}

	var backup models.CompleteBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		logger.LogError("Failed to unmarshal backup data", err)
		return nil, err
	}

	logger.LogInfo(fmt.Sprintf("Backup loaded successfully: %s", backupPath))
	return &backup, nil
}

// ListBackups lists all backups for a workspace
func (s *StorageService) ListBackups(workspaceID string) ([]string, error) {
	workspaceDir := filepath.Join(s.backupPath, workspaceID)

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to read workspace directory: %s", workspaceDir), err)
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, filepath.Join(workspaceDir, entry.Name()))
		}
	}

	return backups, nil
}

// GetLatestBackup gets the most recent backup for a workspace
func (s *StorageService) GetLatestBackup(workspaceID string) (string, error) {
	backups, err := s.ListBackups(workspaceID)
	if err != nil {
		return "", err
	}

	if len(backups) == 0 {
		return "", fmt.Errorf("no backups found for workspace: %s", workspaceID)
	}

	// Backups are sorted by timestamp in directory name
	latest := backups[len(backups)-1]
	return latest, nil
}

// CreateBackupMetadata creates a metadata file for the backup
func (s *StorageService) CreateBackupMetadata(backupDir string, metadata map[string]interface{}) error {
	metadata["created_at"] = time.Now().Format(time.RFC3339)
	
	metadataFile := filepath.Join(backupDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataFile, data, 0644)
}
