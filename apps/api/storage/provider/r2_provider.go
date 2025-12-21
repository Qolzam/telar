// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
)

// r2Provider implements BlobProvider for Cloudflare R2 using AWS S3 SDK
type r2Provider struct {
	s3Client    *s3.Client
	bucket      string
	publicURL   string
	accountID   string
}

// NewR2Provider creates a new R2 provider from configuration
func NewR2Provider(cfg *platformconfig.StorageConfig) (BlobProvider, error) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID and R2_SECRET_ACCESS_KEY are required")
	}
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("R2_BUCKET_NAME is required")
	}

	// Build custom endpoint for R2
	// Format: https://<account-id>.r2.cloudflarestorage.com
	endpoint := cfg.Endpoint
	if endpoint == "" && cfg.AccountID != "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("R2_ENDPOINT or R2_ACCOUNT_ID is required")
	}

	// Create AWS config with static credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint resolver for R2
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // R2 requires path-style addressing
	})

	return &r2Provider{
		s3Client:  s3Client,
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
		accountID: cfg.AccountID,
	}, nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading a file
// contentLength enforces the exact file size at the R2 level (prevents size manipulation)
func (r *r2Provider) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, contentLength int64, expiresIn time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(r.s3Client)

	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ContentLength: aws.Int64(contentLength), // Enforce exact size
	}

	req, err := presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return req.URL, nil
}

// GeneratePresignedDownloadURL generates a presigned URL for downloading/viewing a file
// CRITICAL: If publicURL (CDN) is configured, return the CDN URL to avoid Class B operations
// Only use presigned URLs for private files or when CDN is not configured
func (r *r2Provider) GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	// If public CDN URL is configured, return it directly (avoids Class B operations)
	// Format: https://media.telar.press/users/123/file.jpg
	if r.publicURL != "" {
		// Ensure publicURL doesn't end with /
		publicBase := strings.TrimSuffix(r.publicURL, "/")
		// Key already includes the path (e.g., users/123/file.jpg)
		cdnURL := fmt.Sprintf("%s/%s", publicBase, key)
		return cdnURL, nil
	}

	// Fallback: Generate presigned URL for private files or when CDN not configured
	presignClient := s3.NewPresignClient(r.s3Client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return req.URL, nil
}

// Delete deletes a file from R2
func (r *r2Provider) Delete(ctx context.Context, key string) error {
	_, err := r.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}

	return nil
}

// GetMetadata retrieves file metadata (size) from R2
func (r *r2Provider) GetMetadata(ctx context.Context, key string) (int64, error) {
	headOutput, err := r.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get file metadata: %w", err)
	}

	if headOutput.ContentLength == nil {
		return 0, fmt.Errorf("content length is nil")
	}

	return *headOutput.ContentLength, nil
}

