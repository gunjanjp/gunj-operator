// Package backup provides backup and restore functionality for the Gunj operator
package backup

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupType represents the type of backup
type BackupType string

const (
	// FullBackup captures all data
	FullBackup BackupType = "Full"
	// IncrementalBackup captures only changes since last backup
	IncrementalBackup BackupType = "Incremental"
	// SnapshotBackup captures point-in-time state
	SnapshotBackup BackupType = "Snapshot"
)

// BackupPhase represents the current phase of a backup
type BackupPhase string

const (
	// BackupPhasePending indicates backup is pending
	BackupPhasePending BackupPhase = "Pending"
	// BackupPhaseInProgress indicates backup is in progress
	BackupPhaseInProgress BackupPhase = "InProgress"
	// BackupPhaseCompleted indicates backup completed successfully
	BackupPhaseCompleted BackupPhase = "Completed"
	// BackupPhaseFailed indicates backup failed
	BackupPhaseFailed BackupPhase = "Failed"
	// BackupPhaseDeleting indicates backup is being deleted
	BackupPhaseDeleting BackupPhase = "Deleting"
)

// RestorePhase represents the current phase of a restore
type RestorePhase string

const (
	// RestorePhasePending indicates restore is pending
	RestorePhasePending RestorePhase = "Pending"
	// RestorePhaseInProgress indicates restore is in progress
	RestorePhaseInProgress RestorePhase = "InProgress"
	// RestorePhaseCompleted indicates restore completed successfully
	RestorePhaseCompleted RestorePhase = "Completed"
	// RestorePhaseFailed indicates restore failed
	RestorePhaseFailed RestorePhase = "Failed"
)

// StorageProvider represents the storage provider type
type StorageProvider string

const (
	// S3Provider represents Amazon S3 storage
	S3Provider StorageProvider = "S3"
	// GCSProvider represents Google Cloud Storage
	GCSProvider StorageProvider = "GCS"
	// AzureBlobProvider represents Azure Blob Storage
	AzureBlobProvider StorageProvider = "AzureBlob"
	// LocalProvider represents local filesystem storage
	LocalProvider StorageProvider = "Local"
)

// BackupSpec defines the specification for a backup
type BackupSpec struct {
	// Type is the type of backup
	Type BackupType `json:"type"`
	
	// IncludedNamespaces is a list of namespaces to include
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`
	
	// ExcludedNamespaces is a list of namespaces to exclude
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	
	// IncludedResources is a list of resources to include
	IncludedResources []string `json:"includedResources,omitempty"`
	
	// ExcludedResources is a list of resources to exclude
	ExcludedResources []string `json:"excludedResources,omitempty"`
	
	// LabelSelector to filter resources
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	
	// StorageLocation is the backup storage location
	StorageLocation StorageLocation `json:"storageLocation"`
	
	// TTL is the time to live for the backup
	TTL *metav1.Duration `json:"ttl,omitempty"`
	
	// SnapshotVolumes specifies whether to snapshot volumes
	SnapshotVolumes bool `json:"snapshotVolumes,omitempty"`
	
	// IncludeClusterResources specifies whether to include cluster resources
	IncludeClusterResources bool `json:"includeClusterResources,omitempty"`
	
	// Hooks are commands to run at different phases
	Hooks *BackupHooks `json:"hooks,omitempty"`
	
	// EncryptionConfig specifies encryption settings
	EncryptionConfig *EncryptionConfig `json:"encryptionConfig,omitempty"`
	
	// CompressionConfig specifies compression settings
	CompressionConfig *CompressionConfig `json:"compressionConfig,omitempty"`
}

// BackupStatus defines the status of a backup
type BackupStatus struct {
	// Phase is the current phase of the backup
	Phase BackupPhase `json:"phase"`
	
	// StartTimestamp is when the backup started
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	
	// CompletionTimestamp is when the backup completed
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`
	
	// Progress is the backup progress percentage
	Progress int `json:"progress,omitempty"`
	
	// BackupItemsCount is the number of items backed up
	BackupItemsCount int `json:"backupItemsCount,omitempty"`
	
	// BackupSizeBytes is the size of the backup in bytes
	BackupSizeBytes int64 `json:"backupSizeBytes,omitempty"`
	
	// Errors is a list of errors during backup
	Errors []string `json:"errors,omitempty"`
	
	// Warnings is a list of warnings during backup
	Warnings []string `json:"warnings,omitempty"`
	
	// BackupLocation is where the backup is stored
	BackupLocation string `json:"backupLocation,omitempty"`
	
	// ValidationStatus is the validation status
	ValidationStatus *ValidationStatus `json:"validationStatus,omitempty"`
}

// RestoreSpec defines the specification for a restore
type RestoreSpec struct {
	// BackupName is the name of the backup to restore from
	BackupName string `json:"backupName"`
	
	// IncludedNamespaces is a list of namespaces to include
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`
	
	// ExcludedNamespaces is a list of namespaces to exclude
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	
	// IncludedResources is a list of resources to include
	IncludedResources []string `json:"includedResources,omitempty"`
	
	// ExcludedResources is a list of resources to exclude
	ExcludedResources []string `json:"excludedResources,omitempty"`
	
	// LabelSelector to filter resources
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	
	// NamespaceMapping maps backup namespaces to restore namespaces
	NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`
	
	// RestorePVs specifies whether to restore persistent volumes
	RestorePVs bool `json:"restorePVs,omitempty"`
	
	// PreserveNodePorts specifies whether to preserve node ports
	PreserveNodePorts bool `json:"preserveNodePorts,omitempty"`
	
	// Hooks are commands to run at different phases
	Hooks *RestoreHooks `json:"hooks,omitempty"`
	
	// RestorePointInTime specifies point-in-time recovery
	RestorePointInTime *metav1.Time `json:"restorePointInTime,omitempty"`
	
	// ValidationPolicy specifies validation requirements
	ValidationPolicy *ValidationPolicy `json:"validationPolicy,omitempty"`
}

// RestoreStatus defines the status of a restore
type RestoreStatus struct {
	// Phase is the current phase of the restore
	Phase RestorePhase `json:"phase"`
	
	// StartTimestamp is when the restore started
	StartTimestamp *metav1.Time `json:"startTimestamp,omitempty"`
	
	// CompletionTimestamp is when the restore completed
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`
	
	// Progress is the restore progress percentage
	Progress int `json:"progress,omitempty"`
	
	// RestoredItemsCount is the number of items restored
	RestoredItemsCount int `json:"restoredItemsCount,omitempty"`
	
	// Errors is a list of errors during restore
	Errors []string `json:"errors,omitempty"`
	
	// Warnings is a list of warnings during restore
	Warnings []string `json:"warnings,omitempty"`
	
	// ValidationStatus is the validation status
	ValidationStatus *ValidationStatus `json:"validationStatus,omitempty"`
}

// StorageLocation defines where backups are stored
type StorageLocation struct {
	// Provider is the storage provider
	Provider StorageProvider `json:"provider"`
	
	// Bucket is the storage bucket/container name
	Bucket string `json:"bucket"`
	
	// Prefix is the path prefix in the bucket
	Prefix string `json:"prefix,omitempty"`
	
	// Region is the storage region
	Region string `json:"region,omitempty"`
	
	// Config is provider-specific configuration
	Config map[string]string `json:"config,omitempty"`
	
	// Credential is the credential secret reference
	Credential *corev1.SecretKeySelector `json:"credential,omitempty"`
	
	// CACert is the CA certificate for the storage endpoint
	CACert []byte `json:"caCert,omitempty"`
	
	// InsecureSkipTLSVerify skips TLS verification
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// BackupHooks defines hooks for backup operations
type BackupHooks struct {
	// PreBackup are commands to run before backup
	PreBackup []Hook `json:"preBackup,omitempty"`
	
	// PostBackup are commands to run after backup
	PostBackup []Hook `json:"postBackup,omitempty"`
}

// RestoreHooks defines hooks for restore operations
type RestoreHooks struct {
	// PreRestore are commands to run before restore
	PreRestore []Hook `json:"preRestore,omitempty"`
	
	// PostRestore are commands to run after restore
	PostRestore []Hook `json:"postRestore,omitempty"`
}

// Hook defines a command to run
type Hook struct {
	// Name is the hook name
	Name string `json:"name"`
	
	// Command is the command to run
	Command []string `json:"command"`
	
	// Container is the container to run the command in
	Container string `json:"container,omitempty"`
	
	// OnError specifies what to do on error
	OnError HookErrorMode `json:"onError,omitempty"`
	
	// Timeout is the command timeout
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// HookErrorMode defines behavior on hook error
type HookErrorMode string

const (
	// HookErrorModeContinue continues on error
	HookErrorModeContinue HookErrorMode = "Continue"
	// HookErrorModeFail fails on error
	HookErrorModeFail HookErrorMode = "Fail"
)

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	// Algorithm is the encryption algorithm
	Algorithm string `json:"algorithm"`
	
	// KeyRef is the encryption key reference
	KeyRef *corev1.SecretKeySelector `json:"keyRef"`
	
	// KeyRotationPolicy defines key rotation
	KeyRotationPolicy *KeyRotationPolicy `json:"keyRotationPolicy,omitempty"`
}

// CompressionConfig defines compression settings
type CompressionConfig struct {
	// Algorithm is the compression algorithm
	Algorithm string `json:"algorithm"`
	
	// Level is the compression level
	Level int `json:"level,omitempty"`
}

// KeyRotationPolicy defines key rotation policy
type KeyRotationPolicy struct {
	// RotationPeriod is how often to rotate keys
	RotationPeriod *metav1.Duration `json:"rotationPeriod"`
	
	// RetainOldKeys is how many old keys to retain
	RetainOldKeys int `json:"retainOldKeys,omitempty"`
}

// ValidationStatus represents validation results
type ValidationStatus struct {
	// Valid indicates if validation passed
	Valid bool `json:"valid"`
	
	// Errors is a list of validation errors
	Errors []string `json:"errors,omitempty"`
	
	// Warnings is a list of validation warnings
	Warnings []string `json:"warnings,omitempty"`
	
	// ChecksumVerified indicates if checksums were verified
	ChecksumVerified bool `json:"checksumVerified"`
	
	// IntegrityVerified indicates if integrity was verified
	IntegrityVerified bool `json:"integrityVerified"`
}

// ValidationPolicy defines validation requirements
type ValidationPolicy struct {
	// VerifyChecksums requires checksum verification
	VerifyChecksums bool `json:"verifyChecksums"`
	
	// VerifyIntegrity requires integrity verification
	VerifyIntegrity bool `json:"verifyIntegrity"`
	
	// FailOnWarning fails validation on warnings
	FailOnWarning bool `json:"failOnWarning"`
	
	// MaxRetries is the maximum validation retries
	MaxRetries int `json:"maxRetries,omitempty"`
}

// BackupSchedule defines a backup schedule
type BackupSchedule struct {
	// Schedule is the cron schedule
	Schedule string `json:"schedule"`
	
	// Template is the backup template
	Template BackupSpec `json:"template"`
	
	// Suspend suspends the schedule
	Suspend bool `json:"suspend,omitempty"`
	
	// SuccessfulBackupsHistoryLimit is how many successful backups to keep
	SuccessfulBackupsHistoryLimit *int32 `json:"successfulBackupsHistoryLimit,omitempty"`
	
	// FailedBackupsHistoryLimit is how many failed backups to keep
	FailedBackupsHistoryLimit *int32 `json:"failedBackupsHistoryLimit,omitempty"`
	
	// RetentionPolicy defines retention rules
	RetentionPolicy *RetentionPolicy `json:"retentionPolicy,omitempty"`
}

// RetentionPolicy defines backup retention rules
type RetentionPolicy struct {
	// KeepLast is the number of backups to keep
	KeepLast int `json:"keepLast,omitempty"`
	
	// KeepDaily is the number of daily backups to keep
	KeepDaily int `json:"keepDaily,omitempty"`
	
	// KeepWeekly is the number of weekly backups to keep
	KeepWeekly int `json:"keepWeekly,omitempty"`
	
	// KeepMonthly is the number of monthly backups to keep
	KeepMonthly int `json:"keepMonthly,omitempty"`
	
	// KeepYearly is the number of yearly backups to keep
	KeepYearly int `json:"keepYearly,omitempty"`
}

// BackupItem represents an item in a backup
type BackupItem struct {
	// Name is the resource name
	Name string `json:"name"`
	
	// Namespace is the resource namespace
	Namespace string `json:"namespace,omitempty"`
	
	// GroupVersionKind identifies the resource type
	GroupVersionKind metav1.GroupVersionKind `json:"groupVersionKind"`
	
	// Data is the resource data
	Data runtime.RawExtension `json:"data"`
	
	// Metadata contains additional metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BackupProvider interface for backup storage providers
type BackupProvider interface {
	// Upload uploads backup data
	Upload(ctx context.Context, backup *BackupSpec, data []byte) error
	
	// Download downloads backup data
	Download(ctx context.Context, backup *BackupSpec) ([]byte, error)
	
	// Delete deletes backup data
	Delete(ctx context.Context, backup *BackupSpec) error
	
	// List lists backups
	List(ctx context.Context, location StorageLocation) ([]string, error)
	
	// Exists checks if backup exists
	Exists(ctx context.Context, backup *BackupSpec) (bool, error)
}

// BackupController interface for backup operations
type BackupController interface {
	// CreateBackup creates a new backup
	CreateBackup(ctx context.Context, backup *BackupSpec) (*BackupStatus, error)
	
	// GetBackup gets backup status
	GetBackup(ctx context.Context, name string) (*BackupStatus, error)
	
	// DeleteBackup deletes a backup
	DeleteBackup(ctx context.Context, name string) error
	
	// ListBackups lists all backups
	ListBackups(ctx context.Context) ([]BackupStatus, error)
}

// RestoreController interface for restore operations
type RestoreController interface {
	// CreateRestore creates a new restore
	CreateRestore(ctx context.Context, restore *RestoreSpec) (*RestoreStatus, error)
	
	// GetRestore gets restore status
	GetRestore(ctx context.Context, name string) (*RestoreStatus, error)
	
	// ListRestores lists all restores
	ListRestores(ctx context.Context) ([]RestoreStatus, error)
}

// SnapshotProvider interface for volume snapshots
type SnapshotProvider interface {
	// CreateSnapshot creates a volume snapshot
	CreateSnapshot(ctx context.Context, volumeID string, tags map[string]string) (string, error)
	
	// DeleteSnapshot deletes a volume snapshot
	DeleteSnapshot(ctx context.Context, snapshotID string) error
	
	// RestoreSnapshot restores from a snapshot
	RestoreSnapshot(ctx context.Context, snapshotID string, volumeID string) error
	
	// ListSnapshots lists volume snapshots
	ListSnapshots(ctx context.Context, tags map[string]string) ([]string, error)
}

// ScheduleManager interface for backup scheduling
type ScheduleManager interface {
	// CreateSchedule creates a backup schedule
	CreateSchedule(ctx context.Context, name string, schedule *BackupSchedule) error
	
	// UpdateSchedule updates a backup schedule
	UpdateSchedule(ctx context.Context, name string, schedule *BackupSchedule) error
	
	// DeleteSchedule deletes a backup schedule
	DeleteSchedule(ctx context.Context, name string) error
	
	// GetSchedule gets a backup schedule
	GetSchedule(ctx context.Context, name string) (*BackupSchedule, error)
	
	// ListSchedules lists all backup schedules
	ListSchedules(ctx context.Context) ([]BackupSchedule, error)
	
	// TriggerSchedule manually triggers a scheduled backup
	TriggerSchedule(ctx context.Context, name string) error
}

// BackupMetrics contains backup-related metrics
type BackupMetrics struct {
	// TotalBackups is the total number of backups
	TotalBackups int64 `json:"totalBackups"`
	
	// SuccessfulBackups is the number of successful backups
	SuccessfulBackups int64 `json:"successfulBackups"`
	
	// FailedBackups is the number of failed backups
	FailedBackups int64 `json:"failedBackups"`
	
	// TotalRestores is the total number of restores
	TotalRestores int64 `json:"totalRestores"`
	
	// SuccessfulRestores is the number of successful restores
	SuccessfulRestores int64 `json:"successfulRestores"`
	
	// FailedRestores is the number of failed restores
	FailedRestores int64 `json:"failedRestores"`
	
	// StorageUsageBytes is the total storage usage
	StorageUsageBytes int64 `json:"storageUsageBytes"`
	
	// LastBackupTime is the time of the last backup
	LastBackupTime *time.Time `json:"lastBackupTime,omitempty"`
	
	// LastRestoreTime is the time of the last restore
	LastRestoreTime *time.Time `json:"lastRestoreTime,omitempty"`
}
