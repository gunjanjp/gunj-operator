package providers

import (
	"context"
	"fmt"
	"io"
	"path"

	"cloud.google.com/go/storage"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
	"google.golang.org/api/iterator"
)

// GCSProvider implements backup provider for Google Cloud Storage
type GCSProvider struct {
	client *storage.Client
	log    logr.Logger
}

// NewGCSProvider creates a new GCS backup provider
func NewGCSProvider(log logr.Logger) (*GCSProvider, error) {
	// Create GCS client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating GCS client: %w", err)
	}
	
	return &GCSProvider{
		client: client,
		log:    log.WithName("gcs-provider"),
	}, nil
}

// Upload uploads backup data to GCS
func (p *GCSProvider) Upload(ctx context.Context, spec *backup.BackupSpec, data []byte) error {
	objectName := p.getObjectName(spec)
	
	p.log.V(1).Info("Uploading backup to GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	// Get bucket handle
	bucket := p.client.Bucket(spec.StorageLocation.Bucket)
	obj := bucket.Object(objectName)
	
	// Create writer
	writer := obj.NewWriter(ctx)
	
	// Set storage class if configured
	if storageClass, ok := spec.StorageLocation.Config["storageClass"]; ok {
		writer.StorageClass = storageClass
	}
	
	// Set encryption if configured
	if spec.EncryptionConfig != nil {
		// GCS supports customer-managed encryption keys
		if kmsKeyName, ok := spec.StorageLocation.Config["kmsKeyName"]; ok {
			writer.KMSKeyName = kmsKeyName
		}
	}
	
	// Write data
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("writing to GCS: %w", err)
	}
	
	// Close writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing GCS writer: %w", err)
	}
	
	p.log.Info("Successfully uploaded backup to GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	return nil
}

// Download downloads backup data from GCS
func (p *GCSProvider) Download(ctx context.Context, spec *backup.BackupSpec) ([]byte, error) {
	objectName := p.getObjectName(spec)
	
	p.log.V(1).Info("Downloading backup from GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	// Get bucket handle
	bucket := p.client.Bucket(spec.StorageLocation.Bucket)
	obj := bucket.Object(objectName)
	
	// Create reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating GCS reader: %w", err)
	}
	defer reader.Close()
	
	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading from GCS: %w", err)
	}
	
	p.log.Info("Successfully downloaded backup from GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName, "size", len(data))
	
	return data, nil
}

// Delete deletes backup data from GCS
func (p *GCSProvider) Delete(ctx context.Context, spec *backup.BackupSpec) error {
	objectName := p.getObjectName(spec)
	
	p.log.V(1).Info("Deleting backup from GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	// Get bucket handle
	bucket := p.client.Bucket(spec.StorageLocation.Bucket)
	obj := bucket.Object(objectName)
	
	// Delete object
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("deleting from GCS: %w", err)
	}
	
	p.log.Info("Successfully deleted backup from GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	return nil
}

// List lists backups in GCS
func (p *GCSProvider) List(ctx context.Context, location backup.StorageLocation) ([]string, error) {
	p.log.V(1).Info("Listing backups in GCS", "bucket", location.Bucket, "prefix", location.Prefix)
	
	var backups []string
	
	// Get bucket handle
	bucket := p.client.Bucket(location.Bucket)
	
	// Create query
	query := &storage.Query{
		Prefix: location.Prefix,
	}
	
	// List objects
	it := bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}
		
		backups = append(backups, attrs.Name)
	}
	
	p.log.V(1).Info("Found backups in GCS", "count", len(backups))
	
	return backups, nil
}

// Exists checks if backup exists in GCS
func (p *GCSProvider) Exists(ctx context.Context, spec *backup.BackupSpec) (bool, error) {
	objectName := p.getObjectName(spec)
	
	p.log.V(1).Info("Checking if backup exists in GCS", "bucket", spec.StorageLocation.Bucket, "object", objectName)
	
	// Get bucket handle
	bucket := p.client.Bucket(spec.StorageLocation.Bucket)
	obj := bucket.Object(objectName)
	
	// Check if object exists
	_, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("checking object existence: %w", err)
	}
	
	return true, nil
}

// getObjectName generates the GCS object name for a backup
func (p *GCSProvider) getObjectName(spec *backup.BackupSpec) string {
	// Generate a unique name based on backup spec
	timestamp := spec.StorageLocation.Config["timestamp"]
	if timestamp == "" {
		timestamp = "latest"
	}
	
	return path.Join(spec.StorageLocation.Prefix, fmt.Sprintf("backup-%s-%s.tar.gz", spec.Type, timestamp))
}
