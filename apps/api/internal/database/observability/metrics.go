// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package observability

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// MetricsCollector handles transaction metrics collection and reporting
type MetricsCollector struct {
	activeTransactions     int64
	totalTransactions      int64
	committedTransactions  int64
	rolledBackTransactions int64
	failedTransactions     int64
	totalDuration          int64 // nanoseconds
	mu                     sync.RWMutex
	transactionMetrics     map[string]*interfaces.TransactionMetrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		transactionMetrics: make(map[string]*interfaces.TransactionMetrics),
	}
}

// Global metrics collector instance
var globalMetrics = NewMetricsCollector()

// GetGlobalMetrics returns the global metrics collector
func GetGlobalMetrics() *MetricsCollector {
	return globalMetrics
}

// StartTransaction records the start of a new transaction
func (mc *MetricsCollector) StartTransaction(txID, databaseType string, config *interfaces.TransactionConfig) *interfaces.TransactionMetrics {
	atomic.AddInt64(&mc.activeTransactions, 1)
	atomic.AddInt64(&mc.totalTransactions, 1)

	metrics := &interfaces.TransactionMetrics{
		TransactionID:   txID,
		StartTime:       time.Now(),
		OperationsCount: 0,
		Status:          "active",
		DatabaseType:    databaseType,
	}

	mc.mu.Lock()
	mc.transactionMetrics[txID] = metrics
	mc.mu.Unlock()

	log.Info("Transaction started: %s (%s)", txID, databaseType)
	return metrics
}

// IncrementOperations increments the operation count for a transaction
func (mc *MetricsCollector) IncrementOperations(txID string) {
	mc.mu.RLock()
	if metrics, exists := mc.transactionMetrics[txID]; exists {
		atomic.AddInt64(&metrics.OperationsCount, 1)
	}
	mc.mu.RUnlock()
}

// CommitTransaction records a successful transaction commit
func (mc *MetricsCollector) CommitTransaction(txID string) {
	atomic.AddInt64(&mc.activeTransactions, -1)
	atomic.AddInt64(&mc.committedTransactions, 1)

	mc.mu.Lock()
	if metrics, exists := mc.transactionMetrics[txID]; exists {
		metrics.Status = "committed"
		metrics.Duration = time.Since(metrics.StartTime)
		atomic.AddInt64(&mc.totalDuration, int64(metrics.Duration))
		log.Info("Transaction committed: %s (duration: %v, operations: %d)",
			txID, metrics.Duration, metrics.OperationsCount)
	}
	mc.mu.Unlock()
}

// RollbackTransaction records a transaction rollback
func (mc *MetricsCollector) RollbackTransaction(txID string, err error) {
	atomic.AddInt64(&mc.activeTransactions, -1)
	atomic.AddInt64(&mc.rolledBackTransactions, 1)

	mc.mu.Lock()
	if metrics, exists := mc.transactionMetrics[txID]; exists {
		metrics.Status = "rolled_back"
		metrics.Duration = time.Since(metrics.StartTime)
		if err != nil {
			metrics.ErrorMessage = err.Error()
			if repoErr, ok := err.(*interfaces.RepositoryError); ok {
				metrics.ErrorCode = repoErr.Code
			}
		}
		log.Warn("Transaction rolled back: %s (duration: %v, error: %v)",
			txID, metrics.Duration, err)
	}
	mc.mu.Unlock()
}

// FailTransaction records a transaction failure
func (mc *MetricsCollector) FailTransaction(txID string, err error) {
	atomic.AddInt64(&mc.activeTransactions, -1)
	atomic.AddInt64(&mc.failedTransactions, 1)

	mc.mu.Lock()
	if metrics, exists := mc.transactionMetrics[txID]; exists {
		metrics.Status = "failed"
		metrics.Duration = time.Since(metrics.StartTime)
		if err != nil {
			metrics.ErrorMessage = err.Error()
			if repoErr, ok := err.(*interfaces.RepositoryError); ok {
				metrics.ErrorCode = repoErr.Code
			}
		}
		log.Error("Transaction failed: %s (duration: %v, error: %v)",
			txID, metrics.Duration, err)
	}
	mc.mu.Unlock()
}

// GetTransactionMetrics returns metrics for a specific transaction
func (mc *MetricsCollector) GetTransactionMetrics(txID string) *interfaces.TransactionMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, exists := mc.transactionMetrics[txID]; exists {
		// Return a copy to prevent concurrent access issues
		copy := *metrics
		return &copy
	}
	return nil
}

// GetGlobalStats returns global transaction statistics
func (mc *MetricsCollector) GetGlobalStats() map[string]interface{} {
	active := atomic.LoadInt64(&mc.activeTransactions)
	total := atomic.LoadInt64(&mc.totalTransactions)
	committed := atomic.LoadInt64(&mc.committedTransactions)
	rolledBack := atomic.LoadInt64(&mc.rolledBackTransactions)
	failed := atomic.LoadInt64(&mc.failedTransactions)
	totalDur := atomic.LoadInt64(&mc.totalDuration)

	var avgDuration time.Duration
	if completed := committed + rolledBack + failed; completed > 0 {
		avgDuration = time.Duration(totalDur / completed)
	}

	return map[string]interface{}{
		"active_transactions":      active,
		"total_transactions":       total,
		"committed_transactions":   committed,
		"rolled_back_transactions": rolledBack,
		"failed_transactions":      failed,
		"average_duration":         avgDuration,
		"success_rate":             float64(committed) / float64(total) * 100,
	}
}

// CleanupCompletedTransactions removes metrics for completed transactions older than the specified duration
func (mc *MetricsCollector) CleanupCompletedTransactions(olderThan time.Duration) {
	cutoff := time.Now().Add(-olderThan)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	for txID, metrics := range mc.transactionMetrics {
		if metrics.Status != "active" && metrics.StartTime.Before(cutoff) {
			delete(mc.transactionMetrics, txID)
		}
	}
}

// StartPeriodicCleanup starts a goroutine that periodically cleans up old transaction metrics
func (mc *MetricsCollector) StartPeriodicCleanup(ctx context.Context, interval, retentionPeriod time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mc.CleanupCompletedTransactions(retentionPeriod)
			}
		}
	}()
}
