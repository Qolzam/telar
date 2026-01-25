package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Database configuration from environment variables
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USERNAME", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DATABASE", "ai_engine")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Generate API key
	key := generateRandomString(32)
	fullKey := "gk_" + key
	hash, err := bcrypt.GenerateFromPassword([]byte(fullKey), 10)
	if err != nil {
		log.Fatalf("Failed to generate hash: %v", err)
	}

	// Create tenant
	tenantID := uuid.Must(uuid.NewV4())
	tenantName := getEnv("SEED_TENANT_NAME", "Telar Community")
	tenantEmail := getEnv("SEED_TENANT_EMAIL", "admin@telar.dev")

	_, err = db.ExecContext(ctx, `
		INSERT INTO tenants (id, name, owner_email, plan_tier, monthly_quota, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (owner_email) DO NOTHING
	`, tenantID, tenantName, tenantEmail, "free", 1000, true)
	if err != nil {
		log.Fatalf("Failed to create tenant: %v", err)
	}

	// Verify tenant was created or get existing
	var existingTenantID uuid.UUID
	err = db.GetContext(ctx, &existingTenantID, `SELECT id FROM tenants WHERE owner_email = $1`, tenantEmail)
	if err != nil {
		log.Fatalf("Failed to verify tenant: %v", err)
	}
	tenantID = existingTenantID

	log.Printf("✓ Tenant created/verified: %s (%s)", tenantID, tenantEmail)

	// Create app
	appID := uuid.Must(uuid.NewV4())
	appName := getEnv("SEED_APP_NAME", "Telar API")
	appSourceType := getEnv("SEED_APP_SOURCE_TYPE", "telar")

	_, err = db.ExecContext(ctx, `
		INSERT INTO apps (id, tenant_id, name, source_type, api_key_hash, metadata, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, '{}', $6, NOW())
	`, appID, tenantID, appName, appSourceType, string(hash), true)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	log.Printf("✓ App created: %s (%s)", appID, appName)
	log.Printf("")
	log.Printf("==========================================")
	log.Printf("API Key Generated:")
	log.Printf("  Key:  %s", fullKey)
	log.Printf("  Hash: %s", string(hash))
	log.Printf("")
	log.Printf("Tenant:")
	log.Printf("  ID:    %s", tenantID)
	log.Printf("  Name:  %s", tenantName)
	log.Printf("  Email: %s", tenantEmail)
	log.Printf("")
	log.Printf("App:")
	log.Printf("  ID:         %s", appID)
	log.Printf("  Name:       %s", appName)
	log.Printf("  SourceType: %s", appSourceType)
	log.Printf("==========================================")
	log.Printf("")
	log.Printf("To use this API key, set environment variable:")
	log.Printf("  export AI_ENGINE_INTERNAL_API_KEY=%s", fullKey)
	log.Printf("")
	log.Printf("Test the endpoint:")
	log.Printf("  curl -X POST http://localhost:9066/api/v1/analyze/content \\")
	log.Printf("    -H \"Authorization: Bearer %s\" \\", fullKey)
	log.Printf("    -H \"Content-Type: application/json\" \\")
	log.Printf("    -d '{\"content\":\"test\"}'")
	log.Printf("")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
