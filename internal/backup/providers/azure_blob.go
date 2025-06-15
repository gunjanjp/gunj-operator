package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
)

// AzureBlobProvider implements backup provider for Azure Blob Storage
type AzureBlobProvider struct {
	client *azblob.Client
	log    logr.Logger
}

// NewAzureBlobProvider creates a new Azure Blob backup provider
func NewAzureBlobProvider(log logr.Logger) (*AzureBlobProvider, error) {
	// Get Azure storage account connection string from environment or config
	connectionString := "" // This should be loaded from configuration
	
	if connectionString == "" {
		return nil, fmt.Errorf("Azure storage connection string not configured")
	}
	
	// Create Azure Blob client
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("creating Azure Blob client: %w", err)
	}
	
	return &AzureBlobProvider{
		client: client,
		log:    log.WithName("azure-blob-provider"),
	}, nil
}

// Upload uploads backup data to Azure Blob Storage
func (p *AzureBlobProvider) Upload(ctx context.Context, spec *backup.BackupSpec, data []byte) error {
	blobName := p.getBlobName(spec)
	containerName := spec.StorageLocation.Bucket
	
	p.log.V(1).Info("Uploading backup to Azure Blob Storage", "container", containerName, "blob", blobName)
	
	// Upload the backup
	_, err := p.client.UploadBuffer(ctx, containerName, blobName, data, &azblob.UploadBufferOptions{
		Metadata: map[string]*string{
			"backupType": stringPtr(string(spec.Type)),
		},
	})
	
	if err != nil {
		return fmt.Errorf("uploading to Azure Blob: %w", err)
	}
	
	p.log.Info("Successfully uploaded backup to Azure Blob Storage", "container", containerName, "blob", blobName)
	
	return nil
}

// Download downloads backup data from Azure Blob Storage
func (p *AzureBlobProvider) Download(ctx context.Context, spec *backup.BackupSpec) ([]byte, error) {
	blobName := p.getBlobName(spec)
	containerName := spec.StorageLocation.Bucket
	
	p.log.V(1).Info("Downloading backup from Azure Blob Storage", "container", containerName, "blob", blobName)
	
	// Download the backup
	get, err := p.client.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading from Azure Blob: %w", err)
	}
	
	// Read the data
	downloadedData := bytes.Buffer{}
	reader := get.NewRetryReader(ctx, &azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("reading backup data: %w", err)
	}
	err = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("closing reader: %w", err)
	}
	
	data := downloadedData.Bytes()
	p.log.Info("Successfully downloaded backup from Azure Blob Storage", "container", containerName, "blob", blobName, "size", len(data))
	
	return data, nil
}

// Delete deletes backup data from Azure Blob Storage
func (p *AzureBlobProvider) Delete(ctx context.Context, spec *backup.BackupSpec) error {
	blobName := p.getBlobName(spec)
	containerName := spec.StorageLocation.Bucket
	
	p.log.V(1).Info("Deleting backup from Azure Blob Storage", "container", containerName, "blob", blobName)
	
	// Delete the backup
	_, err := p.client.DeleteBlob(ctx, containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("deleting from Azure Blob: %w", err)
	}
	
	p.log.Info("Successfully deleted backup from Azure Blob Storage", "container", containerName, "blob", blobName)
	
	return nil
}

// List lists backups in Azure Blob Storage
func (p *AzureBlobProvider) List(ctx context.Context, location backup.StorageLocation) ([]string, error) {
	containerName := location.Bucket
	prefix := location.Prefix
	
	p.log.V(1).Info("Listing backups in Azure Blob Storage", "container", containerName, "prefix", prefix)
	
	var backups []string
	
	// List blobs with prefix
	pager := p.client.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})
	
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing blobs: %w", err)
		}
		
		for _, blob := range resp.Segment.BlobItems {
			if blob.Name != nil {
				backups = append(backups, *blob.Name)
			}
		}
	}
	
	p.log.V(1).Info("Found backups in Azure Blob Storage", "count", len(backups))
	
	return backups, nil
}

// Exists checks if backup exists in Azure Blob Storage
func (p *AzureBlobProvider) Exists(ctx context.Context, spec *backup.BackupSpec) (bool, error) {
	blobName := p.getBlobName(spec)
	containerName := spec.StorageLocation.Bucket
	
	p.log.V(1).Info("Checking if backup exists in Azure Blob Storage", "container", containerName, "blob", blobName)
	
	// Check if blob exists
	_, err := p.client.GetBlobProperties(ctx, containerName, blobName, nil)
	if err != nil {
		// Check if it's a not found error
		if isAzureNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking blob existence: %w", err)
	}
	
	return true, nil
}

// getBlobName generates the Azure blob name for a backup
func (p *AzureBlobProvider) getBlobName(spec *backup.BackupSpec) string {
	// Generate a unique name based on backup spec
	timestamp := spec.StorageLocation.Config["timestamp"]
	if timestamp == "" {
		timestamp = "latest"
	}
	
	return path.Join(spec.StorageLocation.Prefix, fmt.Sprintf("backup-%s-%s.tar.gz", spec.Type, timestamp))
}

// isAzureNotFoundError checks if an error is a not found error
func isAzureNotFoundError(err error) bool {
	// Check for specific Azure not found errors
	// This is a simplified check - Azure SDK has more sophisticated error handling
	return err != nil && err.Error() == "BlobNotFound"
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
