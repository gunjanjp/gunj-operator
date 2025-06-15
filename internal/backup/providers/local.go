package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
)

// LocalProvider implements backup provider for local filesystem
type LocalProvider struct {
	basePath string
	log      logr.Logger
}

// NewLocalProvider creates a new local filesystem backup provider
func NewLocalProvider(log logr.Logger) *LocalProvider {
	// Default base path for local backups
	basePath := os.Getenv("LOCAL_BACKUP_PATH")
	if basePath == "" {
		basePath = "/var/lib/gunj-operator/backups"
	}
	
	return &LocalProvider{
		basePath: basePath,
		log:      log.WithName("local-provider"),
	}
}

// Upload uploads backup data to local filesystem
func (p *LocalProvider) Upload(ctx context.Context, spec *backup.BackupSpec, data []byte) error {
	filePath := p.getFilePath(spec)
	
	p.log.V(1).Info("Uploading backup to local filesystem", "path", filePath)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating backup directory: %w", err)
	}
	
	// Write backup data
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("writing backup file: %w", err)
	}
	
	p.log.Info("Successfully uploaded backup to local filesystem", "path", filePath, "size", len(data))
	
	return nil
}

// Download downloads backup data from local filesystem
func (p *LocalProvider) Download(ctx context.Context, spec *backup.BackupSpec) ([]byte, error) {
	filePath := p.getFilePath(spec)
	
	p.log.V(1).Info("Downloading backup from local filesystem", "path", filePath)
	
	// Read backup data
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading backup file: %w", err)
	}
	
	p.log.Info("Successfully downloaded backup from local filesystem", "path", filePath, "size", len(data))
	
	return data, nil
}

// Delete deletes backup data from local filesystem
func (p *LocalProvider) Delete(ctx context.Context, spec *backup.BackupSpec) error {
	filePath := p.getFilePath(spec)
	
	p.log.V(1).Info("Deleting backup from local filesystem", "path", filePath)
	
	// Remove backup file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("removing backup file: %w", err)
	}
	
	// Try to remove empty parent directories
	dir := filepath.Dir(filePath)
	for dir != p.basePath && dir != "/" {
		if err := os.Remove(dir); err != nil {
			// Directory not empty or other error, stop trying
			break
		}
		dir = filepath.Dir(dir)
	}
	
	p.log.Info("Successfully deleted backup from local filesystem", "path", filePath)
	
	return nil
}

// List lists backups in local filesystem
func (p *LocalProvider) List(ctx context.Context, location backup.StorageLocation) ([]string, error) {
	searchPath := filepath.Join(p.basePath, location.Bucket, location.Prefix)
	
	p.log.V(1).Info("Listing backups in local filesystem", "path", searchPath)
	
	var backups []string
	
	// Walk the directory tree
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Handle errors gracefully
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(p.basePath, path)
		if err != nil {
			return err
		}
		
		backups = append(backups, relPath)
		return nil
	})
	
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("listing backups: %w", err)
	}
	
	p.log.V(1).Info("Found backups in local filesystem", "count", len(backups))
	
	return backups, nil
}

// Exists checks if backup exists in local filesystem
func (p *LocalProvider) Exists(ctx context.Context, spec *backup.BackupSpec) (bool, error) {
	filePath := p.getFilePath(spec)
	
	p.log.V(1).Info("Checking if backup exists in local filesystem", "path", filePath)
	
	// Check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking file existence: %w", err)
	}
	
	return true, nil
}

// getFilePath generates the local file path for a backup
func (p *LocalProvider) getFilePath(spec *backup.BackupSpec) string {
	// Generate a unique path based on backup spec
	timestamp := spec.StorageLocation.Config["timestamp"]
	if timestamp == "" {
		timestamp = "latest"
	}
	
	filename := fmt.Sprintf("backup-%s-%s.tar.gz", spec.Type, timestamp)
	
	return filepath.Join(
		p.basePath,
		spec.StorageLocation.Bucket,
		spec.StorageLocation.Prefix,
		filename,
	)
}
