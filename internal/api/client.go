package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/veeam/powerbi-backup-go/internal/auth"
	"github.com/veeam/powerbi-backup-go/internal/config"
	"github.com/veeam/powerbi-backup-go/internal/logger"
)

// Client is the Power BI API client
type Client struct {
	authService *auth.AuthService
	baseURL     string
	httpClient  *http.Client
}

// NewClient creates a new Power BI API client
func NewClient(authService *auth.AuthService, settings *config.Settings) *Client {
	return &Client{
		authService: authService,
		baseURL:     settings.APIBaseURL,
		httpClient:  &http.Client{},
	}
}

// fetchWithAuth makes an authenticated request to the Power BI API
func (c *Client) fetchWithAuth(ctx context.Context, method, endpoint string, body interface{}) (map[string]interface{}, error) {
	token, err := c.authService.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to create request for %s", endpoint), err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to fetch %s", endpoint), err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("Failed to read response body", err)
		return nil, err
	}

	if resp.StatusCode >= 400 {
		logger.LogError(fmt.Sprintf("Error fetching %s: %d - %s", endpoint, resp.StatusCode, string(respBody)), nil)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		logger.LogError("Failed to parse response JSON", err)
		return nil, err
	}

	return result, nil
}

// GetReports retrieves all reports from a workspace
func (c *Client) GetReports(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s/reports", workspaceID), nil)
}

// GetDatasets retrieves all datasets from a workspace
func (c *Client) GetDatasets(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s/datasets", workspaceID), nil)
}

// GetDataflows retrieves all dataflows from a workspace
func (c *Client) GetDataflows(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s/dataflows", workspaceID), nil)
}

// GetDashboards retrieves all dashboards from a workspace
func (c *Client) GetDashboards(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s/dashboards", workspaceID), nil)
}

// GetApps retrieves all apps
func (c *Client) GetApps(ctx context.Context) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", "/apps", nil)
}

// GetWorkspaceSettings retrieves workspace settings
func (c *Client) GetWorkspaceSettings(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s", workspaceID), nil)
}

// GetRefreshSchedule retrieves the refresh schedule for a dataset
func (c *Client) GetRefreshSchedule(ctx context.Context, workspaceID, datasetID string) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", fmt.Sprintf("/groups/%s/datasets/%s/refreshSchedule", workspaceID, datasetID), nil)
}

// ExportReport exports a report as a PBIX file
// Uses the simple /Export endpoint that returns the PBIX directly
func (c *Client) ExportReport(ctx context.Context, workspaceID, reportID, outputPath string) (bool, error) {
	token, err := c.authService.GetAccessToken(ctx)
	if err != nil {
		return false, err
	}

	// Direct export endpoint - GET returns PBIX file directly
	exportURL := fmt.Sprintf("%s/groups/%s/reports/%s/Export", c.baseURL, workspaceID, reportID)
	logger.LogInfo(fmt.Sprintf("✅ URL %s", exportURL))
	req, err := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
	if err != nil {
		logger.LogError("Failed to create export request", err)
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.LogError("Failed to export report", err)
		return false, err
	}
	defer resp.Body.Close()

	// If status is not 200, export failed
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.LogError(fmt.Sprintf("Export failed: %d - %s", resp.StatusCode, string(respBody)), nil)
		return false, fmt.Errorf("export failed: status %d", resp.StatusCode)
	}

	// Write the PBIX file to disk
	file, err := os.Create(outputPath)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to create output file: %s", outputPath), err)
		return false, err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		logger.LogError("Failed to write PBIX file", err)
		return false, err
	}

	logger.LogInfo(fmt.Sprintf("✅ Exported report to: %s", outputPath))
	return true, nil
}

// ImportPBIX imports a PBIX file to a workspace
func (c *Client) ImportPBIX(ctx context.Context, workspaceID, pbixPath, datasetName string) (bool, error) {
	token, err := c.authService.GetAccessToken(ctx)
	if err != nil {
		return false, err
	}

	// Open the PBIX file
	file, err := os.Open(pbixPath)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to open PBIX file: %s", pbixPath), err)
		return false, err
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file part
	part, err := writer.CreateFormFile("file", filepath.Base(pbixPath))
	if err != nil {
		logger.LogError("Failed to create form file", err)
		return false, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		logger.LogError("Failed to copy file to form", err)
		return false, err
	}

	writer.Close()

	// Create request
	url := fmt.Sprintf("%s/groups/%s/imports?datasetDisplayName=%s&nameConflict=Abort", 
		c.baseURL, workspaceID, datasetName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		logger.LogError("Failed to create import request", err)
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.LogError("Failed to import PBIX", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		logger.LogInfo(fmt.Sprintf("PBIX import queued successfully: %s", datasetName))
		return true, nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	logger.LogError(fmt.Sprintf("PBIX import failed: %d - %s", resp.StatusCode, string(respBody)), nil)
	return false, fmt.Errorf("import failed: %d - %s", resp.StatusCode, string(respBody))
}

// UpdateRefreshSchedule updates the refresh schedule for a dataset
func (c *Client) UpdateRefreshSchedule(ctx context.Context, workspaceID, datasetID string, schedule map[string]interface{}) error {
	_, err := c.fetchWithAuth(ctx, "PATCH", fmt.Sprintf("/groups/%s/datasets/%s/refreshSchedule", workspaceID, datasetID), schedule)
	return err
}

// GetWorkspaces retrieves all workspaces the user has access to
func (c *Client) GetWorkspaces(ctx context.Context) (map[string]interface{}, error) {
	return c.fetchWithAuth(ctx, "GET", "/groups", nil)
}

// CreateWorkspace creates a new workspace in Power BI
func (c *Client) CreateWorkspace(ctx context.Context, workspaceData map[string]interface{}) (interface{}, error) {
	result, err := c.fetchWithAuth(ctx, "POST", "/groups", workspaceData)
	if err != nil {
		return nil, err
	}
	return result, nil
}
