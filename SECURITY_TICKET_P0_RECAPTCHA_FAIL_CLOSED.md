# P0 Security Ticket: Recaptcha Fail Closed Refactoring

## Priority: P0 (Critical Security)

## Issue Description

The current Recaptcha implementation uses a "Fail Open" design pattern, which poses a security risk in production environments.

### Current Behavior

**Location:** `apps/api/cmd/server/main.go:167`
```go
signupHandler := signupUC.NewHandler(signupService, "", privateKey)
```

**Problem:**
- When `RECAPTCHA_KEY` is empty or not set, `NewGoogleVerifier("")` returns `nil`
- Handler skips recaptcha verification if `recaptchaVerifier` is `nil` (see `apps/api/auth/signup/handler.go:86`)
- This means if a DevOps engineer forgets to set `RECAPTCHA_KEY` in production, bot protection is silently disabled

### Security Risk

**Fail Open Design Flaw:**
- If `RECAPTCHA_KEY` is missing in production, the API will silently disable bot protection
- Anyone can sign up without recaptcha verification
- No warning or error is raised

### Enterprise Standard

Security features should **Fail Closed**, not **Fail Open**:
- If `RECAPTCHA_KEY` is missing, the server should either:
  1. **Panic on Startup** (preventing deployment), OR
  2. **Reject All Signups** (preventing abuse)
- It should almost never "just skip verification"

## Proposed Solution

### Option A: Fail Closed with Startup Panic (Recommended)

Modify `apps/api/cmd/server/main.go`:
```go
recaptchaKey := cfg.RecaptchaKey
if recaptchaKey == "" {
    log.Fatalf("SECURITY: RECAPTCHA_KEY is required in production. Set RECAPTCHA_KEY environment variable or disable recaptcha explicitly via RECAPTCHA_DISABLED=true")
}
signupHandler := signupUC.NewHandler(signupService, recaptchaKey, privateKey)
```

### Option B: Fail Closed with Explicit Disable Flag

Allow explicit disabling for E2E/testing:
```go
recaptchaKey := cfg.RecaptchaKey
if recaptchaKey == "" {
    if os.Getenv("RECAPTCHA_DISABLED") != "true" {
        log.Fatalf("SECURITY: RECAPTCHA_KEY is required. Set RECAPTCHA_KEY or RECAPTCHA_DISABLED=true")
    }
    // Use FakeVerifier for E2E
    signupHandler := signupUC.NewHandler(signupService, "", privateKey)
    signupHandler = signupHandler.WithRecaptcha(&testutil.FakeRecaptchaVerifier{ShouldSucceed: true})
} else {
    signupHandler := signupUC.NewHandler(signupService, recaptchaKey, privateKey)
}
```

### Option C: Reject Signups When Verifier is Nil

Modify `apps/api/auth/signup/handler.go`:
```go
// Recaptcha validation via injected verifier
if h.recaptchaVerifier == nil {
    // Fail closed: reject signups if recaptcha is not configured
    return errors.HandleSystemError(c, "Recaptcha verification is required but not configured. Contact system administrator.")
}
success, err := h.recaptchaVerifier.Verify(c.Context(), model.Recaptcha)
```

## Implementation Notes

1. **E2E Testing:** If Option B is chosen, update `tools/dev/scripts/*_e2e_test.sh` to set `RECAPTCHA_DISABLED=true` or use Google Test Keys
2. **Documentation:** Update deployment docs to emphasize `RECAPTCHA_KEY` requirement
3. **Monitoring:** Add alerting for missing `RECAPTCHA_KEY` in production environments

## Acceptance Criteria

- [ ] Server panics or rejects signups if `RECAPTCHA_KEY` is missing (unless explicitly disabled)
- [ ] E2E tests continue to pass with new configuration
- [ ] Documentation updated with security requirements
- [ ] No silent failures in production

## Related Files

- `apps/api/cmd/server/main.go:167`
- `apps/api/auth/signup/handler.go:86-94`
- `apps/api/auth/signup/handler.go:25-36` (NewHandler)
- `apps/api/internal/recaptcha/google_verifier.go:22-30` (NewGoogleVerifier)

## Created

Date: $(date)
Reason: Identified during E2E script fix verification - "Fail Open" security architecture weakness

