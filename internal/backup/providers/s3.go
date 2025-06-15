package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
)

// S3Provider implements backup provider for Amazon S3
type S3Provider struct {
	client *s3.Client
	log    logr.Logger
}

// NewS3Provider creates a new S3 backup provider
func NewS3Provider(log logr.Logger) (*S3Provider, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	
	// Create S3 client
	client := s3.NewFromConfig(cfg)
	
	return &S3Provider{
		client: client,
		log:    log.WithName("s3-provider"),
	}, nil
}

// Upload uploads backup data to S3
func (p *S3Provider) Upload(ctx context.Context, spec *backup.BackupSpec, data []byte) error {
	key := p.getObjectKey(spec)
	
	p.log.V(1).Info("Uploading backup to S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket: aws.String(spec.StorageLocation.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	
	// Add encryption if configured
	if spec.EncryptionConfig != nil {
		input.ServerSideEncryption = types.ServerSideEncryptionAes256
	}
	
	// Add storage class if configured
	if storageClass, ok := spec.StorageLocation.Config["storageClass"]; ok {
		input.StorageClass = types.StorageClass(storageClass)
	}
	
	// Upload the backup
	_, err := p.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("uploading to S3: %w", err)
	}
	
	p.log.Info("Successfully uploaded backup to S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	return nil
}

// Download downloads backup data from S3
func (p *S3Provider) Download(ctx context.Context, spec *backup.BackupSpec) ([]byte, error) {
	key := p.getObjectKey(spec)
	
	p.log.V(1).Info("Downloading backup from S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	// Download the backup
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(spec.StorageLocation.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading from S3: %w", err)
	}
	defer result.Body.Close()
	
	// Read the data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("reading backup data: %w", err)
	}
	
	p.log.Info("Successfully downloaded backup from S3", "bucket", spec.StorageLocation.Bucket, "key", key, "size", len(data))
	
	return data, nil
}

// Delete deletes backup data from S3
func (p *S3Provider) Delete(ctx context.Context, spec *backup.BackupSpec) error {
	key := p.getObjectKey(spec)
	
	p.log.V(1).Info("Deleting backup from S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	// Delete the backup
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(spec.StorageLocation.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}
	
	p.log.Info("Successfully deleted backup from S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	return nil
}

// List lists backups in S3
func (p *S3Provider) List(ctx context.Context, location backup.StorageLocation) ([]string, error) {
	p.log.V(1).Info("Listing backups in S3", "bucket", location.Bucket, "prefix", location.Prefix)
	
	var backups []string
	
	// List objects with prefix
	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(location.Bucket),
		Prefix: aws.String(location.Prefix),
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}
		
		for _, obj := range page.Contents {
			backups = append(backups, *obj.Key)
		}
	}
	
	p.log.V(1).Info("Found backups in S3", "count", len(backups))
	
	return backups, nil
}

// Exists checks if backup exists in S3
func (p *S3Provider) Exists(ctx context.Context, spec *backup.BackupSpec) (bool, error) {
	key := p.getObjectKey(spec)
	
	p.log.V(1).Info("Checking if backup exists in S3", "bucket", spec.StorageLocation.Bucket, "key", key)
	
	// Check if object exists
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(spec.StorageLocation.Bucket),
		Key:    aws.String(key),
	})
	
	if err != nil {
		// Check if it's a not found error
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking object existence: %w", err)
	}
	
	return true, nil
}

// getObjectKey generates the S3 object key for a backup
func (p *S3Provider) getObjectKey(spec *backup.BackupSpec) string {
	// Generate a unique key based on backup spec
	// This is a simplified version - in production, you'd want a more robust naming scheme
	timestamp := spec.StorageLocation.Config["timestamp"]
	if timestamp == "" {
		timestamp = "latest"
	}
	
	return path.Join(spec.StorageLocation.Prefix, fmt.Sprintf("backup-%s-%s.tar.gz", spec.Type, timestamp))
}

// isNotFoundError checks if an error is a not found error
func isNotFoundError(err error) bool {
	// Check for specific S3 not found errors
	// This is a simplified check - AWS SDK v2 has more sophisticated error handling
	return err != nil && err.Error() == "NoSuchKey"
}
