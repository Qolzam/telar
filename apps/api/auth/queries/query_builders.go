package services

import (
	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

// userAuthQueryBuilder is a helper struct for building userAuth-specific queries.
// It knows the schema of the `userAuth` table and provides fluent methods for query construction.
type userAuthQueryBuilder struct {
	query *dbi.Query
}

// NewUserAuthQueryBuilder creates a new userAuthQueryBuilder instance.
func NewUserAuthQueryBuilder() *userAuthQueryBuilder {
	return &userAuthQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereUsername adds a filter for the username (JSONB field).
func (b *userAuthQueryBuilder) WhereUsername(username string) *userAuthQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'username'",
		Value:    username,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereObjectID adds a filter for the object_id (indexed column).
func (b *userAuthQueryBuilder) WhereObjectID(objectID uuid.UUID) *userAuthQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "object_id",
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereRole adds a filter for the role (JSONB field).
func (b *userAuthQueryBuilder) WhereRole(role string) *userAuthQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'role'",
		Value:    role,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereEmailVerified adds a filter for emailVerified (JSONB field).
func (b *userAuthQueryBuilder) WhereEmailVerified(verified bool) *userAuthQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'emailVerified'",
		Value:      verified,
		Operator:   "=",
		IsJSONB:    true,
		JSONBCast:  "::boolean",
	})
	return b
}

// Build returns the constructed Query object.
func (b *userAuthQueryBuilder) Build() *dbi.Query {
	return b.query
}

// userProfileQueryBuilder is a helper struct for building userProfile-specific queries.
type userProfileQueryBuilder struct {
	query *dbi.Query
}

// NewUserProfileQueryBuilder creates a new userProfileQueryBuilder instance.
func NewUserProfileQueryBuilder() *userProfileQueryBuilder {
	return &userProfileQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereObjectID adds a filter for the object_id (indexed column).
func (b *userProfileQueryBuilder) WhereObjectID(objectID uuid.UUID) *userProfileQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "object_id",
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereEmail adds a filter for the email (JSONB field).
func (b *userProfileQueryBuilder) WhereEmail(email string) *userProfileQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'email'",
		Value:    email,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereSocialName adds a filter for the socialName (JSONB field).
func (b *userProfileQueryBuilder) WhereSocialName(socialName string) *userProfileQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'socialName'",
		Value:    socialName,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// Build returns the constructed Query object.
func (b *userProfileQueryBuilder) Build() *dbi.Query {
	return b.query
}

// userVerificationQueryBuilder is a helper struct for building userVerification-specific queries.
type userVerificationQueryBuilder struct {
	query *dbi.Query
}

// NewUserVerificationQueryBuilder creates a new userVerificationQueryBuilder instance.
func NewUserVerificationQueryBuilder() *userVerificationQueryBuilder {
	return &userVerificationQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereObjectID adds a filter for the object_id (indexed column).
func (b *userVerificationQueryBuilder) WhereObjectID(objectID uuid.UUID) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "object_id",
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return b
}

// WhereUserId adds a filter for the userId (JSONB field).
func (b *userVerificationQueryBuilder) WhereUserId(userID uuid.UUID) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'userId'",
		Value:    userID.String(),
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereCode adds a filter for the code (JSONB field).
func (b *userVerificationQueryBuilder) WhereCode(code string) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'code'",
		Value:    code,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereTarget adds a filter for the target (JSONB field).
func (b *userVerificationQueryBuilder) WhereTarget(target string) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'target'",
		Value:    target,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereTargetType adds a filter for the targetType (JSONB field).
func (b *userVerificationQueryBuilder) WhereTargetType(targetType string) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'targetType'",
		Value:    targetType,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereExpiresAtBefore adds a filter for expiresAt with < operator (JSONB field).
func (b *userVerificationQueryBuilder) WhereExpiresAtBefore(timestamp int64) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'expiresAt'",
		Value:      timestamp,
		Operator:   "<",
		IsJSONB:    true,
		JSONBCast:  "::bigint",
	})
	return b
}

// WhereUsed adds a filter for the used flag (JSONB field).
func (b *userVerificationQueryBuilder) WhereUsed(used bool) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'used'",
		Value:      used,
		Operator:   "=",
		IsJSONB:    true,
		JSONBCast:  "::boolean",
	})
	return b
}

// WhereIsVerified adds a filter for the isVerified flag (JSONB field).
func (b *userVerificationQueryBuilder) WhereIsVerified(verified bool) *userVerificationQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'isVerified'",
		Value:      verified,
		Operator:   "=",
		IsJSONB:    true,
		JSONBCast:  "::boolean",
	})
	return b
}

// Build returns the constructed Query object.
func (b *userVerificationQueryBuilder) Build() *dbi.Query {
	return b.query
}

// resetTokenQueryBuilder is a helper struct for building resetToken-specific queries.
type resetTokenQueryBuilder struct {
	query *dbi.Query
}

// NewResetTokenQueryBuilder creates a new resetTokenQueryBuilder instance.
func NewResetTokenQueryBuilder() *resetTokenQueryBuilder {
	return &resetTokenQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
			OrGroups:   [][]dbi.Field{},
		},
	}
}

// WhereToken adds a filter for the token (JSONB field).
func (b *resetTokenQueryBuilder) WhereToken(token string) *resetTokenQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'token'",
		Value:    token,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereEmail adds a filter for the email (JSONB field).
func (b *resetTokenQueryBuilder) WhereEmail(email string) *resetTokenQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:     "data->>'email'",
		Value:    email,
		Operator: "=",
		IsJSONB:  true,
	})
	return b
}

// WhereExpiresAtBefore adds a filter for expiresAt with < operator (JSONB field).
func (b *resetTokenQueryBuilder) WhereExpiresAtBefore(timestamp int64) *resetTokenQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'expiresAt'",
		Value:      timestamp,
		Operator:   "<",
		IsJSONB:    true,
		JSONBCast:  "::bigint",
	})
	return b
}

// WhereUsed adds a filter for the used flag (JSONB field).
func (b *resetTokenQueryBuilder) WhereUsed(used bool) *resetTokenQueryBuilder {
	b.query.Conditions = append(b.query.Conditions, dbi.Field{
		Name:       "data->>'used'",
		Value:      used,
		Operator:   "=",
		IsJSONB:    true,
		JSONBCast:  "::boolean",
	})
	return b
}

// Build returns the constructed Query object.
func (b *resetTokenQueryBuilder) Build() *dbi.Query {
	return b.query
}

