package performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/comments/models"
	"github.com/qolzam/telar/apps/api/comments/services"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
)

// Helper function to build service from environment
func buildServiceFromEnv(ctx context.Context, dbType string) (*platform.BaseService, string, error) {
	// Use the Config-First pattern with test harness
	suite := testutil.Setup(&testing.T{})
	iso := testutil.NewIsolatedTest(&testing.T{}, dbType, suite.Config())
	if iso.Repo == nil {
		return nil, fmt.Sprintf("%s not available", dbType), nil
	}
	
	base, err := platform.NewBaseService(ctx, iso.Config)
	return base, "", err
}

// Benchmark single comment creation
func BenchmarkCommentService_CreateComment_MongoDB(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()
	var createdComments []uuid.UUID

	for i := 0; i < b.N; i++ {
		postID := uuid.Must(uuid.NewV4())
		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: "benchmark test comment",
		}

		_, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("create comment error: %v", err)
		}
		// Note: We can't get the comment ID from the response in this model
		// So we'll just track the post ID for cleanup
		createdComments = append(createdComments, postID)
	}

	// Cleanup - delete comments by post ID
	for _, postID := range createdComments {
		<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
	}
}

func BenchmarkCommentService_CreateComment_PostgreSQL(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypePostgreSQL)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()
	var createdComments []uuid.UUID

	for i := 0; i < b.N; i++ {
		postID := uuid.Must(uuid.NewV4())
		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: "benchmark test comment",
		}

		_, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("create comment error: %v", err)
		}
		// Note: We can't get the comment ID from the response in this model
		// So we'll just track the post ID for cleanup
		createdComments = append(createdComments, postID)
	}

	// Cleanup - delete comments by post ID
	for _, postID := range createdComments {
		<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
	}
}

// Benchmark comment retrieval
func BenchmarkCommentService_GetCommentsByPost_MongoDB(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	// Pre-populate with test data
	postID := uuid.Must(uuid.NewV4())
	var createdComments []uuid.UUID

	// Create 100 comments for this post
	for i := 0; i < 100; i++ {
		createdComments = append(createdComments, postID)

		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: fmt.Sprintf("benchmark comment %d", i+1),
		}

		_, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("setup comment error: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := commentService.GetCommentsByPost(ctx, postID, nil)
		if err != nil {
			b.Fatalf("get comments error: %v", err)
		}
	}

	// Cleanup
	<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
}

func BenchmarkCommentService_GetCommentsByPost_PostgreSQL(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypePostgreSQL)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	// Pre-populate with test data
	postID := uuid.Must(uuid.NewV4())
	var createdComments []uuid.UUID

	// Create 100 comments for this post
	for i := 0; i < 100; i++ {
		createdComments = append(createdComments, postID)

		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: fmt.Sprintf("benchmark comment %d", i+1),
		}

		_, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("setup comment error: %v", err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := commentService.GetCommentsByPost(ctx, postID, nil)
		if err != nil {
			b.Fatalf("get comments error: %v", err)
		}
	}

	// Cleanup
	<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
}

// Benchmark concurrent comment creation
func BenchmarkCommentService_CreateComment_Concurrent_MongoDB(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	commentChan := make(chan uuid.UUID, b.N)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			postID := uuid.Must(uuid.NewV4())
			req := &models.CreateCommentRequest{
				PostId:  postID,
				Text: fmt.Sprintf("concurrent comment %d", index+1),
			}

			_, err := commentService.CreateComment(ctx, req, userCtx)
			if err != nil {
				b.Errorf("concurrent create comment error: %v", err)
				return
			}
			commentChan <- postID
		}(i)
	}

	wg.Wait()
	close(commentChan)

	// Cleanup
	for postID := range commentChan {
		<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
	}
}

func BenchmarkCommentService_CreateComment_Concurrent_PostgreSQL(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypePostgreSQL)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	commentChan := make(chan uuid.UUID, b.N)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			postID := uuid.Must(uuid.NewV4())
			req := &models.CreateCommentRequest{
				PostId:  postID,
				Text: fmt.Sprintf("concurrent comment %d", index+1),
			}

			_, err := commentService.CreateComment(ctx, req, userCtx)
			if err != nil {
				b.Errorf("concurrent create comment error: %v", err)
				return
			}
			commentChan <- postID
		}(i)
	}

	wg.Wait()
	close(commentChan)

	// Cleanup
	for postID := range commentChan {
		<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
	}
}

// Benchmark comment update
func BenchmarkCommentService_UpdateComment_MongoDB(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	// Pre-create a comment
	postID := uuid.Must(uuid.NewV4())
	req := &models.CreateCommentRequest{
		PostId:  postID,
		Text: "benchmark comment for update",
	}

	comment, err := commentService.CreateComment(ctx, req, userCtx)
	if err != nil {
		b.Fatalf("setup comment error: %v", err)
	}

	// Update request
	updateReq := &models.UpdateCommentRequest{
		Text: "updated benchmark comment",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := commentService.UpdateComment(ctx, comment.ObjectId, updateReq, userCtx)
		if err != nil {
			b.Fatalf("update comment error: %v", err)
		}
	}

	// Cleanup
	<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
}

func BenchmarkCommentService_UpdateComment_PostgreSQL(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypePostgreSQL)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	// Pre-create a comment
	postID := uuid.Must(uuid.NewV4())
	req := &models.CreateCommentRequest{
		PostId:  postID,
		Text: "benchmark comment for update",
	}

	comment, err := commentService.CreateComment(ctx, req, userCtx)
	if err != nil {
		b.Fatalf("setup comment error: %v", err)
	}

	// Update request
	updateReq := &models.UpdateCommentRequest{
		Text: "updated benchmark comment",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := commentService.UpdateComment(ctx, comment.ObjectId, updateReq, userCtx)
		if err != nil {
			b.Fatalf("update comment error: %v", err)
		}
	}

	// Cleanup
	<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
}

// Benchmark comment deletion
func BenchmarkCommentService_DeleteComment_MongoDB(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create a comment for deletion
		postID := uuid.Must(uuid.NewV4())
		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: "benchmark comment for deletion",
		}

		comment, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("setup comment error: %v", err)
		}

		// Delete the comment
		err = commentService.DeleteComment(ctx, comment.ObjectId, postID, userCtx)
		if err != nil {
			b.Fatalf("delete comment error: %v", err)
		}
	}
}

func BenchmarkCommentService_DeleteComment_PostgreSQL(b *testing.B) {
	if !testutil.ShouldRunDatabaseTests() {
		b.Skip("set RUN_DB_TESTS=1 to run performance tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypePostgreSQL)
	if skip != "" {
		b.Skip(skip)
	}
	if err != nil {
		b.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "benchmark@example.com",
		DisplayName: "Benchmark User",
		SocialName:  "benchmarkuser",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create a comment for deletion
		postID := uuid.Must(uuid.NewV4())
		req := &models.CreateCommentRequest{
			PostId:  postID,
			Text: "benchmark comment for deletion",
		}

		comment, err := commentService.CreateComment(ctx, req, userCtx)
		if err != nil {
			b.Fatalf("setup comment error: %v", err)
		}

		// Delete the comment
		err = commentService.DeleteComment(ctx, comment.ObjectId, postID, userCtx)
		if err != nil {
			b.Fatalf("delete comment error: %v", err)
		}
	}
}

// Load test for comment service
func TestCommentService_LoadTest(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run load tests")
	}

	ctx := context.Background()
	base, skip, err := buildServiceFromEnv(ctx, dbi.DatabaseTypeMongoDB)
	if skip != "" {
		t.Skip(skip)
	}
	if err != nil {
		t.Fatalf("base service error: %v", err)
	}

	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}

	commentService := services.NewCommentService(base, cfg)
	userCtx := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "loadtest@example.com",
		DisplayName: "Load Test User",
		SocialName:  "loadtestuser",
	}

	// Test parameters
	numRoutines := 10
	commentsPerRoutine := 100
	startTime := time.Now()

	var wg sync.WaitGroup
	commentChan := make(chan uuid.UUID, numRoutines*commentsPerRoutine)

	// Start concurrent routines
	for routineID := 0; routineID < numRoutines; routineID++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			postID := uuid.Must(uuid.NewV4())

			for j := 0; j < commentsPerRoutine; j++ {
				req := &models.CreateCommentRequest{
					PostId:  postID,
					Text: fmt.Sprintf("load test comment %d from routine %d", j+1, routineID+1),
				}

				_, err := commentService.CreateComment(ctx, req, userCtx)
				if err != nil {
					t.Errorf("routine %d, comment %d creation error: %v", routineID+1, j+1, err)
					continue
				}
				commentChan <- postID
			}
		}(routineID)
	}

	wg.Wait()
	close(commentChan)

	duration := time.Since(startTime)
	t.Logf("Load test completed: %d comments created in %v", numRoutines*commentsPerRoutine, duration)

	// Cleanup
	for postID := range commentChan {
		<-base.Repository.Delete(ctx, "comment", map[string]interface{}{"postId": postID})
	}
}
