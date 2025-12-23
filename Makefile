# Build the application
build:
	go build -o powerbi-backup.exe ./cmd/main.go

# Run the application (backup)
run-backup:
	go run cmd/main.go --cmd backup --workspace-id $(WORKSPACE_ID)

# Run the application (restore)
run-restore:
	go run cmd/main.go --cmd restore --workspace-id $(WORKSPACE_ID) --backup-path $(BACKUP_PATH)

# Run all workspaces backup
run-backup-all:
	go run cmd/main.go --cmd backup --all

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f powerbi-backup.exe
	rm -f powerbi-backup

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Initialize project (first time setup)
init:
	go mod download
	copy .env.example .env
	@echo Please edit .env file with your credentials

# Quick start for Windows
quickstart-windows:
	copy .env.example .env
	go mod download
	go build -o powerbi-backup.exe ./cmd/main.go
	@echo Setup complete! Edit .env and run: powerbi-backup.exe --help

.PHONY: build run-backup run-restore run-backup-all deps test clean fmt lint init quickstart-windows
