package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/gunjanjp/gunj-operator/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// setupRedisContainer starts a Redis container for testing
func setupRedisContainer(t *testing.T) (string, func()) {
	ctx := context.Background()
	
	req := testcontainers.ContainerRequest{
		Image:        "redis:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	
	host, err := container.Host(ctx)
	require.NoError(t, err)
	
	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)
	
	redisAddr := host + ":" + port.Port()
	
	cleanup := func() {
		_ = container.Terminate(ctx)
	}
	
	return redisAddr, cleanup
}

func TestRedisClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	redisAddr, cleanup := setupRedisContainer(t)
	defer cleanup()
	
	logger := zap.New()
	log.SetLogger(logger)
	
	cfg := cache.DefaultConfig()
	cfg.Addresses = []string{redisAddr}
	
	client, err := cache.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()
	
	ctx := context.Background()
	
	t.Run("Basic Operations", func(t *testing.T) {
		// Test Set and Get
		err := client.Set(ctx, "test:key", "test-value", time.Minute)
		assert.NoError(t, err)
		
		val, err := client.Get(ctx, "test:key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", val)
		
		// Test Exists
		exists, err := client.Exists(ctx, "test:key")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)
		
		// Test Delete
		err = client.Delete(ctx, "test:key")
		assert.NoError(t, err)
		
		exists, err = client.Exists(ctx, "test:key")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})
	
	t.Run("Hash Operations", func(t *testing.T) {
		// Test HSet
		err := client.HSet(ctx, "test:hash", "field1", "value1", "field2", "value2")
		assert.NoError(t, err)
		
		// Test HGet
		val, err := client.HGet(ctx, "test:hash", "field1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)
		
		// Test HGetAll
		all, err := client.HGetAll(ctx, "test:hash")
		assert.NoError(t, err)
		assert.Equal(t, map[string]string{
			"field1": "value1",
			"field2": "value2",
		}, all)
	})
	
	t.Run("List Operations", func(t *testing.T) {
		// Test LPush
		err := client.LPush(ctx, "test:list", "item1", "item2", "item3")
		assert.NoError(t, err)
		
		// Test RPop
		val, err := client.RPop(ctx, "test:list")
		assert.NoError(t, err)
		assert.Equal(t, "item1", val)
	})
}

func TestRateLimiter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	redisAddr, cleanup := setupRedisContainer(t)
	defer cleanup()
	
	logger := zap.New()
	cfg := cache.DefaultConfig()
	cfg.Addresses = []string{redisAddr}
	
	client, err := cache.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()
	
	rateLimiter := cache.NewRateLimiter(client, "ratelimit")
	ctx := context.Background()
	
	t.Run("Allow Within Limit", func(t *testing.T) {
		identifier := "user:123"
		limit := 5
		window := time.Minute
		
		// Should allow first requests
		for i := 0; i < limit; i++ {
			allowed, remaining, err := rateLimiter.Allow(ctx, identifier, limit, window)
			assert.NoError(t, err)
			assert.True(t, allowed)
			assert.Equal(t, limit-i-1, remaining)
		}
		
		// Should deny when limit reached
		allowed, remaining, err := rateLimiter.Allow(ctx, identifier, limit, window)
		assert.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, 0, remaining)
	})
	
	t.Run("Reset Rate Limit", func(t *testing.T) {
		identifier := "user:456"
		limit := 3
		window := time.Minute
		
		// Use up the limit
		for i := 0; i < limit; i++ {
			_, _, err := rateLimiter.Allow(ctx, identifier, limit, window)
			assert.NoError(t, err)
		}
		
		// Should be denied
		allowed, _, err := rateLimiter.Allow(ctx, identifier, limit, window)
		assert.NoError(t, err)
		assert.False(t, allowed)
		
		// Reset
		err = rateLimiter.Reset(ctx, identifier)
		assert.NoError(t, err)
		
		// Should be allowed again
		allowed, _, err = rateLimiter.Allow(ctx, identifier, limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}

func TestSessionStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	redisAddr, cleanup := setupRedisContainer(t)
	defer cleanup()
	
	logger := zap.New()
	cfg := cache.DefaultConfig()
	cfg.Addresses = []string{redisAddr}
	
	client, err := cache.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()
	
	sessionStore := cache.NewSessionStore(client, "session", time.Hour)
	ctx := context.Background()
	
	t.Run("Create and Get Session", func(t *testing.T) {
		session := &cache.Session{
			ID:        "sess-123",
			UserID:    "user-456",
			Username:  "testuser",
			Roles:     []string{"admin", "user"},
			CreatedAt: time.Now(),
			Data:      map[string]interface{}{"theme": "dark"},
		}
		
		// Create session
		err := sessionStore.Create(ctx, session)
		assert.NoError(t, err)
		
		// Get session
		retrieved, err := sessionStore.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.Username, retrieved.Username)
	})
	
	t.Run("Refresh Session", func(t *testing.T) {
		session := &cache.Session{
			ID:        "sess-789",
			UserID:    "user-789",
			Username:  "testuser2",
			CreatedAt: time.Now(),
		}
		
		err := sessionStore.Create(ctx, session)
		assert.NoError(t, err)
		
		// Refresh
		err = sessionStore.Refresh(ctx, session.ID)
		assert.NoError(t, err)
		
		// Should still exist
		retrieved, err := sessionStore.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
	
	t.Run("Delete Session", func(t *testing.T) {
		session := &cache.Session{
			ID:        "sess-delete",
			UserID:    "user-delete",
			Username:  "deleteuser",
			CreatedAt: time.Now(),
		}
		
		err := sessionStore.Create(ctx, session)
		assert.NoError(t, err)
		
		// Delete
		err = sessionStore.Delete(ctx, session.ID)
		assert.NoError(t, err)
		
		// Should not exist
		_, err = sessionStore.Get(ctx, session.ID)
		assert.Error(t, err)
	})
}

func TestAPICache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	redisAddr, cleanup := setupRedisContainer(t)
	defer cleanup()
	
	logger := zap.New()
	cfg := cache.DefaultConfig()
	cfg.Addresses = []string{redisAddr}
	
	client, err := cache.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()
	
	apiCache := cache.NewAPICache(client, "api")
	ctx := context.Background()
	
	t.Run("Cache API Response", func(t *testing.T) {
		key := "GET:/api/v1/platforms"
		data := []byte(`{"platforms": [{"name": "test"}]}`)
		
		// Set cache
		err := apiCache.Set(ctx, key, data, time.Minute)
		assert.NoError(t, err)
		
		// Get cache
		cached, err := apiCache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, data, cached)
		
		// Invalidate
		err = apiCache.Invalidate(ctx, key)
		assert.NoError(t, err)
		
		// Should not exist
		_, err = apiCache.Get(ctx, key)
		assert.Error(t, err)
	})
}
