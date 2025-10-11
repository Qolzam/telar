package postgresql

import (
	"os"
	"testing"
	"time"
)

// BenchmarkArrayOperations benchmarks array operations performance
func BenchmarkArrayOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("InOperator_SmallArray", func(b *testing.B) {
		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$in": []string{"golang", "database", "test"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("InOperator_LargeArray", func(b *testing.B) {
		// Create a large array of tags
		largeTags := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			largeTags[i] = "tag" + string(rune(i%26+'a'))
		}

		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$in": largeTags,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("AllOperator_SmallArray", func(b *testing.B) {
		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$all": []string{"golang", "database"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("AllOperator_LargeArray", func(b *testing.B) {
		// Create a large array of tags
		largeTags := make([]string, 100)
		for i := 0; i < 100; i++ {
			largeTags[i] = "tag" + string(rune(i%26+'a'))
		}

		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$all": largeTags,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkRegexOperations benchmarks regex operations performance
func BenchmarkRegexOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("Regex_CaseInsensitive", func(b *testing.B) {
		filter := map[string]interface{}{
			"body": map[string]interface{}{
				"$regex":   "test pattern",
				"$options": "i",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Regex_CaseSensitive", func(b *testing.B) {
		filter := map[string]interface{}{
			"body": map[string]interface{}{
				"$regex": "Test Pattern",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Regex_ComplexPattern", func(b *testing.B) {
		complexPattern := "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
		filter := map[string]interface{}{
			"body": map[string]interface{}{
				"$regex":   complexPattern,
				"$options": "i",
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkOrOperations benchmarks OR operations performance
func BenchmarkOrOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("Or_SmallConditions", func(b *testing.B) {
		filter := map[string]interface{}{
			"$or": []interface{}{
				map[string]interface{}{"status": "active"},
				map[string]interface{}{"status": "pending"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Or_LargeConditions", func(b *testing.B) {
		// Create many OR conditions
		orConditions := make([]interface{}, 50)
		for i := 0; i < 50; i++ {
			orConditions[i] = map[string]interface{}{
				"status": "status" + string(rune(i%26+'a')),
			}
		}

		filter := map[string]interface{}{
			"$or": orConditions,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Or_InterfaceSlice", func(b *testing.B) {
		filter := map[string]interface{}{
			"$or": []interface{}{
				map[string]interface{}{"status": "active"},
				map[string]interface{}{"status": "pending"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("Or_MapSlice", func(b *testing.B) {
		filter := map[string]interface{}{
			"$or": []map[string]interface{}{
				{"status": "active"},
				{"status": "pending"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkComplexOperations benchmarks complex operation combinations
func BenchmarkComplexOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("ComplexFilter_AllOperations", func(b *testing.B) {
		filter := map[string]interface{}{
			"objectId": map[string]interface{}{
				"$ne": "exclude-id",
			},
			"tags": map[string]interface{}{
				"$in": []string{"golang", "test", "database"},
			},
			"body": map[string]interface{}{
				"$regex":   "test",
				"$options": "i",
			},
			"score": map[string]interface{}{
				"$gte": 10,
			},
			"viewCount": map[string]interface{}{
				"$lt": 1000,
			},
			"deleted": false,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("ComplexFilter_WithOr", func(b *testing.B) {
		filter := map[string]interface{}{
			"$or": []interface{}{
				map[string]interface{}{"status": "active"},
				map[string]interface{}{"status": "pending"},
			},
			"tags": map[string]interface{}{
				"$all": []string{"golang", "test"},
			},
			"score": map[string]interface{}{
				"$gte": 50,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkNestedFields benchmarks nested field operations
func BenchmarkNestedFields(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("NestedField_Simple", func(b *testing.B) {
		filter := map[string]interface{}{
			"user.profile.name": "John Doe",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("NestedField_Deep", func(b *testing.B) {
		filter := map[string]interface{}{
			"user.profile.settings.theme": "dark",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("NestedField_VeryDeep", func(b *testing.B) {
		filter := map[string]interface{}{
			"level1.level2.level3.level4.level5.field": "value",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkUpdateOperations benchmarks update operations
func BenchmarkUpdateOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("SetOperation_Simple", func(b *testing.B) {
		update := map[string]interface{}{
			"$set": map[string]interface{}{
				"body": "Updated content",
				"tags": []string{"updated", "test"},
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildSetOperation(update["$set"].(map[string]interface{}))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("SetOperation_Complex", func(b *testing.B) {
		update := map[string]interface{}{
			"$set": map[string]interface{}{
				"body": "Updated content",
				"tags": []string{"updated", "test", "complex"},
				"user.profile.name": "John Doe",
				"user.profile.age": 30,
				"metadata.created_by": "system",
				"metadata.updated_at": time.Now().Unix(),
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildSetOperation(update["$set"].(map[string]interface{}))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("IncrementOperation_Simple", func(b *testing.B) {
		update := map[string]interface{}{
			"$inc": map[string]interface{}{
				"score": 10,
				"viewCount": 1,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildIncrementOperation(update["$inc"].(map[string]interface{}))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("IncrementOperation_Complex", func(b *testing.B) {
		update := map[string]interface{}{
			"$inc": map[string]interface{}{
				"score": 10,
				"viewCount": 1,
				"commentCounter": -1,
				"user.profile.views": 5,
				"metadata.visit_count": 1,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildIncrementOperation(update["$inc"].(map[string]interface{}))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("MixedOperation", func(b *testing.B) {
		update := map[string]interface{}{
			"$set": map[string]interface{}{
				"body": "Updated content",
				"tags": []string{"updated", "test"},
			},
			"$inc": map[string]interface{}{
				"score": 10,
				"viewCount": 1,
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildIncrementOperation(update["$inc"].(map[string]interface{}))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkMemoryAllocations benchmarks memory allocations
func BenchmarkMemoryAllocations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("ArrayOperation_Allocations", func(b *testing.B) {
		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$in": []string{"golang", "database", "test"},
			},
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("RegexOperation_Allocations", func(b *testing.B) {
		filter := map[string]interface{}{
			"body": map[string]interface{}{
				"$regex":   "test pattern",
				"$options": "i",
			},
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})

	b.Run("ComplexOperation_Allocations", func(b *testing.B) {
		filter := map[string]interface{}{
			"objectId": map[string]interface{}{
				"$ne": "exclude-id",
			},
			"tags": map[string]interface{}{
				"$in": []string{"golang", "test"},
			},
			"body": map[string]interface{}{
				"$regex":   "test",
				"$options": "i",
			},
			"score": map[string]interface{}{
				"$gte": 10,
			},
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	// Skip if not running database tests
	if os.Getenv("RUN_DB_TESTS") != "1" {
		b.Skip("RUN_DB_TESTS not set, skipping performance benchmark")
	}

	repo := &PostgreSQLRepository{}

	b.Run("Concurrent_ArrayOperations", func(b *testing.B) {
		filter := map[string]interface{}{
			"tags": map[string]interface{}{
				"$in": []string{"golang", "database", "test"},
			},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	})

	b.Run("Concurrent_RegexOperations", func(b *testing.B) {
		filter := map[string]interface{}{
			"body": map[string]interface{}{
				"$regex":   "test pattern",
				"$options": "i",
			},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	})

	b.Run("Concurrent_ComplexOperations", func(b *testing.B) {
		filter := map[string]interface{}{
			"objectId": map[string]interface{}{
				"$ne": "exclude-id",
			},
			"tags": map[string]interface{}{
				"$in": []string{"golang", "test"},
			},
			"body": map[string]interface{}{
				"$regex":   "test",
				"$options": "i",
			},
			"score": map[string]interface{}{
				"$gte": 10,
			},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	})
}
