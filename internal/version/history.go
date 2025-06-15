// Package version provides version history tracking
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// HistoryTracker tracks version history for components
type HistoryTracker struct {
	mu       sync.RWMutex
	history  map[string][]VersionHistoryEntry
	client   client.Client
	storage  HistoryStorage
	maxItems int
	log      logr.Logger
}

// VersionHistoryEntry represents a version change event
type VersionHistoryEntry struct {
	ID            string                 `json:"id"`
	Platform      PlatformReference      `json:"platform"`
	Component     string                 `json:"component"`
	FromVersion   string                 `json:"fromVersion"`
	ToVersion     string                 `json:"toVersion"`
	Action        VersionAction          `json:"action"`
	Status        VersionStatus          `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty"`
	Duration      *time.Duration         `json:"duration,omitempty"`
	PerformedBy   string                 `json:"performedBy,omitempty"`
	Reason        string                 `json:"reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// PlatformReference identifies a platform
type PlatformReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

// VersionAction represents the type of version change
type VersionAction string

const (
	VersionActionInstall   VersionAction = "install"
	VersionActionUpgrade   VersionAction = "upgrade"
	VersionActionDowngrade VersionAction = "downgrade"
	VersionActionRollback  VersionAction = "rollback"
	VersionActionUninstall VersionAction = "uninstall"
)

// VersionStatus represents the status of a version change
type VersionStatus string

const (
	VersionStatusPending    VersionStatus = "pending"
	VersionStatusInProgress VersionStatus = "in_progress"
	VersionStatusSucceeded  VersionStatus = "succeeded"
	VersionStatusFailed     VersionStatus = "failed"
	VersionStatusRolledBack VersionStatus = "rolled_back"
)

// HistoryStorage interface for persistent storage
type HistoryStorage interface {
	Save(ctx context.Context, entries []VersionHistoryEntry) error
	Load(ctx context.Context, filter HistoryFilter) ([]VersionHistoryEntry, error)
	Delete(ctx context.Context, filter HistoryFilter) error
}

// HistoryFilter filters history entries
type HistoryFilter struct {
	Platform  *PlatformReference
	Component string
	Action    VersionAction
	Status    VersionStatus
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// NewHistoryTracker creates a new history tracker
func NewHistoryTracker(client client.Client, storage HistoryStorage, maxItems int) *HistoryTracker {
	if maxItems <= 0 {
		maxItems = 1000
	}

	return &HistoryTracker{
		history:  make(map[string][]VersionHistoryEntry),
		client:   client,
		storage:  storage,
		maxItems: maxItems,
		log:      log.Log.WithName("history-tracker"),
	}
}

// RecordChange records a version change
func (ht *HistoryTracker) RecordChange(ctx context.Context, entry VersionHistoryEntry) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%s-%s-%d", entry.Component, entry.ToVersion, time.Now().UnixNano())
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Add to in-memory history
	key := ht.getHistoryKey(entry.Platform)
	ht.history[key] = append(ht.history[key], entry)

	// Limit history size
	if len(ht.history[key]) > ht.maxItems {
		ht.history[key] = ht.history[key][len(ht.history[key])-ht.maxItems:]
	}

	// Persist to storage if available
	if ht.storage != nil {
		if err := ht.storage.Save(ctx, []VersionHistoryEntry{entry}); err != nil {
			ht.log.Error(err, "Failed to persist history entry", "entry", entry.ID)
		}
	}

	ht.log.Info("Recorded version change",
		"platform", entry.Platform.Name,
		"component", entry.Component,
		"from", entry.FromVersion,
		"to", entry.ToVersion,
		"action", entry.Action,
	)

	return nil
}

// UpdateStatus updates the status of a history entry
func (ht *HistoryTracker) UpdateStatus(ctx context.Context, entryID string, status VersionStatus, error string) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	// Find and update the entry
	for _, entries := range ht.history {
		for i, entry := range entries {
			if entry.ID == entryID {
				entries[i].Status = status
				entries[i].Error = error
				
				if status == VersionStatusSucceeded || status == VersionStatusFailed {
					now := time.Now()
					entries[i].CompletedAt = &now
					duration := now.Sub(entry.Timestamp)
					entries[i].Duration = &duration
				}

				// Persist update
				if ht.storage != nil {
					if err := ht.storage.Save(ctx, []VersionHistoryEntry{entries[i]}); err != nil {
						ht.log.Error(err, "Failed to persist status update", "entry", entryID)
					}
				}

				return nil
			}
		}
	}

	return fmt.Errorf("history entry %s not found", entryID)
}

// GetHistory retrieves version history
func (ht *HistoryTracker) GetHistory(filter HistoryFilter) []VersionHistoryEntry {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	var results []VersionHistoryEntry

	// If platform specified, only check that platform's history
	if filter.Platform != nil {
		key := ht.getHistoryKey(*filter.Platform)
		if entries, exists := ht.history[key]; exists {
			results = ht.filterEntries(entries, filter)
		}
	} else {
		// Check all platforms
		for _, entries := range ht.history {
			results = append(results, ht.filterEntries(entries, filter)...)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply limit
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results
}

// GetComponentHistory retrieves history for a specific component
func (ht *HistoryTracker) GetComponentHistory(platform PlatformReference, component string, limit int) []VersionHistoryEntry {
	return ht.GetHistory(HistoryFilter{
		Platform:  &platform,
		Component: component,
		Limit:     limit,
	})
}

// GetLatestVersion gets the latest version for a component
func (ht *HistoryTracker) GetLatestVersion(platform PlatformReference, component string) (string, bool) {
	history := ht.GetComponentHistory(platform, component, 1)
	if len(history) > 0 && history[0].Status == VersionStatusSucceeded {
		return history[0].ToVersion, true
	}
	return "", false
}

// GetUpgradeHistory retrieves upgrade history
func (ht *HistoryTracker) GetUpgradeHistory(platform PlatformReference, since time.Time) []VersionHistoryEntry {
	return ht.GetHistory(HistoryFilter{
		Platform: &platform,
		Action:   VersionActionUpgrade,
		Since:    &since,
	})
}

// GetFailedChanges retrieves failed version changes
func (ht *HistoryTracker) GetFailedChanges(since time.Time) []VersionHistoryEntry {
	return ht.GetHistory(HistoryFilter{
		Status: VersionStatusFailed,
		Since:  &since,
	})
}

// GenerateReport generates a version history report
func (ht *HistoryTracker) GenerateReport(filter HistoryFilter) (*VersionHistoryReport, error) {
	entries := ht.GetHistory(filter)

	report := &VersionHistoryReport{
		GeneratedAt: time.Now(),
		Filter:      filter,
		TotalCount:  len(entries),
		Entries:     entries,
	}

	// Calculate statistics
	stats := &VersionHistoryStats{
		TotalChanges:     len(entries),
		ByAction:         make(map[VersionAction]int),
		ByStatus:         make(map[VersionStatus]int),
		ByComponent:      make(map[string]int),
		AverageDuration:  time.Duration(0),
		SuccessRate:      0,
	}

	var totalDuration time.Duration
	var durationCount int
	successCount := 0

	for _, entry := range entries {
		// Count by action
		stats.ByAction[entry.Action]++

		// Count by status
		stats.ByStatus[entry.Status]++

		// Count by component
		stats.ByComponent[entry.Component]++

		// Calculate duration stats
		if entry.Duration != nil {
			totalDuration += *entry.Duration
			durationCount++
		}

		// Count successes
		if entry.Status == VersionStatusSucceeded {
			successCount++
		}
	}

	// Calculate averages
	if durationCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(durationCount)
	}

	if len(entries) > 0 {
		stats.SuccessRate = float64(successCount) / float64(len(entries)) * 100
	}

	report.Statistics = stats

	return report, nil
}

// LoadFromStorage loads history from persistent storage
func (ht *HistoryTracker) LoadFromStorage(ctx context.Context, filter HistoryFilter) error {
	if ht.storage == nil {
		return fmt.Errorf("no storage configured")
	}

	entries, err := ht.storage.Load(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	// Group by platform
	for _, entry := range entries {
		key := ht.getHistoryKey(entry.Platform)
		ht.history[key] = append(ht.history[key], entry)
	}

	ht.log.Info("Loaded history from storage", "entries", len(entries))

	return nil
}

// Cleanup removes old history entries
func (ht *HistoryTracker) Cleanup(ctx context.Context, olderThan time.Time) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	totalRemoved := 0

	// Clean in-memory history
	for key, entries := range ht.history {
		var kept []VersionHistoryEntry
		for _, entry := range entries {
			if entry.Timestamp.After(olderThan) {
				kept = append(kept, entry)
			} else {
				totalRemoved++
			}
		}
		ht.history[key] = kept
	}

	// Clean persistent storage
	if ht.storage != nil {
		filter := HistoryFilter{
			Until: &olderThan,
		}
		if err := ht.storage.Delete(ctx, filter); err != nil {
			return fmt.Errorf("failed to cleanup storage: %w", err)
		}
	}

	ht.log.Info("Cleaned up old history entries", "removed", totalRemoved)

	return nil
}

// Private helper methods

func (ht *HistoryTracker) getHistoryKey(platform PlatformReference) string {
	return fmt.Sprintf("%s/%s", platform.Namespace, platform.Name)
}

func (ht *HistoryTracker) filterEntries(entries []VersionHistoryEntry, filter HistoryFilter) []VersionHistoryEntry {
	var filtered []VersionHistoryEntry

	for _, entry := range entries {
		// Filter by component
		if filter.Component != "" && entry.Component != filter.Component {
			continue
		}

		// Filter by action
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}

		// Filter by status
		if filter.Status != "" && entry.Status != filter.Status {
			continue
		}

		// Filter by time range
		if filter.Since != nil && entry.Timestamp.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && entry.Timestamp.After(*filter.Until) {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// VersionHistoryReport represents a history report
type VersionHistoryReport struct {
	GeneratedAt time.Time            `json:"generatedAt"`
	Filter      HistoryFilter        `json:"filter"`
	TotalCount  int                  `json:"totalCount"`
	Entries     []VersionHistoryEntry `json:"entries"`
	Statistics  *VersionHistoryStats  `json:"statistics"`
}

// VersionHistoryStats contains statistics about version changes
type VersionHistoryStats struct {
	TotalChanges    int                     `json:"totalChanges"`
	ByAction        map[VersionAction]int   `json:"byAction"`
	ByStatus        map[VersionStatus]int   `json:"byStatus"`
	ByComponent     map[string]int          `json:"byComponent"`
	AverageDuration time.Duration           `json:"averageDuration"`
	SuccessRate     float64                 `json:"successRate"`
}

// InMemoryHistoryStorage provides in-memory storage for testing
type InMemoryHistoryStorage struct {
	mu      sync.RWMutex
	entries []VersionHistoryEntry
}

// NewInMemoryHistoryStorage creates a new in-memory storage
func NewInMemoryHistoryStorage() *InMemoryHistoryStorage {
	return &InMemoryHistoryStorage{
		entries: []VersionHistoryEntry{},
	}
}

// Save saves entries to memory
func (s *InMemoryHistoryStorage) Save(ctx context.Context, entries []VersionHistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entries...)
	return nil
}

// Load loads entries from memory
func (s *InMemoryHistoryStorage) Load(ctx context.Context, filter HistoryFilter) ([]VersionHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Simple filtering - in production, this would be more sophisticated
	var results []VersionHistoryEntry
	for _, entry := range s.entries {
		results = append(results, entry)
	}

	return results, nil
}

// Delete deletes entries from memory
func (s *InMemoryHistoryStorage) Delete(ctx context.Context, filter HistoryFilter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var kept []VersionHistoryEntry
	for _, entry := range s.entries {
		if filter.Until == nil || entry.Timestamp.After(*filter.Until) {
			kept = append(kept, entry)
		}
	}

	s.entries = kept
	return nil
}
