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

    findDocs []map[string]interface{}
    findFilter interface{}
    findOptions *interfaces.FindOptions

    singleDoc map[string]interface{}
}

func (f *fakeRepo) Save(ctx context.Context, collectionName string, data interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Find(ctx context.Context, collectionName string, filter interface{}, options *interfaces.FindOptions) <-chan interfaces.QueryResult {
    f.findFilter = filter
    f.findOptions = options
    ch := make(chan interfaces.QueryResult,1)
    ch <- &fakeQueryResult{docs: f.findDocs}
    return ch
}
func (f *fakeRepo) FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.SingleResult {
    ch := make(chan interfaces.SingleResult,1)
    ch <- &fakeSingleResult{doc: f.singleDoc}
    return ch
}
func (f *fakeRepo) Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, options *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
    f.lastUpdateFilter = filter
    f.lastUpdateData = data
    f.lastUpdateOptions = options
    ch := make(chan interfaces.RepositoryResult,1)
    ch <- interfaces.RepositoryResult{}
    return ch
}
func (f *fakeRepo) UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, options *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Delete(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) Count(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult { ch:=make(chan interfaces.CountResult,1); ch<-interfaces.CountResult{}; return ch }
func (f *fakeRepo) BulkWrite(ctx context.Context, collectionName string, operations []interfaces.BulkOperation) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error { f.lastCreateIndex = indexes; ch:=make(chan error,1); ch<-nil; return ch }
func (f *fakeRepo) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) { return nil, nil }
func (f *fakeRepo) Begin(ctx context.Context) (interfaces.Transaction, error) { return nil, nil }
func (f *fakeRepo) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) { return nil, nil }
func (f *fakeRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }
func (f *fakeRepo) Ping(ctx context.Context) <-chan error { ch:=make(chan error,1); ch<-nil; return ch }
func (f *fakeRepo) Close() error { return nil }
func (f *fakeRepo) UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) UpdateWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) DeleteWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) IncrementWithOwnership(ctx context.Context, collectionName string, postID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult { ch:=make(chan interfaces.RepositoryResult,1); ch<-interfaces.RepositoryResult{}; return ch }
func (f *fakeRepo) FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult { ch:=make(chan interfaces.QueryResult,1); ch<-&fakeQueryResult{}; return ch }
func (f *fakeRepo) CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult { ch:=make(chan interfaces.CountResult,1); ch<-interfaces.CountResult{}; return ch }

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
    updateMap, _ := fr.lastUpdateData.(map[string]interface{})
    setMap, _ := updateMap["$set"].(map[string]interface{})
    if _, ok := setMap["evil"]; ok { t.Fatalf("unexpected field 'evil' passed to update") }
    for _, k := range []string{"fullName","avatar","banner","tagLine","socialName"} {
        if _, ok := setMap[k]; !ok { t.Fatalf("expected key %s in $set", k) }
    }
}

func TestIncrease_BuildsIncUpdate(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    if err := svc.Increase(context.Background(), "followCount", 2, uuid.Nil); err != nil { t.Fatalf("Increase err=%v", err) }
    updateMap, _ := fr.lastUpdateData.(map[string]interface{})
    incMap, _ := updateMap["$inc"].(map[string]interface{})
    if got := incMap["followCount"]; !reflect.DeepEqual(got, 2) { t.Fatalf("$inc mismatch: %v", got) }
}

func TestQuery_FilterBuilds(t *testing.T) {
    fr := &fakeRepo{ findDocs: []map[string]interface{}{{"a":1}} }
    svc := newServiceWithFakeRepo(fr)
    if _, err := svc.Query(context.Background(), "", 10, 0); err != nil { t.Fatalf("Query err=%v", err) }
    if m, ok := fr.findFilter.(map[string]interface{}); !ok || len(m) != 0 { t.Fatalf("expected empty filter, got %v", fr.findFilter) }
    if _, err := svc.Query(context.Background(), "bob", 10, 0); err != nil { t.Fatalf("Query err=%v", err) }
    m, ok := fr.findFilter.(map[string]interface{})
    if !ok { t.Fatalf("filter type") }
    text, ok := m["$text"].(map[string]interface{})
    if !ok || text["$search"] != "bob" { t.Fatalf("text search mismatch: %v", m) }
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
    filt, ok := fr.lastUpdateFilter.(map[string]interface{})
    if !ok { t.Fatalf("filter type") }
    if got := filt["objectId"]; !reflect.DeepEqual(got, uid) { t.Fatalf("filter objectId mismatch: %v", got) }
    updateMap, _ := fr.lastUpdateData.(map[string]interface{})
    setMap, _ := updateMap["$set"].(map[string]interface{})
    if setMap["fullName"] != "Bob" { t.Fatalf("$set fullName mismatch: %v", setMap["fullName"]) }
}

func TestUpdateLastSeen_SetsField(t *testing.T) {
    fr := &fakeRepo{}
    svc := newServiceWithFakeRepo(fr)
    uid := uuid.Nil
    if err := svc.UpdateLastSeen(context.Background(), uid, 1234); err != nil { t.Fatalf("UpdateLastSeen err=%v", err) }
    updateMap, _ := fr.lastUpdateData.(map[string]interface{})
    setMap, _ := updateMap["$set"].(map[string]interface{})
    if _, ok := setMap["lastSeen"]; !ok { t.Fatalf("expected lastSeen set") }
}



