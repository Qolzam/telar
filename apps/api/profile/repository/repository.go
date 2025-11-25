// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package repository

import (
	"context"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/profile/models"
)

// ProfileFilter represents filtering criteria for querying profiles
type ProfileFilter struct {
	SocialName   *string
	Email        *string
	SearchText   *string // For full-text search
	CreatedAfter *int64
}

// ProfileRepository defines the interface for profile-specific database operations
// This is a domain-specific repository that knows exactly what a "Profile" is
// and how to execute optimized SQL queries for that specific domain.
type ProfileRepository interface {
	// Create inserts a new profile
	Create(ctx context.Context, profile *models.Profile) error

	// FindByID retrieves a profile by user ID
	FindByID(ctx context.Context, userID uuid.UUID) (*models.Profile, error)

	// FindBySocialName retrieves a profile by social name (unique)
	FindBySocialName(ctx context.Context, socialName string) (*models.Profile, error)

	// FindByIDs retrieves multiple profiles by user IDs
	FindByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*models.Profile, error)

	// Find retrieves profiles matching the filter criteria with pagination
	Find(ctx context.Context, filter ProfileFilter, limit, offset int) ([]*models.Profile, error)

	// Count returns the number of profiles matching the filter criteria
	Count(ctx context.Context, filter ProfileFilter) (int64, error)

	// Update updates an existing profile
	Update(ctx context.Context, profile *models.Profile) error

	// UpdateLastSeen updates the last_seen timestamp for a profile
	UpdateLastSeen(ctx context.Context, userID uuid.UUID) error

	// UpdateOwnerProfile updates display name and avatar for a profile
	// This is used when a user updates their profile information
	UpdateOwnerProfile(ctx context.Context, userID uuid.UUID, displayName, avatar string) error

	// Delete deletes a profile by user ID (soft delete)
	Delete(ctx context.Context, userID uuid.UUID) error
}

