package services

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
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
    update := map[string]interface{}{ "$set": map[string]interface{}{"lastSeen": now} }
    upsert := false
    return (<-s.base.Repository.Update(ctx, "userProfile", map[string]interface{}{"objectId": userId}, update, &interfaces.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) UpdateProfile(ctx context.Context, userId uuid.UUID, req *models.UpdateProfileRequest) error {
    set := map[string]interface{}{}
    if req.FullName != nil { set["fullName"] = *req.FullName }
    if req.Avatar != nil { set["avatar"] = *req.Avatar }
    if req.Banner != nil { set["banner"] = *req.Banner }
    if req.TagLine != nil { set["tagLine"] = *req.TagLine }
    if req.SocialName != nil { set["socialName"] = *req.SocialName }
    if req.WebUrl != nil { set["webUrl"] = *req.WebUrl }
    if req.CompanyName != nil { set["companyName"] = *req.CompanyName }
    if req.FacebookId != nil { set["facebookId"] = *req.FacebookId }
    if req.InstagramId != nil { set["instagramId"] = *req.InstagramId }
    if req.TwitterId != nil { set["twitterId"] = *req.TwitterId }
    
    if len(set) == 0 { return nil }
    update := map[string]interface{}{ "$set": set }
    upsert := false
    return (<-s.base.Repository.Update(ctx, "userProfile", map[string]interface{}{"objectId": userId}, update, &interfaces.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) CreateOrUpdateDTO(ctx context.Context, req *models.CreateProfileRequest) error {
    if req.ObjectId == uuid.Nil { return nil }
    
    data := map[string]interface{}{"objectId": req.ObjectId}
    if req.FullName != nil { data["fullName"] = *req.FullName }
    if req.SocialName != nil { data["socialName"] = *req.SocialName }
    if req.Email != nil { data["email"] = *req.Email }
    if req.Avatar != nil { data["avatar"] = *req.Avatar }
    if req.Banner != nil { data["banner"] = *req.Banner }
    if req.TagLine != nil { data["tagLine"] = *req.TagLine }
    if req.CreatedDate != nil { data["createdDate"] = *req.CreatedDate }
    if req.LastUpdated != nil { data["lastUpdated"] = *req.LastUpdated }
    if req.LastSeen != nil { data["lastSeen"] = *req.LastSeen }
    if req.FollowCount != nil { data["followCount"] = *req.FollowCount }
    if req.FollowerCount != nil { data["followerCount"] = *req.FollowerCount }
    
    // Upsert to create or update based on objectId
    upsert := true
    update := map[string]interface{}{"$set": data}
    return (<-s.base.Repository.Update(ctx, "userProfile", map[string]interface{}{"objectId": req.ObjectId}, update, &interfaces.UpdateOptions{Upsert: &upsert})).Error
}

func (s *Service) FindMyProfile(ctx context.Context, userId uuid.UUID) (*models.Profile, error) {
    res := <-s.base.Repository.FindOne(ctx, "userProfile", map[string]interface{}{"objectId": userId})
    if res.Error() != nil {
        if errors.Is(res.Error(), interfaces.ErrNoDocuments) {
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
    res := <-s.base.Repository.FindOne(ctx, "userProfile", map[string]interface{}{"objectId": id})
    if res.Error() != nil {
        if errors.Is(res.Error(), interfaces.ErrNoDocuments) {
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
    res := <-s.base.Repository.FindOne(ctx, "userProfile", map[string]interface{}{"socialName": strings.ToLower(name)})
    if res.Error() != nil {
        // Map repository "not found" error to domain error
        if errors.Is(res.Error(), interfaces.ErrNoDocuments) {
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
    filter := map[string]interface{}{"objectId": map[string]interface{}{"$in": ids}}
    res := <-s.base.Repository.Find(ctx, "userProfile", filter, &interfaces.FindOptions{Limit: nil, Skip: nil, Sort: map[string]int{"createdDate": -1}})
    if res.Error() != nil { return nil, res.Error() }
    var docs []*models.Profile
    for res.Next() {
        var m models.Profile
        if err := res.Decode(&m); err == nil { docs = append(docs, &m) }
    }
    res.Close()
    return docs, nil
}

func (s *Service) Query(ctx context.Context, search string, limit int64, skip int64) ([]*models.Profile, error) {
    filter := map[string]interface{}{}
    if search != "" { filter["$text"] = map[string]interface{}{"$search": search} }
    res := <-s.base.Repository.Find(ctx, "userProfile", filter, &interfaces.FindOptions{Limit: &limit, Skip: &skip, Sort: map[string]int{"createdDate": -1}})
    if res.Error() != nil { return nil, res.Error() }
    var docs []*models.Profile
    for res.Next() {
        var m models.Profile
        if err := res.Decode(&m); err == nil { docs = append(docs, &m) }
    }
    res.Close()
    return docs, nil
}

func (s *Service) Increase(ctx context.Context, field string, inc int, userId uuid.UUID) error {
    update := map[string]interface{}{"$inc": map[string]interface{}{field: inc}}
    upsert := false
    return (<-s.base.Repository.Update(ctx, "userProfile", map[string]interface{}{"objectId": userId}, update, &interfaces.UpdateOptions{Upsert: &upsert})).Error
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

