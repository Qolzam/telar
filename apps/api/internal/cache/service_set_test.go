package cache

import (
	"context"
	"testing"
	"time"
)

// fakeSetCache implements Cache and setOps for unit testing GenericCacheService set helpers
type fakeSetCache struct {
	kv         map[string][]byte
	setMembers map[string]map[string]struct{}
}

func newFakeSetCache() *fakeSetCache {
	return &fakeSetCache{
		kv:         map[string][]byte{},
		setMembers: map[string]map[string]struct{}{},
	}
}

func (f *fakeSetCache) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := f.kv[key]; ok {
		return v, nil
	}
	return nil, ErrKeyNotFound
}
func (f *fakeSetCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.kv[key] = value
	return nil
}
func (f *fakeSetCache) Delete(ctx context.Context, key string) error {
	delete(f.kv, key)
	delete(f.setMembers, key)
	return nil
}
func (f *fakeSetCache) DeletePattern(ctx context.Context, pattern string) error { return nil }
func (f *fakeSetCache) Exists(ctx context.Context, key string) (bool, error)   { _, ok := f.kv[key]; return ok, nil }
func (f *fakeSetCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}
func (f *fakeSetCache) Close() error { return nil }
func (f *fakeSetCache) Stats() CacheStats {
	return CacheStats{}
}
func (f *fakeSetCache) SetAdd(ctx context.Context, key string, member string) error {
	if _, ok := f.setMembers[key]; !ok {
		f.setMembers[key] = map[string]struct{}{}
	}
	f.setMembers[key][member] = struct{}{}
	return nil
}
func (f *fakeSetCache) SetIsMember(ctx context.Context, key string, member string) (bool, error) {
	if m, ok := f.setMembers[key]; ok {
		_, ok := m[member]
		return ok, nil
	}
	return false, nil
}

func TestGenericCacheService_SetOps(t *testing.T) {
	cfg := DefaultCacheConfig()
	cfg.Enabled = true
	cfg.Prefix = "test:"
	fc := newFakeSetCache()
	svc := NewGenericCacheService(fc, cfg)

	key := svc.GenerateHashKey("sessions", map[string]interface{}{"uid": "u1"})

	// Initially not a member
	isMember, err := svc.SetIsMember(context.Background(), key, "j1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isMember {
		t.Fatalf("expected not member initially")
	}

	// Add member
	if err := svc.SetAdd(context.Background(), key, "j1"); err != nil {
		t.Fatalf("unexpected error on SetAdd: %v", err)
	}

	// Should be a member now
	isMember, err = svc.SetIsMember(context.Background(), key, "j1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isMember {
		t.Fatalf("expected member after SetAdd")
	}
}






