package members

import (
	"context"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	authmgmt "github.com/qolzam/telar/apps/api/auth/management"
	"strings"
)

type Member struct {
	ObjectId    string `json:"objectId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	CreatedDate int64  `json:"createdDate"`
	Avatar      string `json:"avatar,omitempty"`
}

type ListResult struct {
	Members []Member `json:"members"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
	Total   int64    `json:"total"`
}

type ListMembersParams struct {
	Search     string
	Limit      int
	Offset     int
	SortBy     string // "created_date" | "full_name" | "email"
	SortOrder  string // "asc" | "desc"
}

type Service interface {
	List(ctx context.Context, params ListMembersParams) (*ListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Member, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role string) error
	Ban(ctx context.Context, id uuid.UUID) error
}

type service struct {
	base *platform.BaseService
	auth authmgmt.UserManagement
}

func NewService(base *platform.BaseService, authManager authmgmt.UserManagement) Service {
	return &service{base: base, auth: authManager}
}

func (s *service) List(ctx context.Context, params ListMembersParams) (*ListResult, error) {
	// Build a simple query on userProfile. Full-text search TBD; return all when search empty.
	query := &dbi.Query{
		Conditions: []dbi.Field{},
		OrGroups:   [][]dbi.Field{},
	}
	// Search across fullName and email (ILIKE)
	if params.Search != "" {
		pat := "%" + params.Search + "%"
		query.OrGroups = append(query.OrGroups,
			[]dbi.Field{{
				Name:      "data->>'fullName'",
				Value:     pat,
				Operator:  "ILIKE",
				IsJSONB:   true,
				JSONBCast: "",
			}},
		)
		query.OrGroups = append(query.OrGroups,
			[]dbi.Field{{
				Name:      "data->>'email'",
				Value:     pat,
				Operator:  "ILIKE",
				IsJSONB:   true,
				JSONBCast: "",
			}},
		)
	}
	// Sort whitelist
	sort := map[string]int{"created_date": -1}
	switch params.SortBy {
	case "created_date":
		// default
	case "full_name":
		if strings.ToLower(params.SortOrder) == "asc" {
			sort = map[string]int{"data->>'fullName'": 1}
		} else {
			sort = map[string]int{"data->>'fullName'": -1}
		}
	case "email":
		if strings.ToLower(params.SortOrder) == "asc" {
			sort = map[string]int{"data->>'email'": 1}
		} else {
			sort = map[string]int{"data->>'email'": -1}
		}
	default:
		if strings.ToLower(params.SortOrder) == "asc" {
			sort = map[string]int{"created_date": 1}
		}
	}
	findOpts := &dbi.FindOptions{
		Limit: func() *int64 { v := int64(params.Limit); return &v }(),
		Skip:  func() *int64 { v := int64(params.Offset); return &v }(),
		Sort:  sort,
	}
	cur := <-s.base.Repository.Find(ctx, "userProfile", query, findOpts)
	if err := cur.Error(); err != nil {
		return nil, err
	}
	defer cur.Close()
	var out []Member
	for cur.Next() {
		var profile struct {
			ObjectId    uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
			FullName    string    `json:"fullName" bson:"fullName"`
			Email       string    `json:"email" bson:"email"`
			Avatar      string    `json:"avatar" bson:"avatar"`
			CreatedDate int64     `json:"createdDate" bson:"createdDate"`
			Role        string    `json:"role" bson:"role"`
		}
		if err := cur.Decode(&profile); err == nil {
			out = append(out, Member{
				ObjectId:    profile.ObjectId.String(),
				DisplayName: profile.FullName,
				Email:       profile.Email,
				Role:        profile.Role,
				Avatar:      profile.Avatar,
				CreatedDate: profile.CreatedDate,
			})
		}
	}
	countRes := <-s.base.Repository.Count(ctx, "userProfile", query)
	total := int64(0)
	if countRes.Error == nil {
		total = countRes.Count
	}
	return &ListResult{
		Members: out,
		Limit:   params.Limit,
		Offset:  params.Offset,
		Total:   total,
	}, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Member, error) {
	query := &dbi.Query{
		Conditions: []dbi.Field{
			{Name: "object_id", Value: id, Operator: "=", IsJSONB: false},
		},
	}
	res := <-s.base.Repository.FindOne(ctx, "userProfile", query)
	if err := res.Error(); err != nil {
		return nil, err
	}
	var profile struct {
		ObjectId    uuid.UUID `json:"objectId" bson:"objectId" db:"objectId"`
		FullName    string    `json:"fullName" bson:"fullName"`
		Email       string    `json:"email" bson:"email"`
		Avatar      string    `json:"avatar" bson:"avatar"`
		CreatedDate int64     `json:"createdDate" bson:"createdDate"`
		Role        string    `json:"role" bson:"role"`
	}
	if err := res.Decode(&profile); err != nil {
		return nil, err
	}
	return &Member{
		ObjectId:    profile.ObjectId.String(),
		DisplayName: profile.FullName,
		Email:       profile.Email,
		Role:        profile.Role,
		Avatar:      profile.Avatar,
		CreatedDate: profile.CreatedDate,
	}, nil
}

func (s *service) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	if s.auth == nil {
		return nil
	}
	return s.auth.UpdateUserRole(ctx, id.String(), role)
}

func (s *service) Ban(ctx context.Context, id uuid.UUID) error {
	if s.auth == nil {
		return nil
	}
	return s.auth.UpdateUserStatus(ctx, id.String(), "banned")
}


