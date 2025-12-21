package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
)

type postgresRepository struct {
	client *postgres.Client
	schema string
}

// NewPostgresRepository creates a repository using the default schema.
func NewPostgresRepository(client *postgres.Client) Repository {
	return &postgresRepository{client: client, schema: ""}
}

// NewPostgresRepositoryWithSchema creates a repository using a specific schema.
func NewPostgresRepositoryWithSchema(client *postgres.Client, schema string) Repository {
	return &postgresRepository{client: client, schema: schema}
}

func (r *postgresRepository) getExecutor(ctx context.Context) sqlx.ExtContext {
	if txVal := ctx.Value("tx"); txVal != nil {
		if tx, ok := txVal.(*sqlx.Tx); ok {
			return tx
		}
	}
	return r.client.DB()
}

func (r *postgresRepository) AddBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	query := `
		INSERT INTO %sbookmarks (owner_user_id, post_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	result, err := exec.ExecContext(ctx, sqlStr, userID, postID)
	if err != nil {
		return false, fmt.Errorf("insert bookmark: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return rows > 0, nil
}

func (r *postgresRepository) RemoveBookmark(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	query := `
		DELETE FROM %sbookmarks
		WHERE owner_user_id = $1 AND post_id = $2
	`

	exec := r.getExecutor(ctx)
	sqlStr := r.prefixSchema(query)
	result, err := exec.ExecContext(ctx, sqlStr, userID, postID)
	if err != nil {
		return false, fmt.Errorf("delete bookmark: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return rows > 0, nil
}

func (r *postgresRepository) GetMapByUserAndPosts(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}

	idStrings := make([]string, len(postIDs))
	for i, id := range postIDs {
		idStrings[i] = id.String()
	}

	query := `
		SELECT post_id
		FROM %sbookmarks
		WHERE owner_user_id = $1 AND post_id = ANY($2::uuid[])
	`

	var results []uuid.UUID
	sqlStr := r.prefixSchema(query)
	if err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &results, sqlStr, userID, pq.Array(idStrings)); err != nil {
		if err == sql.ErrNoRows {
			return map[uuid.UUID]bool{}, nil
		}
		return nil, fmt.Errorf("get bookmark map: %w", err)
	}

	resultMap := make(map[uuid.UUID]bool, len(results))
	for _, id := range results {
		resultMap[id] = true
	}

	// Ensure requested IDs exist with false default for deterministic presence.
	for _, id := range postIDs {
		if _, ok := resultMap[id]; !ok {
			resultMap[id] = false
		}
	}

	return resultMap, nil
}

func (r *postgresRepository) FindMyBookmarks(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]BookmarkEntry, string, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var createdBefore time.Time
	var idBefore uuid.UUID
	if cursor != "" {
		decoded, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		createdBefore = decoded.CreatedAt
		idBefore = decoded.PostID
	}

	var query string
	var args []interface{}
	
	if !createdBefore.IsZero() && idBefore != uuid.Nil {
		// With cursor: use tuple comparison
		query = `
			SELECT post_id, created_at
			FROM %sbookmarks
			WHERE owner_user_id = $1
			AND (created_at, post_id) < ($2, $3)
			ORDER BY created_at DESC, post_id DESC
			LIMIT $4
		`
		args = []interface{}{userID, createdBefore, idBefore, limit + 1}
	} else {
		// Without cursor: no tuple comparison needed
		query = `
			SELECT post_id, created_at
			FROM %sbookmarks
			WHERE owner_user_id = $1
			ORDER BY created_at DESC, post_id DESC
			LIMIT $2
		`
		args = []interface{}{userID, limit + 1}
	}

	sqlStr := fmt.Sprintf(query, r.schemaPrefix())

	var rows []BookmarkEntry
	if err := sqlx.SelectContext(ctx, r.getExecutor(ctx), &rows, sqlStr, args...); err != nil {
		return nil, "", fmt.Errorf("find bookmarks: %w", err)
	}

	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}

	nextCursor := ""
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = encodeCursor(last.CreatedAt, last.PostID)
	}

	return rows, nextCursor, nil
}

func (r *postgresRepository) prefixSchema(query string) string {
	if r.schema == "" {
		return fmt.Sprintf(query, "")
	}
	return fmt.Sprintf(query, r.schema+".")
}

func (r *postgresRepository) schemaPrefix() string {
	if r.schema == "" {
		return ""
	}
	return r.schema + "."
}

type cursorData struct {
	CreatedAt time.Time `json:"createdAt"`
	PostID    uuid.UUID `json:"postId"`
}

func encodeCursor(ts time.Time, id uuid.UUID) string {
	payload := cursorData{CreatedAt: ts.UTC(), PostID: id}
	b, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursorData, error) {
	var cd cursorData
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return cd, err
	}
	if err := json.Unmarshal(raw, &cd); err != nil {
		return cd, err
	}
	return cd, nil
}
