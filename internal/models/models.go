package models

import "time"

// Report represents a Power BI report
type Report struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	DatasetID string      `json:"datasetId"`
	EmbedURL  string      `json:"embedUrl"`
	WebURL    string      `json:"webUrl"`
	Pages     []ReportPage `json:"pages,omitempty"`
}

// ReportPage represents a report page
type ReportPage struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Order       int    `json:"order"`
}

// Dataset represents a Power BI dataset
type Dataset struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	ConfigRefreshType  *string `json:"configuredBy,omitempty"`
	IsRefreshable      bool    `json:"isRefreshable"`
	IsEffectiveIdentityRequired bool `json:"isEffectiveIdentityRequired"`
	IsEffectiveIdentityRolesRequired bool `json:"isEffectiveIdentityRolesRequired"`
}

// Dataflow represents a Power BI dataflow
type Dataflow struct {
	ObjectID    string  `json:"objectId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// Dashboard represents a Power BI dashboard
type Dashboard struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	IsReadOnly   bool   `json:"isReadOnly"`
	EmbedURL     string `json:"embedUrl,omitempty"`
}

// App represents a Power BI app
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	WorkspaceID string `json:"workspaceId,omitempty"`
}

// RefreshSchedule represents a dataset refresh schedule
type RefreshSchedule struct {
	DatasetID   string                 `json:"datasetId"`
	DatasetName string                 `json:"datasetName"`
	Schedule    map[string]interface{} `json:"schedule"`
}

// WorkspaceSettings represents workspace configuration
type WorkspaceSettings struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type,omitempty"`
	State    string                 `json:"state,omitempty"`
	IsReadOnly bool                 `json:"isReadOnly,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// CompleteBackup represents a complete backup of a workspace
type CompleteBackup struct {
	Timestamp         time.Time           `json:"timestamp"`
	WorkspaceID       string              `json:"workspaceId"`
	WorkspaceName     string              `json:"workspaceName"`
	Reports           []Report            `json:"reports"`
	Datasets          []Dataset           `json:"datasets"`
	Dataflows         []Dataflow          `json:"dataflows"`
	Dashboards        []Dashboard         `json:"dashboards"`
	Apps              []App               `json:"apps"`
	RefreshSchedules  []RefreshSchedule   `json:"refreshSchedules"`
	WorkspaceSettings WorkspaceSettings   `json:"workspaceSettings"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Value []map[string]interface{} `json:"value"`
}

// TokenResponse represents an OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}
