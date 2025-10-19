package factory

import (
	"context"
	"testing"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

func TestRepositoryFactory_ValidateConfigErrors(t *testing.T) {
    f := NewRepositoryFactory(&interfaces.RepositoryConfig{})
    if err := f.ValidateConfig(); err == nil { t.Fatalf("expected error for empty config") }

    f = NewRepositoryFactory(&interfaces.RepositoryConfig{ DatabaseType: interfaces.DatabaseTypeMongoDB })
    if err := f.ValidateConfig(); err == nil { t.Fatalf("expected error for missing mongo config") }

    f = NewRepositoryFactory(&interfaces.RepositoryConfig{ DatabaseType: interfaces.DatabaseTypePostgreSQL })
    if err := f.ValidateConfig(); err == nil { t.Fatalf("expected error for missing postgres config") }
}

func TestRepositoryFactory_CreateUnsupported(t *testing.T) {
    f := NewRepositoryFactory(&interfaces.RepositoryConfig{ DatabaseType: "oracle" })
    if _, err := f.CreateRepository(context.Background()); err == nil { t.Fatalf("expected unsupported error") }
}


