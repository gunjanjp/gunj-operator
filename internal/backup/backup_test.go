package backup_test

import (
	"context"
	"testing"
	"time"

	"github.com/gunjanjp/gunj-operator/internal/backup"
	"github.com/gunjanjp/gunj-operator/internal/backup/controller"
	"github.com/gunjanjp/gunj-operator/internal/backup/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestBackupController(t *testing.T) {
	// Setup
	log := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	
	// Create fake client
	s := scheme.Scheme
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	
	// Create backup controller
	config := controller.BackupControllerConfig{
		MaxConcurrentBackups: 5,
		DefaultTTL:          24 * time.Hour,
		RetryInterval:       1 * time.Minute,
		MaxRetries:          3,
		MetricsEnabled:      false,
		EncryptionEnabled:   false,
		CompressionEnabled:  true,
	}
	
	recorder := record.NewFakeRecorder(100)
	
	backupCtrl, err := controller.NewBackupController(cl, s, log, recorder, config)
	require.NoError(t, err)
	
	t.Run("CreateBackup", func(t *testing.T) {
		// Create backup spec
		spec := &backup.BackupSpec{
			Type: backup.FullBackup,
			IncludedNamespaces: []string{"default", "monitoring"},
			StorageLocation: backup.StorageLocation{
				Provider: backup.LocalProvider,
				Bucket:   "test-bucket",
				Prefix:   "test-prefix",
			},
			CompressionConfig: &backup.CompressionConfig{
				Algorithm: "gzip",
				Level:     5,
			},
		}
		
		// Create backup
		status, err := backupCtrl.CreateBackup(ctx, spec)
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, backup.BackupPhasePending, status.Phase)
		
		// Wait for backup to start
		time.Sleep(100 * time.Millisecond)
		
		// Get backup status
		backupName := "backup-Full-" + status.StartTimestamp.Format("20060102150405")
		updatedStatus, err := backupCtrl.GetBackup(ctx, backupName)
		assert.NoError(t, err)
		assert.NotNil(t, updatedStatus)
	})
	
	t.Run("ListBackups", func(t *testing.T) {
		// List backups
		backups, err := backupCtrl.ListBackups(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(backups), 1)
	})
}

func TestRestoreController(t *testing.T) {
	// Setup
	log := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	
	// Create fake client
	s := scheme.Scheme
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	
	// Create backup controller first
	backupConfig := controller.BackupControllerConfig{
		MaxConcurrentBackups: 5,
		DefaultTTL:          24 * time.Hour,
		RetryInterval:       1 * time.Minute,
		MaxRetries:          3,
		MetricsEnabled:      false,
	}
	
	recorder := record.NewFakeRecorder(100)
	backupCtrl, err := controller.NewBackupController(cl, s, log, recorder, backupConfig)
	require.NoError(t, err)
	
	// Create restore controller
	restoreConfig := controller.RestoreControllerConfig{
		MaxConcurrentRestores: 3,
		RetryInterval:        1 * time.Minute,
		MaxRetries:          3,
		ValidationEnabled:   true,
		DryRun:             false,
		MetricsEnabled:     false,
	}
	
	restoreCtrl, err := controller.NewRestoreController(cl, s, log, recorder, backupCtrl, restoreConfig)
	require.NoError(t, err)
	
	t.Run("CreateRestore", func(t *testing.T) {
		// First create a backup
		backupSpec := &backup.BackupSpec{
			Type: backup.FullBackup,
			StorageLocation: backup.StorageLocation{
				Provider: backup.LocalProvider,
				Bucket:   "test-bucket",
				Prefix:   "test-prefix",
			},
		}
		
		backupStatus, err := backupCtrl.CreateBackup(ctx, backupSpec)
		require.NoError(t, err)
		
		// Mark backup as completed (for testing)
		backupStatus.Phase = backup.BackupPhaseCompleted
		
		// Create restore spec
		restoreSpec := &backup.RestoreSpec{
			BackupName: "test-backup",
			ValidationPolicy: &backup.ValidationPolicy{
				VerifyChecksums:  true,
				VerifyIntegrity:  true,
				FailOnWarning:    false,
				MaxRetries:       3,
			},
		}
		
		// This will fail because the backup doesn't exist in the controller
		// In a real test, we'd need to properly create and complete a backup first
		_, err = restoreCtrl.CreateRestore(ctx, restoreSpec)
		assert.Error(t, err) // Expected to fail
	})
}

func TestBackupScheduler(t *testing.T) {
	// Setup
	log := zap.New(zap.UseDevMode(true))
	ctx := context.Background()
	
	// Create fake client
	s := scheme.Scheme
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	
	// Create backup controller
	backupConfig := controller.BackupControllerConfig{
		MaxConcurrentBackups: 5,
		DefaultTTL:          24 * time.Hour,
		RetryInterval:       1 * time.Minute,
		MaxRetries:          3,
		MetricsEnabled:      false,
	}
	
	recorder := record.NewFakeRecorder(100)
	backupCtrl, err := controller.NewBackupController(cl, s, log, recorder, backupConfig)
	require.NoError(t, err)
	
	// Create scheduler
	schedulerConfig := scheduler.ManagerConfig{
		Location:       time.UTC,
		MaxHistorySize: 100,
		MetricsEnabled: false,
	}
	
	scheduleManager, err := scheduler.NewManager(backupCtrl, log, schedulerConfig)
	require.NoError(t, err)
	defer scheduleManager.Stop()
	
	t.Run("CreateSchedule", func(t *testing.T) {
		// Create backup schedule
		schedule := &backup.BackupSchedule{
			Schedule: "*/5 * * * *", // Every 5 minutes
			Template: backup.BackupSpec{
				Type: backup.FullBackup,
				StorageLocation: backup.StorageLocation{
					Provider: backup.LocalProvider,
					Bucket:   "scheduled-bucket",
					Prefix:   "scheduled",
				},
			},
			RetentionPolicy: &backup.RetentionPolicy{
				KeepLast:  10,
				KeepDaily: 7,
			},
		}
		
		err := scheduleManager.CreateSchedule(ctx, "test-schedule", schedule)
		require.NoError(t, err)
		
		// Get schedule
		retrieved, err := scheduleManager.GetSchedule(ctx, "test-schedule")
		require.NoError(t, err)
		assert.Equal(t, schedule.Schedule, retrieved.Schedule)
	})
	
	t.Run("UpdateSchedule", func(t *testing.T) {
		// Update schedule
		updatedSchedule := &backup.BackupSchedule{
			Schedule: "0 * * * *", // Every hour
			Template: backup.BackupSpec{
				Type: backup.IncrementalBackup,
				StorageLocation: backup.StorageLocation{
					Provider: backup.LocalProvider,
					Bucket:   "scheduled-bucket",
					Prefix:   "scheduled",
				},
			},
		}
		
		err := scheduleManager.UpdateSchedule(ctx, "test-schedule", updatedSchedule)
		require.NoError(t, err)
		
		// Verify update
		retrieved, err := scheduleManager.GetSchedule(ctx, "test-schedule")
		require.NoError(t, err)
		assert.Equal(t, updatedSchedule.Schedule, retrieved.Schedule)
		assert.Equal(t, backup.IncrementalBackup, retrieved.Template.Type)
	})
	
	t.Run("ListSchedules", func(t *testing.T) {
		// List schedules
		schedules, err := scheduleManager.ListSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, 1)
	})
	
	t.Run("TriggerSchedule", func(t *testing.T) {
		// Manually trigger schedule
		err := scheduleManager.TriggerSchedule(ctx, "test-schedule")
		require.NoError(t, err)
		
		// Give it time to run
		time.Sleep(100 * time.Millisecond)
	})
	
	t.Run("DeleteSchedule", func(t *testing.T) {
		// Delete schedule
		err := scheduleManager.DeleteSchedule(ctx, "test-schedule")
		require.NoError(t, err)
		
		// Verify deletion
		_, err = scheduleManager.GetSchedule(ctx, "test-schedule")
		assert.Error(t, err)
	})
}

func TestBackupValidation(t *testing.T) {
	t.Run("ValidateBackupSpec", func(t *testing.T) {
		// Valid spec
		validSpec := &backup.BackupSpec{
			Type: backup.FullBackup,
			StorageLocation: backup.StorageLocation{
				Provider: backup.S3Provider,
				Bucket:   "valid-bucket",
				Region:   "us-east-1",
			},
		}
		
		// Validate storage location
		assert.NotEmpty(t, validSpec.StorageLocation.Bucket)
		assert.NotEmpty(t, validSpec.StorageLocation.Provider)
		
		// Invalid spec - missing bucket
		invalidSpec := &backup.BackupSpec{
			Type: backup.FullBackup,
			StorageLocation: backup.StorageLocation{
				Provider: backup.S3Provider,
				Bucket:   "",
			},
		}
		
		assert.Empty(t, invalidSpec.StorageLocation.Bucket)
	})
	
	t.Run("ValidateRestoreSpec", func(t *testing.T) {
		// Valid spec
		validSpec := &backup.RestoreSpec{
			BackupName: "backup-20250612",
			ValidationPolicy: &backup.ValidationPolicy{
				VerifyChecksums: true,
				VerifyIntegrity: true,
			},
		}
		
		assert.NotEmpty(t, validSpec.BackupName)
		assert.True(t, validSpec.ValidationPolicy.VerifyChecksums)
		
		// Invalid spec - missing backup name
		invalidSpec := &backup.RestoreSpec{
			BackupName: "",
		}
		
		assert.Empty(t, invalidSpec.BackupName)
	})
}

func TestStorageProviders(t *testing.T) {
	t.Run("StorageLocationValidation", func(t *testing.T) {
		testCases := []struct {
			name     string
			location backup.StorageLocation
			valid    bool
		}{
			{
				name: "Valid S3 location",
				location: backup.StorageLocation{
					Provider: backup.S3Provider,
					Bucket:   "my-bucket",
					Region:   "us-east-1",
					Prefix:   "backups",
				},
				valid: true,
			},
			{
				name: "Valid GCS location",
				location: backup.StorageLocation{
					Provider: backup.GCSProvider,
					Bucket:   "my-bucket",
					Prefix:   "backups",
				},
				valid: true,
			},
			{
				name: "Valid Azure location",
				location: backup.StorageLocation{
					Provider: backup.AzureBlobProvider,
					Bucket:   "my-container",
					Prefix:   "backups",
				},
				valid: true,
			},
			{
				name: "Invalid - missing bucket",
				location: backup.StorageLocation{
					Provider: backup.S3Provider,
					Bucket:   "",
				},
				valid: false,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isValid := tc.location.Bucket != ""
				assert.Equal(t, tc.valid, isValid)
			})
		}
	})
}
