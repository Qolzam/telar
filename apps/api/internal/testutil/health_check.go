package testutil

import (
	"context"
	"fmt"
	"time"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// HealthChecker provides database health checking functionality
type HealthChecker struct {
	config *platformconfig.Config
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *platformconfig.Config) *HealthChecker {
	return &HealthChecker{
		config: config,
	}
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	DatabaseType string
	IsHealthy    bool
	Error        error
	ResponseTime time.Duration
	Details      map[string]interface{}
}

// CheckDatabaseHealth performs a comprehensive health check on the specified database
func (hc *HealthChecker) CheckDatabaseHealth(ctx context.Context, dbType string) *HealthCheckResult {
	start := time.Now()
	result := &HealthCheckResult{
		DatabaseType: dbType,
		Details:      make(map[string]interface{}),
	}

	// Create base service with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	base, err := platform.NewBaseService(checkCtx, hc.config)
	if err != nil {
		result.Error = fmt.Errorf("failed to create base service: %w", err)
		result.IsHealthy = false
		result.ResponseTime = time.Since(start)
		return result
	}
	defer base.Close()

	// Perform ping test
	pingStart := time.Now()
	pingErr := base.HealthCheck(checkCtx)
	pingDuration := time.Since(pingStart)
	
	result.Details["ping_duration"] = pingDuration
	result.ResponseTime = time.Since(start)

	if pingErr != nil {
		result.Error = fmt.Errorf("database ping failed: %w", pingErr)
		result.IsHealthy = false
		return result
	}

	// Perform basic operation test with timeout
	opCtx, opCancel := context.WithTimeout(checkCtx, 10*time.Second)
	defer opCancel()
	
	opStart := time.Now()
	opErr := hc.performBasicOperation(opCtx, base, dbType)
	opDuration := time.Since(opStart)
	
	result.Details["operation_duration"] = opDuration

	if opErr != nil {
		result.Error = fmt.Errorf("basic operation failed: %w", opErr)
		result.IsHealthy = false
		return result
	}

	result.IsHealthy = true
	result.Details["status"] = "healthy"
	result.ResponseTime = time.Since(start)
	return result
}

// performBasicOperation performs a basic database operation to verify functionality
func (hc *HealthChecker) performBasicOperation(ctx context.Context, base *platform.BaseService, dbType string) error {
	// Create a test document/record with proper schema compliance
	now := time.Now().Unix()
	testData := map[string]interface{}{
		"objectId":     "health_check_" + dbType + "_" + fmt.Sprintf("%d", now), // Required for PostgreSQL
		"health_check": true,
		"timestamp":    now,
		"test_id":      "health_check_" + dbType,
		"created":      now,    // For PostgreSQL extractCommonFields
		"last_updated": now,    // For PostgreSQL extractCommonFields
	}

	// Save test data
	saveResult := <-base.Repository.Save(ctx, "health_check", testData)
	if saveResult.Error != nil {
		return fmt.Errorf("save operation failed: %w", saveResult.Error)
	}

	// Query test data with proper FindOptions
	limit := int64(1)
	skip := int64(0)
	queryResult := <-base.Repository.Find(ctx, "health_check", map[string]interface{}{
		"test_id": "health_check_" + dbType,
	}, &dbi.FindOptions{
		Limit: &limit,
		Skip:  &skip,
	})
	if queryResult.Error() != nil {
		return fmt.Errorf("query operation failed: %w", queryResult.Error())
	}

	// Verify we got results and can read at least one document
	defer queryResult.Close()
	
	// Try to read at least one result to verify the query worked
	if !queryResult.Next() {
		return fmt.Errorf("query returned no documents")
	}

	// Clean up test data
	deleteResult := <-base.Repository.Delete(ctx, "health_check", map[string]interface{}{
		"test_id": "health_check_" + dbType,
	})
	if deleteResult.Error != nil {
		// Log cleanup error but don't fail the health check
		fmt.Printf("Warning: Failed to cleanup health check data: %v\n", deleteResult.Error)
	}

	return nil
}

// CheckAllDatabases performs health checks on all configured databases
func (hc *HealthChecker) CheckAllDatabases(ctx context.Context) map[string]*HealthCheckResult {
	results := make(map[string]*HealthCheckResult)

	// Check MongoDB if configured
	if hc.config.Database.MongoDB.Host != "" {
		results[dbi.DatabaseTypeMongoDB] = hc.CheckDatabaseHealth(ctx, dbi.DatabaseTypeMongoDB)
	}

	// Check PostgreSQL if configured
	if hc.config.Database.Postgres.Host != "" {
		results[dbi.DatabaseTypePostgreSQL] = hc.CheckDatabaseHealth(ctx, dbi.DatabaseTypePostgreSQL)
	}

	return results
}

// WaitForHealthyDatabases waits for all databases to become healthy
func (hc *HealthChecker) WaitForHealthyDatabases(ctx context.Context, maxWaitTime time.Duration) error {
	deadline := time.Now().Add(maxWaitTime)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for databases to become healthy")
			}

			results := hc.CheckAllDatabases(ctx)
			allHealthy := true
			
			for dbType, result := range results {
				if !result.IsHealthy {
					fmt.Printf("Database %s is not healthy: %v\n", dbType, result.Error)
					allHealthy = false
				}
			}

			if allHealthy {
				fmt.Println("All databases are healthy")
				return nil
			}
		}
	}
}

// ValidateTestEnvironment validates that the test environment is ready
func (hc *HealthChecker) ValidateTestEnvironment(ctx context.Context) error {
	fmt.Println("Validating test environment...")
	
	// Check all databases with timeout
	validationCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	results := hc.CheckAllDatabases(validationCtx)
	
	var healthyCount int
	var totalCount int
	
	for dbType, result := range results {
		totalCount++
		if result.IsHealthy {
			healthyCount++
			fmt.Printf("✅ %s: Healthy (response time: %v)\n", dbType, result.ResponseTime)
		} else {
			fmt.Printf("❌ %s: Unhealthy - %v\n", dbType, result.Error)
		}
	}
	
	// If no databases are configured, that's also a problem
	if totalCount == 0 {
		return fmt.Errorf("no databases configured for testing")
	}
	
	// Allow partial health for testing - at least one database should be healthy
	if healthyCount == 0 {
		return fmt.Errorf("no healthy databases found")
	}
	
	if healthyCount < totalCount {
		fmt.Printf("⚠️  Warning: %d/%d databases are healthy\n", healthyCount, totalCount)
	}
	
	fmt.Printf("✅ Test environment validation complete: %d/%d databases healthy\n", healthyCount, totalCount)
	return nil
}
