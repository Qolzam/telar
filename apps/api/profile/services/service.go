package services

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/utils"
	profileErrors "github.com/qolzam/telar/apps/api/profile/errors"
	"github.com/qolzam/telar/apps/api/profile/models"
)

type Service struct {
	base   *platform.BaseService
	config *platformconfig.Config
}

// Ensure Service implements ProfileServiceClient interface
var _ ProfileServiceClient = (*Service)(nil)

func NewService(base *platform.BaseService, cfg *platformconfig.Config) *Service {
	return &Service{
		base:   base,
		config: cfg,
	}
}

// profileQueryBuilder is a private helper for building profile service queries
type profileQueryBuilder struct {
	query *dbi.Query
}

func newProfileQueryBuilder() *profileQueryBuilder {
	return &profileQueryBuilder{
		query: &dbi.Query{
			Conditions: []dbi.Field{},
		},
	}
}

func (qb *profileQueryBuilder) WhereObjectID(objectID uuid.UUID) *profileQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "object_id", // Indexed column
		Value:    objectID,
		Operator: "=",
		IsJSONB:  false,
	})
	return qb
}

func (qb *profileQueryBuilder) WhereObjectIDIn(ids []uuid.UUID) *profileQueryBuilder {
	if len(ids) == 0 {
		return qb
	}
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:    "object_id", // Indexed column
		Value:   ids,
		IsJSONB: false,
	})
	return qb
}

func (qb *profileQueryBuilder) WhereEmail(email string) *profileQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "data->>'email'", // JSONB field
		Value:    email,
		Operator: "=",
		IsJSONB:  true,
	})
	return qb
}

func (qb *profileQueryBuilder) WhereSocialName(socialName string) *profileQueryBuilder {
	qb.query.Conditions = append(qb.query.Conditions, dbi.Field{
		Name:     "data->>'socialName'", // JSONB field
		Value:    strings.ToLower(socialName),
		Operator: "=",
		IsJSONB:  true,
	})
	return qb
}

func (qb *profileQueryBuilder) Build() *dbi.Query {
	return qb.query
}

func (s *Service) CreateIndexes(ctx context.Context) error {
	idx := map[string]interface{}{
		"fullName":   "text",
		"socialName": 1,
		"objectId":   1,
	}
	return <-s.base.Repository.CreateIndex(ctx, "userProfile", idx)
}

func (s *Service) UpdateLastSeen(ctx context.Context, userId uuid.UUID, _ int) error {
	now := utils.UTCNowUnix()
	query := newProfileQueryBuilder().WhereObjectID(userId).Build()
	updates := map[string]interface{}{"lastSeen": now}
	upsert := false
	return (<-s.base.Repository.Update(ctx, "userProfile", query, updates, &dbi.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) UpdateProfile(ctx context.Context, userId uuid.UUID, req *models.UpdateProfileRequest) error {
	set := map[string]interface{}{}
	if req.FullName != nil {
		set["fullName"] = *req.FullName
	}
	if req.Avatar != nil {
		set["avatar"] = *req.Avatar
	}
	if req.Banner != nil {
		set["banner"] = *req.Banner
	}
	if req.TagLine != nil {
		set["tagLine"] = *req.TagLine
	}
	if req.SocialName != nil {
		set["socialName"] = *req.SocialName
	}
	if req.WebUrl != nil {
		set["webUrl"] = *req.WebUrl
	}
	if req.CompanyName != nil {
		set["companyName"] = *req.CompanyName
	}
	if req.FacebookId != nil {
		set["facebookId"] = *req.FacebookId
	}
	if req.InstagramId != nil {
		set["instagramId"] = *req.InstagramId
	}
	if req.TwitterId != nil {
		set["twitterId"] = *req.TwitterId
	}

	if len(set) == 0 {
		return nil
	}
	query := newProfileQueryBuilder().WhereObjectID(userId).Build()
	upsert := false
	return (<-s.base.Repository.Update(ctx, "userProfile", query, set, &dbi.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) CreateOrUpdateDTO(ctx context.Context, req *models.CreateProfileRequest) error {
	if req.ObjectId == uuid.Nil {
		return nil
	}

	data := map[string]interface{}{"objectId": req.ObjectId}
	if req.FullName != nil {
		data["fullName"] = *req.FullName
	}
	if req.SocialName != nil {
		data["socialName"] = *req.SocialName
	}
	if req.Email != nil {
		data["email"] = *req.Email
	}
	if req.Avatar != nil {
		data["avatar"] = *req.Avatar
	}
	if req.Banner != nil {
		data["banner"] = *req.Banner
	}
	if req.TagLine != nil {
		data["tagLine"] = *req.TagLine
	}
	if req.CreatedDate != nil {
		data["createdDate"] = *req.CreatedDate
	}
	if req.LastUpdated != nil {
		data["lastUpdated"] = *req.LastUpdated
	}
	if req.LastSeen != nil {
		data["lastSeen"] = *req.LastSeen
	}
	if req.FollowCount != nil {
		data["followCount"] = *req.FollowCount
	}
	if req.FollowerCount != nil {
		data["followerCount"] = *req.FollowerCount
	}

	// Upsert to create or update based on objectId
	query := newProfileQueryBuilder().WhereObjectID(req.ObjectId).Build()
	upsert := true
	return (<-s.base.Repository.Update(ctx, "userProfile", query, data, &dbi.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) FindMyProfile(ctx context.Context, userId uuid.UUID) (*models.Profile, error) {
	query := newProfileQueryBuilder().WhereObjectID(userId).Build()
	res := <-s.base.Repository.FindOne(ctx, "userProfile", query)
	if res.Error() != nil {
		if errors.Is(res.Error(), dbi.ErrNoDocuments) {
			return nil, profileErrors.ErrProfileNotFound
		}
		return nil, res.Error()
	}
	out := new(models.Profile)
	if err := res.Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*models.Profile, error) {
	query := newProfileQueryBuilder().WhereObjectID(id).Build()
	res := <-s.base.Repository.FindOne(ctx, "userProfile", query)
	if res.Error() != nil {
		if errors.Is(res.Error(), dbi.ErrNoDocuments) {
			return nil, profileErrors.ErrProfileNotFound
		}
		return nil, res.Error()
	}
	out := new(models.Profile)
	if err := res.Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) FindBySocialName(ctx context.Context, name string) (*models.Profile, error) {
	query := newProfileQueryBuilder().WhereSocialName(name).Build()
	res := <-s.base.Repository.FindOne(ctx, "userProfile", query)
	if res.Error() != nil {
		// Map repository "not found" error to domain error
		if errors.Is(res.Error(), dbi.ErrNoDocuments) {
			return nil, profileErrors.ErrProfileNotFound
		}
		return nil, res.Error()
	}
	out := new(models.Profile)
	if err := res.Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) FindManyByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Profile, error) {
	query := newProfileQueryBuilder().WhereObjectIDIn(ids).Build()
	res := <-s.base.Repository.Find(ctx, "userProfile", query, &dbi.FindOptions{Limit: nil, Skip: nil, Sort: map[string]int{"created_date": -1}})
	if res.Error() != nil {
		return nil, res.Error()
	}
	var docs []*models.Profile
	for res.Next() {
		var m models.Profile
		if err := res.Decode(&m); err == nil {
			docs = append(docs, &m)
		}
	}
	res.Close()
	return docs, nil
}

func (s *Service) Query(ctx context.Context, search string, limit int64, skip int64) ([]*models.Profile, error) {
	// TODO: Full-text search needs PostgreSQL-specific implementation
	// For now, create an empty query - this will return all profiles
	// Full-text search should use PostgreSQL tsvector/tsquery
	query := newProfileQueryBuilder().Build()
	res := <-s.base.Repository.Find(ctx, "userProfile", query, &dbi.FindOptions{Limit: &limit, Skip: &skip, Sort: map[string]int{"created_date": -1}})
	if res.Error() != nil {
		return nil, res.Error()
	}
	var docs []*models.Profile
	for res.Next() {
		var m models.Profile
		if err := res.Decode(&m); err == nil {
			docs = append(docs, &m)
		}
	}
	res.Close()
	return docs, nil
}

func (s *Service) Increase(ctx context.Context, field string, inc int, userId uuid.UUID) error {
	query := newProfileQueryBuilder().WhereObjectID(userId).Build()
	// Note: IncrementFields should be used for increment operations
	// For now, using Update with explicit increment syntax
	increments := map[string]interface{}{field: inc}
	return (<-s.base.Repository.IncrementFields(ctx, "userProfile", query, increments)).Error
}

// CreateProfileOnSignup creates a profile during signup flow (implements ProfileServiceClient interface)
func (s *Service) CreateProfileOnSignup(ctx context.Context, req *models.CreateProfileRequest) error {
	return s.CreateOrUpdateDTO(ctx, req)
}

// GetProfile retrieves a profile by user ID (implements ProfileServiceClient interface)
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	return s.FindByID(ctx, userID)
}

// GetProfilesByIds retrieves multiple profiles by user IDs (implements ProfileServiceClient interface)
func (s *Service) GetProfilesByIds(ctx context.Context, userIds []uuid.UUID) ([]*models.Profile, error) {
	return s.FindManyByIDs(ctx, userIds)
}
