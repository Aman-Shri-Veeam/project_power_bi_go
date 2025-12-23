package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/veeam/powerbi-backup-go/internal/api"
	"github.com/veeam/powerbi-backup-go/internal/logger"
	"github.com/veeam/powerbi-backup-go/internal/models"
	"github.com/veeam/powerbi-backup-go/internal/storage"
)

// Service orchestrates the backup of all Power BI components
type Service struct {
	apiClient      *api.Client
	storageService *storage.StorageService
}

// NewService creates a new backup service
func NewService(apiClient *api.Client, storageService *storage.StorageService) *Service {
	return &Service{
		apiClient:      apiClient,
		storageService: storageService,
	}
}

// BackupWorkspace performs a complete backup of a workspace
func (s *Service) BackupWorkspace(ctx context.Context, workspaceID string) (*models.CompleteBackup, error) {
	logger.LogInfo(fmt.Sprintf("Starting backup for workspace: %s", workspaceID))

	// Create backup directory first - use consistent timestamp
	backupTime := time.Now()
	timestamp := backupTime.Format("2006-01-02_15-04-05")
	backupDir := filepath.Join(s.storageService.GetBackupPath(), workspaceID, timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.LogError(fmt.Sprintf("Failed to create backup directory: %s", backupDir), err)
		return nil, err
	}

	// Get workspace settings
	workspaceData, err := s.apiClient.GetWorkspaceSettings(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to get workspace settings", err)
		return nil, err
	}

	workspaceName, _ := workspaceData["name"].(string)

	backup := &models.CompleteBackup{
		Timestamp:     backupTime,
		WorkspaceID:   workspaceID,
		WorkspaceName: workspaceName,
		WorkspaceSettings: models.WorkspaceSettings{
			ID:   workspaceID,
			Name: workspaceName,
		},
	}

	// Backup reports
	logger.LogInfo("Backing up reports...")
	reports, err := s.backupReports(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to backup reports", err)
	} else {
		backup.Reports = reports
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d reports", len(reports)))
	}

	// Backup datasets
	logger.LogInfo("Backing up datasets...")
	datasets, err := s.backupDatasets(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to backup datasets", err)
	} else {
		backup.Datasets = datasets
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d datasets", len(datasets)))
	}

	// Backup dataflows
	logger.LogInfo("Backing up dataflows...")
	dataflows, err := s.backupDataflows(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to backup dataflows", err)
	} else {
		backup.Dataflows = dataflows
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d dataflows", len(dataflows)))
	}

	// Backup dashboards
	logger.LogInfo("Backing up dashboards...")
	dashboards, err := s.backupDashboards(ctx, workspaceID)
	if err != nil {
		logger.LogError("Failed to backup dashboards", err)
	} else {
		backup.Dashboards = dashboards
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d dashboards", len(dashboards)))
	}

	// Backup apps
	logger.LogInfo("Backing up apps...")
	apps, err := s.backupApps(ctx, workspaceID)
	if err != nil {
		logger.LogWarn("Failed to backup apps (optional)")
	} else {
		backup.Apps = apps
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d apps", len(apps)))
	}

	// Backup refresh schedules
	logger.LogInfo("Backing up refresh schedules...")
	schedules, err := s.backupRefreshSchedules(ctx, workspaceID, datasets)
	if err != nil {
		logger.LogError("Failed to backup refresh schedules", err)
	} else {
		backup.RefreshSchedules = schedules
		logger.LogInfo(fmt.Sprintf("Successfully backed up %d refresh schedules", len(schedules)))
	}

	// Export PBIX files for reports
	logger.LogInfo("Exporting reports as PBIX files...")
	pbixStatus, err := s.backupReportsPBIX(ctx, workspaceID, reports, backupDir)
	if err != nil {
		logger.LogWarn(fmt.Sprintf("PBIX export failed: %v", err))
	} else {
		logger.LogInfo(fmt.Sprintf("PBIX export status: %d succeeded, %d failed", pbixStatus["succeeded"], pbixStatus["failed"]))
	}

	// Save backup to storage
	logger.LogInfo("Saving backup to storage...")
	backupPath, err := s.storageService.SaveBackup(backup)
	if err != nil {
		logger.LogError("Failed to save backup", err)
		return nil, err
	}

	logger.LogInfo(fmt.Sprintf("✅ Backup completed successfully: %s", backupPath))
	return backup, nil
}

func (s *Service) backupReports(ctx context.Context, workspaceID string) ([]models.Report, error) {
	response, err := s.apiClient.GetReports(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		return []models.Report{}, nil
	}

	reports := make([]models.Report, 0, len(value))
	for _, item := range value {
		reportMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		report := models.Report{
			ID:        getString(reportMap, "id"),
			Name:      getString(reportMap, "name"),
			DatasetID: getString(reportMap, "datasetId"),
			EmbedURL:  getString(reportMap, "embedUrl"),
			WebURL:    getString(reportMap, "webUrl"),
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (s *Service) backupDatasets(ctx context.Context, workspaceID string) ([]models.Dataset, error) {
	response, err := s.apiClient.GetDatasets(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		return []models.Dataset{}, nil
	}

	datasets := make([]models.Dataset, 0, len(value))
	for _, item := range value {
		datasetMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		dataset := models.Dataset{
			ID:                  getString(datasetMap, "id"),
			Name:                getString(datasetMap, "name"),
			IsRefreshable:       getBool(datasetMap, "isRefreshable"),
			IsEffectiveIdentityRequired: getBool(datasetMap, "isEffectiveIdentityRequired"),
			IsEffectiveIdentityRolesRequired: getBool(datasetMap, "isEffectiveIdentityRolesRequired"),
		}
		datasets = append(datasets, dataset)
	}

	return datasets, nil
}

func (s *Service) backupDataflows(ctx context.Context, workspaceID string) ([]models.Dataflow, error) {
	response, err := s.apiClient.GetDataflows(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		return []models.Dataflow{}, nil
	}

	dataflows := make([]models.Dataflow, 0, len(value))
	for _, item := range value {
		dataflowMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		dataflow := models.Dataflow{
			ObjectID: getString(dataflowMap, "objectId"),
			Name:     getString(dataflowMap, "name"),
		}
		dataflows = append(dataflows, dataflow)
	}

	return dataflows, nil
}

func (s *Service) backupDashboards(ctx context.Context, workspaceID string) ([]models.Dashboard, error) {
	response, err := s.apiClient.GetDashboards(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		return []models.Dashboard{}, nil
	}

	dashboards := make([]models.Dashboard, 0, len(value))
	for _, item := range value {
		dashboardMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		dashboard := models.Dashboard{
			ID:          getString(dashboardMap, "id"),
			DisplayName: getString(dashboardMap, "displayName"),
			IsReadOnly:  getBool(dashboardMap, "isReadOnly"),
			EmbedURL:    getString(dashboardMap, "embedUrl"),
		}
		dashboards = append(dashboards, dashboard)
	}

	return dashboards, nil
}

func (s *Service) backupApps(ctx context.Context, workspaceID string) ([]models.App, error) {
	response, err := s.apiClient.GetApps(ctx)
	if err != nil {
		return []models.App{}, nil // Apps are optional
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		return []models.App{}, nil
	}

	apps := make([]models.App, 0)
	for _, item := range value {
		appMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Filter apps for this workspace
		if getString(appMap, "workspaceId") != workspaceID {
			continue
		}

		app := models.App{
			ID:          getString(appMap, "id"),
			Name:        getString(appMap, "name"),
			WorkspaceID: getString(appMap, "workspaceId"),
		}
		apps = append(apps, app)
	}

	return apps, nil
}

func (s *Service) backupRefreshSchedules(ctx context.Context, workspaceID string, datasets []models.Dataset) ([]models.RefreshSchedule, error) {
	schedules := make([]models.RefreshSchedule, 0)

	for _, dataset := range datasets {
		schedule, err := s.apiClient.GetRefreshSchedule(ctx, workspaceID, dataset.ID)
		if err != nil {
			logger.LogDebug(fmt.Sprintf("No refresh schedule for dataset: %s", dataset.Name))
			continue
		}

		schedules = append(schedules, models.RefreshSchedule{
			DatasetID:   dataset.ID,
			DatasetName: dataset.Name,
			Schedule:    schedule,
		})
	}

	return schedules, nil
}

// backupReportsPBIX exports all reports as PBIX files
func (s *Service) backupReportsPBIX(ctx context.Context, workspaceID string, reports []models.Report, backupDir string) (map[string]int, error) {
	if len(reports) == 0 {
		return map[string]int{"succeeded": 0, "failed": 0}, nil
	}

	// Create PBIX directory
	pbixDir := filepath.Join(backupDir, "pbix")
	if err := os.MkdirAll(pbixDir, 0755); err != nil {
		logger.LogError(fmt.Sprintf("Failed to create PBIX directory: %s", pbixDir), err)
		return nil, err
	}

	succeeded := 0
	failed := 0

	for _, report := range reports {
		logger.LogInfo(fmt.Sprintf("📥 Exporting report: %s", report.Name))

		// Create output path for PBIX file
		pbixFile := filepath.Join(pbixDir, fmt.Sprintf("%s.pbix", report.Name))

		// Export the report
		success, err := s.apiClient.ExportReport(ctx, workspaceID, report.ID, pbixFile)
		if err != nil || !success {
			logger.LogError(fmt.Sprintf("❌ Failed to export report: %s", report.Name), err)
			failed++
			continue
		}

		logger.LogInfo(fmt.Sprintf("✅ Report exported successfully: %s", report.Name))
		succeeded++
	}

	return map[string]int{"succeeded": succeeded, "failed": failed}, nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}
