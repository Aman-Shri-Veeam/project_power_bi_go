# PowerBI Backup & Restore - Go Implementation

**Complete web server with PBIX export/import functionality**

![Status](https://img.shields.io/badge/status-production%20ready-brightgreen)
![Go](https://img.shields.io/badge/go-1.21+-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## 📋 Overview

A complete Go web server for backing up and restoring Power BI workspaces. Exports reports as PBIX files and includes refresh schedule management.

**Key Features:**
- ✅ Backup all workspace components (reports, datasets, dashboards, apps, etc.)
- ✅ Export reports as **PBIX files** 
- ✅ Restore PBIX files to target workspace
- ✅ Auto-handle duplicate dataset names
- ✅ Restore refresh schedules to imported datasets
- ✅ Web UI and REST API
- ✅ Service principal authentication
- ✅ Comprehensive error handling & logging

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ installed
- Power BI tenant with Service Principal credentials
- Windows PowerShell or similar terminal

### 1. Configuration

Create or update `.env` file:

```env
POWERBI_CLIENT_ID=your-client-id
POWERBI_CLIENT_SECRET=your-client-secret
POWERBI_TENANT_ID=your-tenant-id
API_BASE_URL=https://api.powerbi.com/v1.0/myorg
BACKUP_PATH=./backups
DEBUG=true
```

### 2. Build & Run

```powershell
# Build
go build -o powerbi-backup-server.exe ./cmd/server

# Run
.\powerbi-backup-server.exe
```

Server starts on: **http://localhost:8060**

### 3. Access UI

Open browser: http://localhost:8060

**API:** http://localhost:8060/api/*

---

## 📦 Project Structure

```
powerbi-backup-go/
├── cmd/
│   └── server/
│       └── main.go              # Server entry point
├── internal/
│   ├── api/
│   │   └── client.go            # Power BI API client
│   ├── auth/
│   │   └── authservice.go       # Service principal auth
│   ├── backup/
│   │   └── service.go           # Backup orchestration
│   ├── restore/
│   │   └── service.go           # Restore orchestration
│   ├── storage/
│   │   └── service.go           # Backup file storage
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── models/
│   │   └── models.go            # Data structures
│   └── logger/
│       └── logger.go            # Logging
├── web/
│   └── static/
│       ├── index.html           # Web UI
│       ├── app.js               # Frontend logic
│       └── style.css            # Styling
├── go.mod                       # Go module
├── go.sum                       # Dependencies
├── .env                         # Configuration (local)
├── .env.example                 # Configuration template
├── Makefile                     # Build commands
└── README.md                    # This file
```

---

## 🔌 API Endpoints

### Health & Status
```
GET /api/health
```

### Workspaces
```
GET /api/workspaces              # List all workspaces
POST /api/workspace/create       # Create new workspace
```

### Backup & Restore
```
POST /api/backup                 # Start backup
POST /api/restore                # Start restore
GET /api/backups                 # List available backups
```

---

## 💾 Backup Structure

```
backups/
  {workspaceId}/
    {timestamp}/
      backup.json               # Metadata (reports, datasets, etc.)
      pbix/
        report1.pbix           # Exported reports as PBIX
        report2.pbix
        report3.pbix
```

---

## 🔄 Backup Workflow

```
1. BackupWorkspace()
   ├─ Get workspace metadata
   ├─ Backup reports (metadata)
   ├─ Backup datasets (metadata)
   ├─ Backup dataflows
   ├─ Backup dashboards
   ├─ Backup apps
   ├─ Backup refresh schedules
   └─ Export reports as PBIX files
       └─ For each report: GET /groups/{id}/reports/{id}/Export
           └─ Save to pbix/{name}.pbix

2. SaveBackup()
   └─ Save backup.json + PBIX files
```

---

## 🔄 Restore Workflow

```
1. RestoreWorkspace()
   ├─ Load backup.json
   ├─ Import PBIX files
   │   └─ For each PBIX: POST /groups/{id}/imports
   │       └─ Handle duplicate names (name -> name_1, name_2)
   └─ Restore refresh schedules
       └─ Update schedules for imported datasets
```

---

## 🧪 Testing

### Backup Example
```bash
curl -X POST http://localhost:8060/api/backup \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"d239010c-9322-4053-bb14-c54167f2c7c6"}'
```

### Check Backup
```bash
ls backups/*/*/pbix/
# Should list PBIX files
```

### Restore Example
```bash
curl -X POST http://localhost:8060/api/restore \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"<TARGET-WS>","backup_path":"backups/.../<TIMESTAMP>"}'
```

---

## 🛠 Build Commands

```bash
# Build
make build
# or
go build -o powerbi-backup-server.exe ./cmd/server

# Run
make run
# or
.\powerbi-backup-server.exe

# Development
make dev
# or
go run ./cmd/server/main.go

# Format code
make fmt
# or
go fmt ./...

# Run tests
make test
# or
go test ./...
```

---

## 📊 Features

### Export (Backup)
- ✅ Simple GET `/Export` endpoint
- ✅ Direct binary PBIX download
- ✅ Saves to `backup/pbix/` directory
- ✅ Error handling and logging

### Import (Restore)
- ✅ Multipart form upload
- ✅ POST `/imports` endpoint
- ✅ Automatic duplicate detection
- ✅ Dataset name conflict resolution

### Refresh Schedules
- ✅ Backup schedule configurations
- ✅ Restore after import
- ✅ Dataset ID mapping
- ✅ Graceful error handling

---

## 🔐 Authentication

Uses **Service Principal** authentication:

1. Credentials from `.env` file
2. OAuth2 token request to Azure AD
3. Bearer token in API requests
4. Automatic token caching

---

## 📋 Key Methods

### API Client (`internal/api/client.go`)
```go
// Export report as PBIX
ExportReport(ctx, workspaceID, reportID, outputPath) (bool, error)

// Import PBIX file
ImportPBIX(ctx, workspaceID, pbixPath, datasetName) (bool, error)

// Get workspaces, reports, datasets, dashboards, etc.
GetWorkspaces(ctx) (map[string]interface{}, error)
GetReports(ctx, workspaceID) (map[string]interface{}, error)
GetDatasets(ctx, workspaceID) (map[string]interface{}, error)
```

### Backup Service (`internal/backup/service.go`)
```go
// Main backup orchestration
BackupWorkspace(ctx, workspaceID) (*models.CompleteBackup, error)

// Export reports as PBIX
backupReportsPBIX(ctx, workspaceID, reports, backupDir) (map[string]int, error)
```

### Restore Service (`internal/restore/service.go`)
```go
// Main restore orchestration
RestoreWorkspace(ctx, targetWorkspaceID, backupPath) error

// Import PBIX files with duplicate handling
restoreReportsPBIX(ctx, workspaceID, backupPath) error

// Restore refresh schedules
restoreRefreshSchedules(ctx, workspaceID, schedules) error
```

---

## 🎯 Configuration

### Environment Variables

| Variable | Required | Example |
|----------|----------|---------|
| `POWERBI_CLIENT_ID` | Yes | `2e20d70b-f8c6-412c-9e0c-7729dc1d5080` |
| `POWERBI_CLIENT_SECRET` | Yes | `3EY8Q~...` |
| `POWERBI_TENANT_ID` | Yes | `48bf783f-81f9-41a8-917e-045fbca6b055` |
| `API_BASE_URL` | No | `https://api.powerbi.com/v1.0/myorg` |
| `BACKUP_PATH` | No | `./backups` |
| `DEBUG` | No | `true` / `false` |

---

## 📝 Logging

Comprehensive logging with levels:
- **INFO** - Normal operations
- **WARN** - Non-critical issues
- **ERROR** - Error conditions
- **DEBUG** - Detailed debug info (when DEBUG=true)

---

## 🤝 Contributing

See individual package documentation for implementation details.

---

## 📄 License

MIT License

---

## 🚀 Deployment

### Run from executable
```bash
.\powerbi-backup-server.exe
```

### Run with Go directly
```bash
go run ./cmd/server/main.go
```

### Access UI
```
http://localhost:8060
```

---

## 📞 Support

- Check logs for detailed error information
- Verify `.env` configuration
- Ensure Power BI service principal has required permissions
- Check network connectivity to Azure AD and Power BI API

---

## ✅ Status

**Version:** 1.0.0  
**Last Updated:** December 23, 2025  

- ✅ PBIX Export/Import
- ✅ Duplicate handling
- ✅ Refresh schedules
- ✅ Web UI & API
- ✅ Complete round-trip backup/restore

---

**Ready to run!** 
