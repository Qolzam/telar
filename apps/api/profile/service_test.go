package profile

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/profile/models"
	"github.com/qolzam/telar/apps/api/profile/services"
)

type fakeRepo struct{
    lastCreateIndex map[string]interface{}
    lastUpdateFilter interface{}
    lastUpdateData interface{}
    lastUpdateOptions *interfaces.UpdateOptions

    lastUpdateFieldsQuery *interfaces.Query
    lastUpdateFieldsUpdates map[string]interface{}
    
    lastIncrementFieldsQuery *interfaces.Query
    lastIncrementFieldsIncrements map[string]interface{}

    findDocs []map[string]interface{}
    findFilter interface{}
    findOptions *interfaces.FindOptions

    singleDoc map[string]interface{}
}

func (f *fakeRepo) Save(ctx context.Context, collectionName string, objectID uuid.UUID, ownerUserID uuid.UUID, createdDate, lastUpdated int64, data interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) SaveMany(ctx context.Context, collectionName string, items []interfaces.SaveItem) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Find(ctx context.Context, collectionName string, query *interfaces.Query, options *interfaces.FindOptions) <-chan interfaces.QueryResult {
    f.findFilter = query
    f.findOptions = options
    ch := make(chan interfaces.QueryResult,1)
    ch <- &fakeQueryResult{docs: f.findDocs}
    return ch
}
func (f *fakeRepo) FindOne(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.SingleResult {
    f.findFilter = query
    ch := make(chan interfaces.SingleResult,1)
    ch <- &fakeSingleResult{doc: f.singleDoc}
    return ch
}
func (f *fakeRepo) Update(ctx context.Context, collectionName string, query *interfaces.Query, data interface{}, options *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
    f.lastUpdateFilter = query
    f.lastUpdateData = data
    f.lastUpdateOptions = options
    ch := make(chan interfaces.RepositoryResult,1)
    ch <- interfaces.RepositoryResult{}
    return ch
}
func (f *fakeRepo) UpdateMany(ctx context.Context, collectionName string, query *interfaces.Query, data interface{}, options *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Delete(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) DeleteMany(ctx context.Context, collectionName string, queries []*interfaces.Query) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Count(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.CountResult { ch:=make(chan interfaces.CountResult,1); ch<-interfaces.CountResult{}; return ch }
func (f *fakeRepo) BulkWrite(ctx context.Context, collectionName string, operations []interfaces.BulkOperation) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error { f.lastCreateIndex = indexes; ch:=make(chan error,1); ch<-nil; return ch }
func (f *fakeRepo) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) { return nil, nil }
func (f *fakeRepo) Begin(ctx context.Context) (interfaces.Transaction, error) { return nil, nil }
func (f *fakeRepo) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) { return nil, nil }
func (f *fakeRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }
func (f *fakeRepo) Ping(ctx context.Context) <-chan error { ch:=make(chan error,1); ch<-nil; return ch }
func (f *fakeRepo) Close() error { return nil }
func (f *fakeRepo) UpdateFields(ctx context.Context, collectionName string, query *interfaces.Query, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
    f.lastUpdateFieldsQuery = query
    f.lastUpdateFieldsUpdates = updates
    ch:=make(chan interfaces.RepositoryResult,1)
    ch<-interfaces.RepositoryResult{}
    return ch
}
func (f *fakeRepo) IncrementFields(ctx context.Context, collectionName string, query *interfaces.Query, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
    f.lastIncrementFieldsQuery = query
    f.lastIncrementFieldsIncrements = increments
    ch:=make(chan interfaces.RepositoryResult,1)
    ch<-interfaces.RepositoryResult{}
    return ch
}
func (f *fakeRepo) UpdateAndIncrement(ctx context.Context, collectionName string, query *interfaces.Query, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) UpdateWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) DeleteWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) IncrementWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) FindWithCursor(ctx context.Context, collectionName string, query *interfaces.Query, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult { ch:=make(chan interfaces.QueryResult,1); ch<-&fakeQueryResult{}; return ch }
func (f *fakeRepo) CountWithFilter(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.CountResult { ch:=make(chan interfaces.CountResult,1); ch<-interfaces.CountResult{}; return ch }

type fakeQueryResult struct{ idx int; docs []map[string]interface{} }
func (r *fakeQueryResult) Next() bool { if r.idx < len(r.docs) { r.idx++; return true }; return false }
func (r *fakeQueryResult) Decode(v interface{}) error { m := r.docs[r.idx-1]; ptr, ok := v.(*map[string]interface{}); if ok { *ptr = m }; return nil }
func (r *fakeQueryResult) Close() {}
func (r *fakeQueryResult) Error() error { return nil }

type fakeSingleResult struct{ doc map[string]interface{} }
func (r *fakeSingleResult) Decode(v interface{}) error { ptr, ok := v.(*map[string]interface{}); if ok { *ptr = r.doc }; return nil }
func (r *fakeSingleResult) Error() error { return nil }
func (r *fakeSingleResult) NoResult() bool { return r.doc == nil }

func createTestPlatformConfig() *platformconfig.Config {
	return &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
}

func newServiceWithFakeRepo(fr *fakeRepo) *services.Service {
    base := &platform.BaseService{ Repository: fr }
    return services.NewService(base, createTestPlatformConfig())
}

func TestUpdateProfile_SanitizesAllowedFields(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    uid := uuid.Nil
    fullName := "A"
    avatar := "x"
    banner := "y"
    tagLine := "z"
    socialName := "s"
    req := &models.UpdateProfileRequest{
        FullName: &fullName,
        Avatar: &avatar,
        Banner: &banner,
        TagLine: &tagLine,
        SocialName: &socialName,
    }
    if err := svc.UpdateProfile(context.Background(), uid, req); err != nil { t.Fatalf("UpdateProfile err=%v", err) }
    updates, _ := fr.lastUpdateData.(map[string]interface{})
    if _, ok := updates["evil"]; ok { t.Fatalf("unexpected field 'evil' passed to update") }
    for _, k := range []string{"fullName","avatar","banner","tagLine","socialName"} {
        if _, ok := updates[k]; !ok { t.Fatalf("expected key %s in updates", k) }
    }
}

func TestIncrease_BuildsIncUpdate(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    if err := svc.Increase(context.Background(), "followCount", 2, uuid.Nil); err != nil { t.Fatalf("Increase err=%v", err) }
    increments := fr.lastIncrementFieldsIncrements
    if got := increments["followCount"]; !reflect.DeepEqual(got, 2) { t.Fatalf("increment mismatch: %v", got) }
}

func TestQuery_FilterBuilds(t *testing.T) {
    fr := &fakeRepo{ findDocs: []map[string]interface{}{{"a":1}} }
    svc := newServiceWithFakeRepo(fr)
    if _, err := svc.Query(context.Background(), "", 10, 0); err != nil { t.Fatalf("Query err=%v", err) }
    query := fr.findFilter.(*interfaces.Query)
    if query == nil || (len(query.Conditions) != 0 || len(query.OrGroups) != 0) { t.Fatalf("expected empty filter, got %v", fr.findFilter) }
    // TODO: Full-text search implementation is pending - Query currently doesn't support text search
    // When implemented, this test should check for text search conditions in the Query object
    if _, err := svc.Query(context.Background(), "bob", 10, 0); err != nil { t.Fatalf("Query err=%v", err) }
    // Note: Full-text search is not yet implemented in the Query pattern
    // This test is currently skipped until text search is properly implemented
}

func TestCreateOrUpdateDTO_UpsertFilterAndData(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    uid := uuid.Must(uuid.NewV4())
    fullName := "Bob"
    req := &models.CreateProfileRequest{
        ObjectId: uid,
        FullName: &fullName,
    }
    if err := svc.CreateOrUpdateDTO(context.Background(), req); err != nil { t.Fatalf("CreateOrUpdateDTO err=%v", err) }
    if fr.lastUpdateOptions == nil || fr.lastUpdateOptions.Upsert == nil || *fr.lastUpdateOptions.Upsert != true {
        t.Fatalf("expected upsert true, got %+v", fr.lastUpdateOptions)
    }
    query := fr.lastUpdateFilter.(*interfaces.Query)
    if query == nil { t.Fatalf("expected query object, got nil") }
    // Check that query contains object_id condition
    found := false
    for _, field := range query.Conditions {
        if field.Name == "object_id" && reflect.DeepEqual(field.Value, uid) {
            found = true
            break
        }
    }
    if !found { t.Fatalf("filter object_id mismatch: expected %v in query conditions", uid) }
    updates, _ := fr.lastUpdateData.(map[string]interface{})
    if updates["fullName"] != "Bob" { t.Fatalf("update fullName mismatch: %v", updates["fullName"]) }
}

func TestUpdateLastSeen_SetsField(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    uid := uuid.Nil
    if err := svc.UpdateLastSeen(context.Background(), uid, 1234); err != nil { t.Fatalf("UpdateLastSeen err=%v", err) }
    updates, _ := fr.lastUpdateData.(map[string]interface{})
    if _, ok := updates["lastSeen"]; !ok { t.Fatalf("expected lastSeen set") }
}



