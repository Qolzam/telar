# Backlog Ticket: Analysis Failure Status & Re-analysis Queue

**Ticket Type:** Feature Enhancement  
**Priority:** Low  
**Status:** Backlog  
**Created:** 2024-12-23  
**Related Feature:** AI Co-Moderator (Guardian Protocol)

---

## Description

Currently, when the AI Engine service permanently fails after all retry attempts are exhausted, posts remain in `'published'` status with only error logging. This implements graceful degradation, which is acceptable for v1.

This feature would add enhanced visibility and recovery mechanisms for posts that fail AI analysis.

---

## Proposed Enhancement

### Database Changes
- Add `'analysis_failed'` status value to the `posts.status` column
- Update migration to include this new status
- Update model documentation

### Backend Changes
- Modify `triggerContentAnalysis` in `post_service.go` to set status to `'analysis_failed'` after permanent failure
- Add `FindByStatus(ctx, "analysis_failed", ...)` query support (already exists via repository)
- Create admin endpoint: `POST /api/v1/moderation/:id/retry-analysis` to manually trigger re-analysis
- Implement retry logic that respects existing retry limits

### Frontend Changes
- Add "Analysis Failed" section to moderation queue UI
- Add "Retry Analysis" button for failed posts
- Display failure reason and timestamp from moderation_details

### Business Logic Considerations
- Should `analysis_failed` posts be visible to users? (Recommendation: Yes, as graceful degradation)
- Should there be a maximum retry count per post?
- Should there be automatic periodic retry for failed posts?

---

## Acceptance Criteria

- [ ] Posts that permanently fail AI analysis are marked with `'analysis_failed'` status
- [ ] Admins can view all posts with `'analysis_failed'` status in a dedicated queue
- [ ] Admins can manually trigger re-analysis for failed posts
- [ ] Re-analysis respects existing retry limits and error classification
- [ ] Failed analysis attempts are stored in `moderation_details` for audit trail
- [ ] Frontend displays analysis failure queue with retry capability

---

## Technical Notes

- Current implementation uses graceful degradation (posts remain `'published'`) which is acceptable for production
- This feature enhances observability and recovery without changing core behavior
- Should be implemented as a minor version enhancement (v1.1) after monitoring production behavior

---

## Related Files

- `apps/api/posts/services/post_service.go:1595` (current graceful degradation implementation)
- `apps/api/posts/migrations/003_add_moderation_to_posts.sql` (status column definition)
- `apps/api/posts/repository/postgres_repository.go:FindByStatus` (existing query support)
- `apps/api/posts/handlers/post_handler.go` (moderation endpoints)

