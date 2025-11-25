package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

// Test Post struct field validation and JSON marshaling/unmarshaling
func TestPost_JSONMarshaling(t *testing.T) {
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	albumCoverID := uuid.Must(uuid.NewV4())
	
	// Create a test post with all fields
	originalPost := &Post{
		ObjectId:         postID,
		PostTypeId:       1,
		Score:            100,
		Votes:            map[string]string{"user1": "up", "user2": "down"},
		ViewCount:        500,
		Body:             "This is a test post body",
		OwnerUserId:      userID,
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "https://example.com/avatar.jpg",
		URLKey:           "test-post-url-key",
		Tags:             []string{"test", "golang", "api"},
		CommentCounter:   25,
		Image:            "https://example.com/image.jpg",
		ImageFullPath:    "https://example.com/full/image.jpg",
		Video:            "https://example.com/video.mp4",
		Thumbnail:        "https://example.com/thumbnail.jpg",
		Album: &Album{
			Count:   3,
			Cover:   "cover.jpg",
			CoverId: albumCoverID,
			Photos:  []string{"photo1.jpg", "photo2.jpg", "photo3.jpg"},
			Title:   "Test Album",
		},
		DisableComments:  false,
		DisableSharing:   true,
		Deleted:          false,
		DeletedDate:      0,
		CreatedDate:      time.Now().Unix(),
		LastUpdated:      time.Now().Unix(),
		AccessUserList:   []string{"user1", "user2"},
		Permission:       "Public",
		Version:          "1.0",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalPost)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Unmarshal back to Post
	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, originalPost.ObjectId, unmarshaledPost.ObjectId)
	assert.Equal(t, originalPost.PostTypeId, unmarshaledPost.PostTypeId)
	assert.Equal(t, originalPost.Score, unmarshaledPost.Score)
	assert.Equal(t, originalPost.Votes, unmarshaledPost.Votes)
	assert.Equal(t, originalPost.ViewCount, unmarshaledPost.ViewCount)
	assert.Equal(t, originalPost.Body, unmarshaledPost.Body)
	assert.Equal(t, originalPost.OwnerUserId, unmarshaledPost.OwnerUserId)
	assert.Equal(t, originalPost.OwnerDisplayName, unmarshaledPost.OwnerDisplayName)
	assert.Equal(t, originalPost.OwnerAvatar, unmarshaledPost.OwnerAvatar)
	assert.Equal(t, originalPost.URLKey, unmarshaledPost.URLKey)
	assert.Equal(t, originalPost.Tags, unmarshaledPost.Tags)
	assert.Equal(t, originalPost.CommentCounter, unmarshaledPost.CommentCounter)
	assert.Equal(t, originalPost.Image, unmarshaledPost.Image)
	assert.Equal(t, originalPost.ImageFullPath, unmarshaledPost.ImageFullPath)
	assert.Equal(t, originalPost.Video, unmarshaledPost.Video)
	assert.Equal(t, originalPost.Thumbnail, unmarshaledPost.Thumbnail)
	assert.Equal(t, originalPost.DisableComments, unmarshaledPost.DisableComments)
	assert.Equal(t, originalPost.DisableSharing, unmarshaledPost.DisableSharing)
	assert.Equal(t, originalPost.Deleted, unmarshaledPost.Deleted)
	assert.Equal(t, originalPost.DeletedDate, unmarshaledPost.DeletedDate)
	assert.Equal(t, originalPost.CreatedDate, unmarshaledPost.CreatedDate)
	assert.Equal(t, originalPost.LastUpdated, unmarshaledPost.LastUpdated)
	assert.Equal(t, originalPost.AccessUserList, unmarshaledPost.AccessUserList)
	assert.Equal(t, originalPost.Permission, unmarshaledPost.Permission)
	assert.Equal(t, originalPost.Version, unmarshaledPost.Version)

	// Verify album fields
	assert.NotNil(t, unmarshaledPost.Album)
	assert.Equal(t, originalPost.Album.Count, unmarshaledPost.Album.Count)
	assert.Equal(t, originalPost.Album.Cover, unmarshaledPost.Album.Cover)
	assert.Equal(t, originalPost.Album.CoverId, unmarshaledPost.Album.CoverId)
	assert.Equal(t, originalPost.Album.Photos, unmarshaledPost.Album.Photos)
	assert.Equal(t, originalPost.Album.Title, unmarshaledPost.Album.Title)
}

// Test Post struct with minimal fields
func TestPost_MinimalFields(t *testing.T) {
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	
	// Create a minimal post
	minimalPost := &Post{
		ObjectId:    postID,
		PostTypeId:  1,
		Body:        "Minimal post",
		OwnerUserId: userID,
		Permission:  "Public",
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(minimalPost)
	assert.NoError(t, err)

	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	// Verify essential fields
	assert.Equal(t, minimalPost.ObjectId, unmarshaledPost.ObjectId)
	assert.Equal(t, minimalPost.PostTypeId, unmarshaledPost.PostTypeId)
	assert.Equal(t, minimalPost.Body, unmarshaledPost.Body)
	assert.Equal(t, minimalPost.OwnerUserId, unmarshaledPost.OwnerUserId)
	assert.Equal(t, minimalPost.Permission, unmarshaledPost.Permission)
	
	// Verify default values for omitted fields
	assert.Equal(t, int64(0), unmarshaledPost.Score)
	assert.Equal(t, int64(0), unmarshaledPost.ViewCount)
	assert.Equal(t, int64(0), unmarshaledPost.CommentCounter)
	assert.False(t, unmarshaledPost.DisableComments)
	assert.False(t, unmarshaledPost.DisableSharing)
	assert.False(t, unmarshaledPost.Deleted)
	assert.Equal(t, int64(0), unmarshaledPost.DeletedDate)
	assert.Equal(t, int64(0), unmarshaledPost.CreatedDate)
	assert.Equal(t, int64(0), unmarshaledPost.LastUpdated)
}

// Test Post permission field handling with migration compatibility
func TestPost_PermissionMigration(t *testing.T) {
	testCases := []struct {
		name           string
		jsonInput      string
		expectedOutput string
	}{
		{
			name:           "String permission - Public",
			jsonInput:      `{"permission": "Public"}`,
			expectedOutput: "Public",
		},
		{
			name:           "String permission - OnlyMe",
			jsonInput:      `{"permission": "OnlyMe"}`,
			expectedOutput: "OnlyMe",
		},
		{
			name:           "String permission - Circles",
			jsonInput:      `{"permission": "Circles"}`,
			expectedOutput: "Circles",
		},
		{
			name:           "Numeric permission - 0 (Public)",
			jsonInput:      `{"permission": 0}`,
			expectedOutput: "Public",
		},
		{
			name:           "Numeric permission - 1 (OnlyMe)",
			jsonInput:      `{"permission": 1}`,
			expectedOutput: "OnlyMe",
		},
		{
			name:           "Numeric permission - 2 (Circles)",
			jsonInput:      `{"permission": 2}`,
			expectedOutput: "Circles",
		},
		{
			name:           "Invalid numeric permission - 999",
			jsonInput:      `{"permission": 999}`,
			expectedOutput: "Public", // Should default to Public
		},
		{
			name:           "Float numeric permission - 1.0",
			jsonInput:      `{"permission": 1.0}`,
			expectedOutput: "OnlyMe",
		},
		{
			name:           "Null permission",
			jsonInput:      `{"permission": null}`,
			expectedOutput: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var post Post
			err := json.Unmarshal([]byte(tc.jsonInput), &post)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, post.Permission)
		})
	}
}

// Test Album struct validation
func TestAlbum_Validation(t *testing.T) {
	albumID := uuid.Must(uuid.NewV4())
	
	album := &Album{
		Count:   5,
		Cover:   "album-cover.jpg",
		CoverId: albumID,
		Photos:  []string{"photo1.jpg", "photo2.jpg", "photo3.jpg", "photo4.jpg", "photo5.jpg"},
		Title:   "My Test Album",
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(album)
	assert.NoError(t, err)

	var unmarshaledAlbum Album
	err = json.Unmarshal(jsonData, &unmarshaledAlbum)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, album.Count, unmarshaledAlbum.Count)
	assert.Equal(t, album.Cover, unmarshaledAlbum.Cover)
	assert.Equal(t, album.CoverId, unmarshaledAlbum.CoverId)
	assert.Equal(t, album.Photos, unmarshaledAlbum.Photos)
	assert.Equal(t, album.Title, unmarshaledAlbum.Title)
}

// Test CreatePostRequest validation
func TestCreatePostRequest_Validation(t *testing.T) {
	objectID := uuid.Must(uuid.NewV4())
	
	request := &CreatePostRequest{
		ObjectId:         &objectID,
		PostTypeId:       1,
		Body:             "Test post body",
		Image:            "test-image.jpg",
		ImageFullPath:    "https://example.com/test-image.jpg",
		Video:            "test-video.mp4",
		Thumbnail:        "test-thumbnail.jpg",
		Tags:             []string{"test", "validation"},
		Album: Album{
			Count:   2,
			Cover:   "album-cover.jpg",
			CoverId: uuid.Must(uuid.NewV4()),
			Photos:  []string{"photo1.jpg", "photo2.jpg"},
			Title:   "Test Album",
		},
		DisableComments: true,
		DisableSharing:  false,
		AccessUserList:  []string{"user1", "user2"},
		Permission:      "Circles",
		Version:         "1.0",
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(request)
	assert.NoError(t, err)

	var unmarshaledRequest CreatePostRequest
	err = json.Unmarshal(jsonData, &unmarshaledRequest)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, request.ObjectId, unmarshaledRequest.ObjectId)
	assert.Equal(t, request.PostTypeId, unmarshaledRequest.PostTypeId)
	assert.Equal(t, request.Body, unmarshaledRequest.Body)
	assert.Equal(t, request.Image, unmarshaledRequest.Image)
	assert.Equal(t, request.ImageFullPath, unmarshaledRequest.ImageFullPath)
	assert.Equal(t, request.Video, unmarshaledRequest.Video)
	assert.Equal(t, request.Thumbnail, unmarshaledRequest.Thumbnail)
	assert.Equal(t, request.Tags, unmarshaledRequest.Tags)
	assert.Equal(t, request.DisableComments, unmarshaledRequest.DisableComments)
	assert.Equal(t, request.DisableSharing, unmarshaledRequest.DisableSharing)
	assert.Equal(t, request.AccessUserList, unmarshaledRequest.AccessUserList)
	assert.Equal(t, request.Permission, unmarshaledRequest.Permission)
	assert.Equal(t, request.Version, unmarshaledRequest.Version)
}

// Test UpdatePostRequest validation
func TestUpdatePostRequest_Validation(t *testing.T) {
	objectID := uuid.Must(uuid.NewV4())
	body := "Updated body"
	image := "updated-image.jpg"
	tags := []string{"updated", "tags"}
	disableComments := true
	
	request := &UpdatePostRequest{
		ObjectId:        &objectID,
		Body:            &body,
		Image:           &image,
		Tags:            &tags,
		DisableComments: &disableComments,
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(request)
	assert.NoError(t, err)

	var unmarshaledRequest UpdatePostRequest
	err = json.Unmarshal(jsonData, &unmarshaledRequest)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, request.ObjectId, unmarshaledRequest.ObjectId)
	assert.Equal(t, request.Body, unmarshaledRequest.Body)
	assert.Equal(t, request.Image, unmarshaledRequest.Image)
	assert.Equal(t, request.Tags, unmarshaledRequest.Tags)
	assert.Equal(t, request.DisableComments, unmarshaledRequest.DisableComments)
}

// Test PostQueryFilter validation
func TestPostQueryFilter_Validation(t *testing.T) {
	userID := uuid.Must(uuid.NewV4())
	postTypeID := 1
	deleted := false
	createdAfter := time.Now().Add(-24 * time.Hour)
	
	filter := &PostQueryFilter{
		OwnerUserId:  &userID,
		PostTypeId:   &postTypeID,
		Tags:         []string{"filter", "test"},
		Search:       "test search",
		Deleted:      &deleted,
		Page:         1,
		Limit:        20,
		SortBy:       "createdDate",
		SortOrder:    "desc",
		CreatedAfter: &createdAfter,
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(filter)
	assert.NoError(t, err)

	var unmarshaledFilter PostQueryFilter
	err = json.Unmarshal(jsonData, &unmarshaledFilter)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, filter.OwnerUserId, unmarshaledFilter.OwnerUserId)
	assert.Equal(t, filter.PostTypeId, unmarshaledFilter.PostTypeId)
	assert.Equal(t, filter.Tags, unmarshaledFilter.Tags)
	assert.Equal(t, filter.Search, unmarshaledFilter.Search)
	assert.Equal(t, filter.Deleted, unmarshaledFilter.Deleted)
	assert.Equal(t, filter.Page, unmarshaledFilter.Page)
	assert.Equal(t, filter.Limit, unmarshaledFilter.Limit)
	assert.Equal(t, filter.SortBy, unmarshaledFilter.SortBy)
	assert.Equal(t, filter.SortOrder, unmarshaledFilter.SortOrder)
	// Note: Time precision may vary during JSON marshaling/unmarshaling
	assert.WithinDuration(t, *filter.CreatedAfter, *unmarshaledFilter.CreatedAfter, time.Second)
}

// Test PostResponse validation
func TestPostResponse_Validation(t *testing.T) {
	response := &PostResponse{
		ObjectId:         uuid.Must(uuid.NewV4()).String(),
		PostTypeId:       1,
		Score:            100,
		Votes:            map[string]string{"user1": "up"},
		ViewCount:        500,
		Body:             "Response test body",
		OwnerUserId:      uuid.Must(uuid.NewV4()).String(),
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "https://example.com/avatar.jpg",
		Tags:             []string{"response", "test"},
		CommentCounter:   10,
		Image:            "response-image.jpg",
		URLKey:           "response-url-key",
		DisableComments:  false,
		DisableSharing:   true,
		Deleted:          false,
		CreatedDate:      time.Now().Unix(),
		Permission:       "Public",
		Version:          "1.0",
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaledResponse PostResponse
	err = json.Unmarshal(jsonData, &unmarshaledResponse)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, response.ObjectId, unmarshaledResponse.ObjectId)
	assert.Equal(t, response.PostTypeId, unmarshaledResponse.PostTypeId)
	assert.Equal(t, response.Score, unmarshaledResponse.Score)
	assert.Equal(t, response.Votes, unmarshaledResponse.Votes)
	assert.Equal(t, response.ViewCount, unmarshaledResponse.ViewCount)
	assert.Equal(t, response.Body, unmarshaledResponse.Body)
	assert.Equal(t, response.OwnerUserId, unmarshaledResponse.OwnerUserId)
	assert.Equal(t, response.OwnerDisplayName, unmarshaledResponse.OwnerDisplayName)
	assert.Equal(t, response.OwnerAvatar, unmarshaledResponse.OwnerAvatar)
	assert.Equal(t, response.Tags, unmarshaledResponse.Tags)
	assert.Equal(t, response.CommentCounter, unmarshaledResponse.CommentCounter)
	assert.Equal(t, response.Image, unmarshaledResponse.Image)
	assert.Equal(t, response.URLKey, unmarshaledResponse.URLKey)
	assert.Equal(t, response.DisableComments, unmarshaledResponse.DisableComments)
	assert.Equal(t, response.DisableSharing, unmarshaledResponse.DisableSharing)
	assert.Equal(t, response.Deleted, unmarshaledResponse.Deleted)
	assert.Equal(t, response.CreatedDate, unmarshaledResponse.CreatedDate)
	assert.Equal(t, response.Permission, unmarshaledResponse.Permission)
	assert.Equal(t, response.Version, unmarshaledResponse.Version)
}

// Test PostsListResponse validation
func TestPostsListResponse_Validation(t *testing.T) {
	posts := []PostResponse{
		{
			ObjectId:   uuid.Must(uuid.NewV4()).String(),
			PostTypeId: 1,
			Body:       "First post",
		},
		{
			ObjectId:   uuid.Must(uuid.NewV4()).String(),
			PostTypeId: 2,
			Body:       "Second post",
		},
	}

	response := &PostsListResponse{
		Posts:      posts,
		TotalCount: 25,
		Page:       2,
		Limit:      10,
		HasMore:    true,
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaledResponse PostsListResponse
	err = json.Unmarshal(jsonData, &unmarshaledResponse)
	assert.NoError(t, err)

	// Verify all fields
	assert.Len(t, unmarshaledResponse.Posts, 2)
	assert.Equal(t, response.TotalCount, unmarshaledResponse.TotalCount)
	assert.Equal(t, response.Page, unmarshaledResponse.Page)
	assert.Equal(t, response.Limit, unmarshaledResponse.Limit)
	assert.Equal(t, response.HasMore, unmarshaledResponse.HasMore)
}

// Test CreatePostResponse validation
func TestCreatePostResponse_Validation(t *testing.T) {
	response := &CreatePostResponse{
		ObjectId: uuid.Must(uuid.NewV4()).String(),
		Message:  "Post created successfully",
	}

	// Marshal and unmarshal
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaledResponse CreatePostResponse
	err = json.Unmarshal(jsonData, &unmarshaledResponse)
	assert.NoError(t, err)

	// Verify all fields
	assert.Equal(t, response.ObjectId, unmarshaledResponse.ObjectId)
	assert.Equal(t, response.Message, unmarshaledResponse.Message)
}

// Test UUID field validation
func TestPost_UUIDValidation(t *testing.T) {
	// Test valid UUID
	validUUID := uuid.Must(uuid.NewV4())
	post := &Post{
		ObjectId:    validUUID,
		OwnerUserId: validUUID,
	}

	jsonData, err := json.Marshal(post)
	assert.NoError(t, err)

	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	assert.Equal(t, validUUID, unmarshaledPost.ObjectId)
	assert.Equal(t, validUUID, unmarshaledPost.OwnerUserId)
}

// Test required field validation (conceptual - actual validation happens at handler level)
func TestCreatePostRequest_RequiredFields(t *testing.T) {
	// Test with all required fields
	validRequest := &CreatePostRequest{
		PostTypeId: 1,
		Body:       "Valid post body",
	}

	jsonData, err := json.Marshal(validRequest)
	assert.NoError(t, err)

	var unmarshaledRequest CreatePostRequest
	err = json.Unmarshal(jsonData, &unmarshaledRequest)
	assert.NoError(t, err)

	assert.Equal(t, validRequest.PostTypeId, unmarshaledRequest.PostTypeId)
	assert.Equal(t, validRequest.Body, unmarshaledRequest.Body)
}

// Test empty and nil values
func TestPost_EmptyAndNilValues(t *testing.T) {
	// Test with empty/nil values
	post := &Post{
		ObjectId:    uuid.Must(uuid.NewV4()),
		OwnerUserId: uuid.Must(uuid.NewV4()),
		PostTypeId:  1,
		Body:        "",
		Tags:        []string{},
		Votes:       map[string]string{},
		Album:       nil,
	}

	jsonData, err := json.Marshal(post)
	assert.NoError(t, err)

	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	assert.Equal(t, "", unmarshaledPost.Body)
	assert.Empty(t, unmarshaledPost.Tags)
	assert.Empty(t, unmarshaledPost.Votes)
	assert.Nil(t, unmarshaledPost.Album)
}

// Test large data handling
func TestPost_LargeDataHandling(t *testing.T) {
	// Test with large data
	largeBody := string(make([]byte, 10000)) // 10KB body
	largeTags := make([]string, 100)
	for i := range largeTags {
		largeTags[i] = "tag" + string(rune(i))
	}

	post := &Post{
		ObjectId:    uuid.Must(uuid.NewV4()),
		OwnerUserId: uuid.Must(uuid.NewV4()),
		PostTypeId:  1,
		Body:        largeBody,
		Tags:        largeTags,
	}

	// Should handle large data without issues
	jsonData, err := json.Marshal(post)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	assert.Equal(t, largeBody, unmarshaledPost.Body)
	assert.Equal(t, []string(largeTags), []string(unmarshaledPost.Tags))
}

// Test special characters and unicode
func TestPost_SpecialCharactersAndUnicode(t *testing.T) {
	// Test with special characters and unicode
	specialBody := "Special chars: 💀☠️🔥💯 and unicode: 你好世界 and emojis: 🚀🎉"
	unicodeTags := []string{"🏷️tag", "标签", "тег"}

	post := &Post{
		ObjectId:         uuid.Must(uuid.NewV4()),
		OwnerUserId:      uuid.Must(uuid.NewV4()),
		PostTypeId:       1,
		Body:             specialBody,
		Tags:             unicodeTags,
		OwnerDisplayName: "用户名 with 🌟",
	}

	jsonData, err := json.Marshal(post)
	assert.NoError(t, err)

	var unmarshaledPost Post
	err = json.Unmarshal(jsonData, &unmarshaledPost)
	assert.NoError(t, err)

	assert.Equal(t, specialBody, unmarshaledPost.Body)
	assert.Equal(t, unicodeTags, []string(unmarshaledPost.Tags))
	assert.Equal(t, "用户名 with 🌟", unmarshaledPost.OwnerDisplayName)
}

// Test BSON tag compatibility (conceptual test)
func TestPost_BSONTagCompatibility(t *testing.T) {
	// This test ensures BSON tags remain for legacy BSON client compatibility
	// In a real scenario, this would test actual BSON marshaling
	
	post := &Post{
		ObjectId:    uuid.Must(uuid.NewV4()),
		OwnerUserId: uuid.Must(uuid.NewV4()),
		PostTypeId:  1,
		Body:        "BSON test",
	}

	// JSON marshaling should use json tags (camelCase)
	jsonData, err := json.Marshal(post)
	assert.NoError(t, err)
	
	// Verify camelCase in JSON
	assert.Contains(t, string(jsonData), "objectId")
	assert.Contains(t, string(jsonData), "ownerUserId")
	assert.Contains(t, string(jsonData), "postTypeId")
	
	// Should not contain snake_case or other formats
	assert.NotContains(t, string(jsonData), "object_id")
	assert.NotContains(t, string(jsonData), "owner_user_id")
}

// Test edge cases in permission migration
func TestPost_PermissionMigrationEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Very large number",
			input:    `{"permission": 9999999}`,
			expected: "Public",
		},
		{
			name:     "Negative number",
			input:    `{"permission": -1}`,
			expected: "Public",
		},
		{
			name:     "String number",
			input:    `{"permission": "1"}`,
			expected: "1", // Should be treated as string
		},
		{
			name:     "Boolean true",
			input:    `{"permission": true}`,
			expected: "Public", // Should default to Public
		},
		{
			name:     "Boolean false",
			input:    `{"permission": false}`,
			expected: "Public", // Should default to Public
		},
		{
			name:     "Array",
			input:    `{"permission": [1, 2]}`,
			expected: "Public", // Should default to Public
		},
		{
			name:     "Object",
			input:    `{"permission": {"type": "public"}}`,
			expected: "Public", // Should default to Public
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			var post Post
			err := json.Unmarshal([]byte(tc.input), &post)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, post.Permission)
		})
	}
}
