/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
)

var _ = Describe("Schema Evolution Tracker", func() {
	var (
		tracker *migration.SchemaEvolutionTracker
		logger  = logf.Log.WithName("test")
	)
	
	BeforeEach(func() {
		tracker = migration.NewSchemaEvolutionTracker(logger)
	})
	
	Describe("Version Information", func() {
		It("should return version info for known versions", func() {
			info, err := tracker.GetVersionInfo("v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(info).NotTo(BeNil())
			Expect(info.Version).To(Equal("v1beta1"))
			Expect(info.Features).To(ContainElement("Multi-cluster support"))
		})
		
		It("should return error for unknown versions", func() {
			_, err := tracker.GetVersionInfo("v2")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version not found"))
		})
		
		It("should check if version is supported", func() {
			Expect(tracker.IsVersionSupported("v1alpha1")).To(BeTrue())
			Expect(tracker.IsVersionSupported("v1beta1")).To(BeTrue())
			Expect(tracker.IsVersionSupported("v2")).To(BeFalse())
		})
		
		It("should return all supported versions", func() {
			versions := tracker.GetSupportedVersions()
			Expect(versions).To(ContainElements("v1alpha1", "v1beta1"))
		})
	})
	
	Describe("Migration Paths", func() {
		It("should return direct migration path", func() {
			path, err := tracker.GetMigrationPath("v1alpha1", "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).NotTo(BeNil())
			Expect(path.Direct).To(BeTrue())
			Expect(path.Complexity).To(Equal(migration.ComplexityModerate))
			Expect(path.DataLossRisk).To(BeFalse())
		})
		
		It("should return backward migration path with data loss warning", func() {
			path, err := tracker.GetMigrationPath("v1beta1", "v1alpha1")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).NotTo(BeNil())
			Expect(path.DataLossRisk).To(BeTrue())
			Expect(path.RequiresManual).To(BeTrue())
			Expect(path.Complexity).To(Equal(migration.ComplexityComplex))
		})
		
		It("should return error for non-existent migration path", func() {
			_, err := tracker.GetMigrationPath("v1alpha1", "v3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no migration path found"))
		})
	})
	
	Describe("Field Changes", func() {
		It("should return field changes between versions", func() {
			changes, err := tracker.GetFieldChanges("v1alpha1", "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(changes).NotTo(BeEmpty())
			
			// Check specific field change
			monitoringChange, exists := changes["spec.monitoring"]
			Expect(exists).To(BeTrue())
			Expect(monitoringChange.Type).To(Equal(migration.ChangeTypeRenamed))
			Expect(monitoringChange.OldPath).To(Equal("spec.monitoring"))
			Expect(monitoringChange.NewPath).To(Equal("spec.observability"))
		})
		
		It("should identify deprecated fields", func() {
			deprecated, err := tracker.GetDeprecatedFields("v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(deprecated).To(ContainElements(
				"spec.legacyConfig field",
				"spec.deprecatedOptions field",
			))
		})
	})
	
	Describe("Migration Recording", func() {
		It("should record migration events", func() {
			resource := types.NamespacedName{Name: "test", Namespace: "default"}
			
			// Record migration
			tracker.RecordMigration("v1alpha1", "v1beta1", resource)
			
			// Get analytics
			analytics := tracker.GetMigrationAnalytics()
			Expect(analytics.TotalMigrations).To(Equal(int64(1)))
			Expect(analytics.MigrationsByVersion["v1alpha1->v1beta1"]).To(Equal(int64(1)))
			Expect(analytics.CommonPaths["v1alpha1->v1beta1"]).To(Equal(int64(1)))
		})
		
		It("should record successful migrations with duration", func() {
			tracker.RecordMigrationSuccess("v1alpha1", "v1beta1", 5*time.Second)
			
			analytics := tracker.GetMigrationAnalytics()
			Expect(analytics.SuccessfulMigrations).To(Equal(int64(1)))
			Expect(analytics.AverageDuration).To(Equal(5 * time.Second))
		})
		
		It("should record failed migrations", func() {
			err := fmt.Errorf("test error")
			tracker.RecordMigrationFailure("v1alpha1", "v1beta1", err)
			
			analytics := tracker.GetMigrationAnalytics()
			Expect(analytics.FailedMigrations).To(Equal(int64(1)))
			Expect(analytics.ErrorPatterns).NotTo(BeEmpty())
		})
		
		It("should calculate average duration correctly", func() {
			tracker.RecordMigrationSuccess("v1alpha1", "v1beta1", 4*time.Second)
			tracker.RecordMigrationSuccess("v1alpha1", "v1beta1", 6*time.Second)
			
			analytics := tracker.GetMigrationAnalytics()
			Expect(analytics.AverageDuration).To(Equal(5 * time.Second))
		})
	})
	
	Describe("Analytics Export", func() {
		It("should export analytics as JSON", func() {
			// Record some data
			tracker.RecordMigration("v1alpha1", "v1beta1", types.NamespacedName{Name: "test", Namespace: "default"})
			tracker.RecordMigrationSuccess("v1alpha1", "v1beta1", 3*time.Second)
			
			// Export
			data, err := tracker.ExportAnalytics()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeEmpty())
			
			// Verify JSON structure
			Expect(string(data)).To(ContainSubstring("TotalMigrations"))
			Expect(string(data)).To(ContainSubstring("SuccessfulMigrations"))
			Expect(string(data)).To(ContainSubstring("MigrationsByVersion"))
		})
	})
})
