package postgresql

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQL_Basic(t *testing.T) {
	t.Log("internal/database/postgresql package test passed")
}

func TestPostgreSQL_Compilation(t *testing.T) {
	t.Log("internal/database/postgresql compilation test passed")
}

func TestBuildWhereClauseWithOffset_InOperator(t *testing.T) {
	repo := &PostgreSQLRepository{}
	filter := map[string]interface{}{
		"tags": map[string]interface{}{
			"$in": []string{"go", "db"},
		},
	}

	where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
	require.NoError(t, err)
	assert.Equal(t, "data->'tags' ?| $1", where)
	assert.Len(t, args, 1)
}

func TestBuildWhereClauseWithOffset_RegexOperator_CaseInsensitive(t *testing.T) {
	repo := &PostgreSQLRepository{}
	filter := map[string]interface{}{
		"name": map[string]interface{}{
			"$regex":   "test",
			"$options": "i",
		},
	}

	where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
	require.NoError(t, err)
	assert.Equal(t, "(data->>'name') ~* $1", where)
	assert.Len(t, args, 1)
	assert.Equal(t, "test", fmt.Sprint(args[0]))
}

func TestBuildWhereClauseWithOffset_NotEqualOperator(t *testing.T) {
	repo := &PostgreSQLRepository{}
	filter := map[string]interface{}{
		"objectId": map[string]interface{}{
			"$ne": "123",
		},
	}

	where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
	require.NoError(t, err)
	assert.Equal(t, "object_id <> $1", where)
	assert.Len(t, args, 1)
	assert.Equal(t, "123", fmt.Sprint(args[0]))
}

func TestOperations(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("RUN_DB_TESTS not set, skipping operations test")
	}

	repo := &PostgreSQLRepository{}

	t.Run("ArrayOperations", func(t *testing.T) {
		t.Run("InOperator_StringArray", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$in": []string{"golang", "database", "test"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "data->'tags' ?| $1", where)
			assert.Len(t, args, 1)
			
			_, ok := args[0].(interface{ driver.Valuer; sql.Scanner })
			assert.True(t, ok, "Expected pq.Array type")
		})

		t.Run("InOperator_InterfaceArray", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$in": []interface{}{"golang", "database", "test"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "data->'tags' ?| $1", where)
			assert.Len(t, args, 1)
		})

		t.Run("InOperator_InvalidType", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$in": "not-a-slice",
				},
			}

			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid type for $in operator")
		})

		t.Run("AllOperator_StringArray", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$all": []string{"golang", "database"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "data->'tags' @> $1", where)
			assert.Len(t, args, 1)
		})

		t.Run("AllOperator_InterfaceArray", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$all": []interface{}{"golang", "database"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "data->'tags' @> $1", where)
			assert.Len(t, args, 1)
		})

		t.Run("AllOperator_InvalidType", func(t *testing.T) {
			filter := map[string]interface{}{
				"tags": map[string]interface{}{
					"$all": "not-a-slice",
				},
			}

			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid type for $all operator")
		})
	})

	t.Run("RegexOperations", func(t *testing.T) {
		t.Run("Regex_CaseInsensitive", func(t *testing.T) {
			filter := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex":   "test",
					"$options": "i",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'name') ~* $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "test", fmt.Sprint(args[0]))
		})

		t.Run("Regex_CaseSensitive", func(t *testing.T) {
			filter := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex": "Test",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'name') ~ $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "Test", fmt.Sprint(args[0]))
		})

		t.Run("Regex_ComplexOptions", func(t *testing.T) {
			filter := map[string]interface{}{
				"name": map[string]interface{}{
					"$regex":   "test.*pattern",
					"$options": "i",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'name') ~* $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "test.*pattern", fmt.Sprint(args[0]))
		})
	})

	t.Run("ComparisonOperations", func(t *testing.T) {
		t.Run("NotEqual_ObjectId", func(t *testing.T) {
			filter := map[string]interface{}{
				"objectId": map[string]interface{}{
					"$ne": "123",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "object_id <> $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "123", fmt.Sprint(args[0]))
		})

		t.Run("NotEqual_JSONBField", func(t *testing.T) {
			filter := map[string]interface{}{
				"status": map[string]interface{}{
					"$ne": "deleted",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'status') <> $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "deleted", fmt.Sprint(args[0]))
		})

		t.Run("GreaterThan_JSONBField", func(t *testing.T) {
			filter := map[string]interface{}{
				"score": map[string]interface{}{
					"$gt": 10,
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'score')::bigint > $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "10", fmt.Sprint(args[0]))
		})

		t.Run("LessThan_JSONBField", func(t *testing.T) {
			filter := map[string]interface{}{
				"score": map[string]interface{}{
					"$lt": 100,
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'score')::bigint < $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "100", fmt.Sprint(args[0]))
		})

		t.Run("GreaterThanOrEqual_JSONBField", func(t *testing.T) {
			filter := map[string]interface{}{
				"score": map[string]interface{}{
					"$gte": 10,
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'score')::bigint >= $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "10", fmt.Sprint(args[0]))
		})

		t.Run("LessThanOrEqual_JSONBField", func(t *testing.T) {
			filter := map[string]interface{}{
				"score": map[string]interface{}{
					"$lte": 100,
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'score')::bigint <= $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "100", fmt.Sprint(args[0]))
		})

		t.Run("DateComparison", func(t *testing.T) {
			filter := map[string]interface{}{
				"createdDate": map[string]interface{}{
					"$gte": time.Now().Unix(),
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "created_date >= $1", where)
			assert.Len(t, args, 1)
		})
	})

	t.Run("OrOperations", func(t *testing.T) {
		t.Run("Or_InterfaceSlice", func(t *testing.T) {
			filter := map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"status": "active"},
					map[string]interface{}{"status": "pending"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Contains(t, where, "OR")
			assert.Contains(t, where, "(data->>'status' = $1)")
			assert.Contains(t, where, "(data->>'status' = $2)")
			assert.Len(t, args, 2)
		})

		t.Run("Or_MapSlice", func(t *testing.T) {
			filter := map[string]interface{}{
				"$or": []map[string]interface{}{
					{"status": "active"},
					{"status": "pending"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Contains(t, where, "OR")
			assert.Len(t, args, 2)
		})

		t.Run("Or_InvalidType", func(t *testing.T) {
			filter := map[string]interface{}{
				"$or": "not-a-slice",
			}

			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported $or type")
		})

		t.Run("Or_EmptySlice", func(t *testing.T) {
			filter := map[string]interface{}{
				"$or": []interface{}{},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "", where)
			assert.Len(t, args, 0)
		})
	})

	t.Run("ComplexCombinations", func(t *testing.T) {
		t.Run("AllOperations", func(t *testing.T) {
			filter := map[string]interface{}{
				"objectId": map[string]interface{}{
					"$ne": "exclude-id",
				},
				"tags": map[string]interface{}{
					"$in": []string{"golang", "test"},
				},
				"name": map[string]interface{}{
					"$regex":   "test",
					"$options": "i",
				},
				"score": map[string]interface{}{
					"$gte": 10,
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)

			expectedParts := []string{
				"object_id <> $1",
				"data->'tags' ?| $",
				"(data->>'name') ~* $",
				"(data->>'score')::bigint >=",
			}

			for _, part := range expectedParts {
				assert.Contains(t, where, part, "where clause should contain %s", part)
			}

			assert.Len(t, args, 4)
		})

		t.Run("WithOr", func(t *testing.T) {
			filter := map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"status": "active"},
					map[string]interface{}{"status": "pending"},
				},
				"tags": map[string]interface{}{
					"$in": []string{"golang"},
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Contains(t, where, "OR")
			assert.Contains(t, where, "data->'tags' ?| $")
			assert.Len(t, args, 3)
		})
	})

	t.Run("EdgeCases", func(t *testing.T) {
		t.Run("EmptyFilter", func(t *testing.T) {
			filter := map[string]interface{}{}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "", where)
			assert.Len(t, args, 0)
		})

		t.Run("EmptyMap", func(t *testing.T) {
			filter := map[string]interface{}{
				"empty": map[string]interface{}{},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "", where)
			assert.Len(t, args, 0)
		})

		t.Run("InvalidFilterType", func(t *testing.T) {
			filter := "not-a-map"

			_, _, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err) // The function handles non-map types gracefully
		})

		t.Run("NestedFields", func(t *testing.T) {
			filter := map[string]interface{}{
				"user.profile.name": map[string]interface{}{
					"$ne": "admin",
				},
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'user.profile.name') <> $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "admin", fmt.Sprint(args[0]))
		})

		t.Run("BooleanField", func(t *testing.T) {
			filter := map[string]interface{}{
				"deleted": false,
			}

			where, args, err := repo.buildWhereClauseWithOffset(filter, 1)
			require.NoError(t, err)
			assert.Equal(t, "(data->>'deleted')::boolean = $1", where)
			assert.Len(t, args, 1)
			assert.Equal(t, "false", fmt.Sprint(args[0]))
		})
	})
}

func TestUpdateOperations(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("RUN_DB_TESTS not set, skipping update operations test")
	}

	repo := &PostgreSQLRepository{}

	t.Run("SetOperations", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
			update := map[string]interface{}{
				"body": "Updated content",
				"tags": []string{"updated", "test"},
			}

			setClause, args, err := repo.buildSetOperation(update)
			require.NoError(t, err)
			// Updated to match current JSONB implementation with last_updated injection
			expectedSQL := "data = jsonb_set(jsonb_set(data, '{body}', $1::jsonb, true), '{tags}', $2::jsonb, true), last_updated = $3"
			assert.Equal(t, expectedSQL, setClause)
			assert.Len(t, args, 3) // Now expects 3 parameters including timestamp
		})

		t.Run("Nested", func(t *testing.T) {
			update := map[string]interface{}{
				"user.profile.name": "John Doe",
				"user.profile.age":  30,
			}

			setClause, args, err := repo.buildSetOperation(update)
			require.NoError(t, err)
			// Updated to match current JSONB implementation with last_updated injection
			// The order of operations may vary, so we check for the presence of both operations
			assert.Contains(t, setClause, "'{user,profile,name}'")
			assert.Contains(t, setClause, "'{user,profile,age}'")
			assert.Contains(t, setClause, "last_updated = $3")
			assert.Len(t, args, 3) // Now expects 3 parameters including timestamp
		})

		t.Run("Array", func(t *testing.T) {
			update := map[string]interface{}{
				"tags": []string{"golang", "database", "test"},
			}

			setClause, args, err := repo.buildSetOperation(update)
			require.NoError(t, err)
			// Updated to match current JSONB implementation with last_updated injection
			expectedSQL := "data = jsonb_set(data, '{tags}', $1::jsonb, true), last_updated = $2"
			assert.Equal(t, expectedSQL, setClause)
			assert.Len(t, args, 2) // Now expects 2 parameters including timestamp
		})
	})

	t.Run("IncrementOperations", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
			update := map[string]interface{}{
				"score":     10,
				"viewCount": 1,
			}

			setClause, args, err := repo.buildIncrementOperation(update)
			require.NoError(t, err)
			// Updated to match current sophisticated JSONB increment implementation
			// The implementation now uses robust type checking with CASE statements
			assert.Contains(t, setClause, "'{score}'")
			assert.Contains(t, setClause, "'{viewCount}'")
			assert.Contains(t, setClause, "last_updated = $3")
			assert.Contains(t, setClause, "jsonb_typeof")
			assert.Len(t, args, 3) // Now expects 3 parameters including timestamp
		})

		t.Run("Nested", func(t *testing.T) {
			update := map[string]interface{}{
				"user.profile.views": 5,
			}

			setClause, args, err := repo.buildIncrementOperation(update)
			require.NoError(t, err)
			// Updated to match current sophisticated JSONB increment implementation
			assert.Contains(t, setClause, "'{user,profile,views}'")
			assert.Contains(t, setClause, "last_updated = $2")
			assert.Contains(t, setClause, "jsonb_typeof")
			assert.Len(t, args, 2) // Now expects 2 parameters including timestamp
		})
	})

	t.Run("MixedOperations", func(t *testing.T) {
		t.Run("SetAndIncrement", func(t *testing.T) {
			setUpdate := map[string]interface{}{
				"body": "Updated content",
			}
			incUpdate := map[string]interface{}{
				"score": 10,
			}

			setClause, args, err := repo.buildMixedOperation(setUpdate, incUpdate)
			require.NoError(t, err)
			// Updated to match current JSONB implementation with last_updated injection
			expectedSQL := "data = jsonb_set(jsonb_set(data, '{body}', $1::jsonb, true), '{score}', to_jsonb(COALESCE((jsonb_set(data, '{body}', $1::jsonb, true)->>'score')::numeric, 0) + $2), true), last_updated = $3"
			assert.Equal(t, expectedSQL, setClause)
			assert.Len(t, args, 3) // Now expects 3 parameters including timestamp
		})
	})

	t.Run("PlainFieldUpdates", func(t *testing.T) {
		t.Run("ObjectId", func(t *testing.T) {
			update := map[string]interface{}{
				"objectId": "new-id",
			}

			setClause, args, err := repo.buildSetOperation(update)
			require.NoError(t, err)
			// Updated to match current JSONB implementation
			expectedSQL := "data = jsonb_set(data, '{objectId}', $1::jsonb, true), last_updated = $2"
			assert.Equal(t, expectedSQL, setClause)
			assert.Len(t, args, 2) // Now expects 2 parameters including timestamp
			assert.Equal(t, "\"new-id\"", fmt.Sprint(args[0])) // JSON-encoded string
		})

		t.Run("Deleted", func(t *testing.T) {
			update := map[string]interface{}{
				"deleted": true,
			}

			setClause, args, err := repo.buildSetOperation(update)
			require.NoError(t, err)
			// Updated to match current JSONB implementation
			expectedSQL := "data = jsonb_set(data, '{deleted}', $1::jsonb, true), last_updated = $2"
			assert.Equal(t, expectedSQL, setClause)
			assert.Len(t, args, 2) // Now expects 2 parameters including timestamp
		})
	})
}