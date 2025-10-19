# Authentication Microservice - Complete Flow Analysis
**Updated for Phase 1 Security Refactoring (September 2025)**

## 📋 Executive Summary

This document provides a comprehensive, route-by-route analysis of the Telar Authentication Microservice after Phase 1 Critical Security Remediation. The analysis reflects the latest security implementations including JWT verification token elimination and canonical HMAC signing. Each route includes detailed flow diagrams, code analysis, and complete request/response cycles.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   Fiber Router  │    │   Auth Handler  │    │  Auth Service   │
│                 │───▶│                 │───▶│                 │───▶│                 │
│ (Postman/Web)   │    │   /auth/*       │    │  (Business Logic)│    │ (Domain Logic)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │                        │
                                                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Response │    │   Error Handler │    │   Base Service  │    │   Repository    │
│                 │◀───│                 │◀───│                 │◀───│                 │
│  (JSON/HTML)    │    │ (Error Mapping) │    │  (Platform)     │    │ (MongoDB/PG)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🔗 Route Mappings

Based on `routes.go` analysis, here are all authentication endpoints:

### Admin Routes (HMAC Protected with Rate Limiting)
- `POST /auth/admin/check` - Check if admin exists
- `POST /auth/admin/signup` - Create new admin  
- `POST /auth/admin/login` - Admin login

**Security Update**: All admin routes now use HMAC authentication with rate limiting:
- `X-Telar-Signature`: HMAC-SHA256 signature with `sha256=` prefix
- `uid`: User ID (UUID format)
- `X-Timestamp`: Unix timestamp for replay attack prevention
- **Rate Limiting**: Login rate limits applied to admin endpoints

### Public Routes (with Rate Limiting)
- `POST /auth/signup` - User registration (returns secure verificationId)
- `GET /auth/signup` - Signup page (SSR)
- `POST /auth/signup/verify` - Verify signup code (uses verificationId)

**Security Update**: Signup flow now uses secure verificationId system instead of JWT tokens:
- Signup returns: `{"verificationId": "uuid", "expiresAt": timestamp, "message": "..."}`
- Verification accepts: `{"verificationId": "uuid", "code": "123456"}`
- **NO JWT tokens containing plaintext passwords**
- **Rate Limiting**: 10 signups per hour per IP, 10 verification attempts per verification ID

### Password Routes (with Rate Limiting)
- `GET /auth/password/reset/:verifyId` - Reset password page
- `POST /auth/password/reset/:verifyId` - Submit new password (3 resets per hour per IP)
- `GET /auth/password/forget` - Forget password page
- `POST /auth/password/forget` - Request password reset (3 requests per hour per IP)
- `PUT /auth/password/change` - Change password (JWT protected + rate limited)

### Login Routes (with Rate Limiting)
- `GET /auth/login` - Login page (SSR)
- `POST /auth/login` - User login (5 attempts per 15 minutes per IP)
- `GET /auth/login/github` - GitHub OAuth redirect
- `GET /auth/login/google` - Google OAuth redirect

### OAuth Routes
- `GET /auth/oauth2/authorized` - OAuth callback handler (full OAuth flow implemented)

### JWKS Route (Public)
- `GET /auth/.well-known/jwks.json` - JSON Web Key Set endpoint

### Profile Routes (JWT Protected)
- `PUT /auth/profile` - Update user profile

---

## 📊 Complete Route Flow Analysis

### 1. Admin Check Route

**Route:** `POST /auth/admin/check`  
**Middleware:** HMAC Authentication + Rate Limiting  
**Purpose:** Check if any admin user exists in the system

**Security Update**: Now requires HMAC signing with rate limiting and timestamp validation

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/admin/check
    │   ├── Headers: {
    │   │     X-Telar-Signature: "sha256=canonical_hmac_signature",
    │   │     uid: "123e4567-e89b-12d3-a456-426614174000",
    │   │     X-Timestamp: "1642781234",
    │   │     Content-Type: "application/json"
    │   │   }
    │   └── Body: {} (empty)
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    MIDDLEWARE LAYER                            │
├─────────────────────────────────────────────────────────────────┤
│ 1. Rate Limiting: Login rate limits applied to admin endpoints  │
│ 2. authHMACMiddleware(false, config) - HMAC SIGNING            │
│    ├── Enforce Required Headers:                               │
│    │   ├── X-Telar-Signature (HMAC signature)                  │
│    │   ├── uid (User ID for context)                           │
│    │   └── X-Timestamp (replay attack prevention)              │
│    ├── Validate Timestamp: ±5 minute window (300 seconds)      │
│    ├── Build Canonical String:                                 │
│    │   └── METHOD\nPATH\nQUERY\nSHA256(BODY)\nUID\nTIMESTAMP  │
│    ├── Validate HMAC-SHA256 against PayloadSecret              │
│    ├── Success: Set user context and continue                  │
│    └── Failure: Return 401 Unauthorized                        │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ AdminHandler.Check(c *fiber.Ctx)                               │
│ ├── Call: h.adminService.CheckAdmin(c.Context())               │
│ ├── Return: c.JSON(fiber.Map{"admin": ok})                     │
│ └── Location: /apps/api/auth/admin/handler.go:17               │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Service.CheckAdmin(ctx context.Context) (bool, error)          │
│ ├── Query: Find user with role="admin"                         │
│ ├── Database Call: s.base.Repository.FindOne()                 │
│ ├── Collection: "userAuth"                                     │
│ ├── Filter: {role: "admin"}                                    │
│ └── Location: /apps/api/auth/admin/service.go:35               │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Repository.FindOne(ctx, "userAuth", {role: "admin"})           │
│ ├── MongoDB Query: db.userAuth.findOne({role: "admin"})        │
│ ├── OR PostgreSQL: SELECT * FROM userAuth WHERE role = 'admin' │
│ ├── Success: Return userAuth document                          │
│ ├── No Result: Return nil (no admin exists)                    │
│ └── Error: Database connection/query error                     │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   RESPONSE FLOW                                │
├─────────────────────────────────────────────────────────────────┤
│ Response Assembly                                               │
│ ├── Success Path: {"admin": true/false}                        │
│ ├── Error Path: Database error → {"admin": false}              │
│ ├── Status Code: 200 OK                                        │
│ ├── Content-Type: application/json                             │
│ └── Return to Client                                            │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Status: 200 OK
├── Headers: {Content-Type: "application/json"}
└── Body: {"admin": true} or {"admin": false}
```

#### Code Trace
1. **Request Reception**: Fiber router receives POST request to `/auth/admin/check`
2. **Middleware Processing**: `authHMACMiddleware` validates HMAC signature using `PayloadSecret`
3. **Handler Execution**: `AdminHandler.Check()` called
4. **Service Call**: `adminService.CheckAdmin()` executes database query
5. **Database Query**: Repository searches `userAuth` collection for `{role: "admin"}`
6. **Response Generation**: JSON response with `admin` boolean field

---

### 2. Admin Signup Route

**Route:** `POST /auth/admin/signup`  
**Middleware:** HMAC Authentication + Rate Limiting  
**Purpose:** Create the first admin user for the system

**Security Update**: Now requires HMAC signing with rate limiting and transaction support

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/admin/signup
    │   ├── Headers: {
    │   │     X-Telar-Signature: "sha256=canonical_hmac_signature",
    │   │     uid: "123e4567-e89b-12d3-a456-426614174000",
    │   │     X-Timestamp: "1642781234",
    │   │     Content-Type: "application/json"
    │   │   }
    │   └── Body: {"email": "admin@example.com", "password": "secretpass"}
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    MIDDLEWARE LAYER                            │
├─────────────────────────────────────────────────────────────────┤
│ 1. Rate Limiting: Login rate limits applied to admin endpoints  │
│ 2. authHMACMiddleware(false, config) - HMAC SIGNING            │
│ ├── Enforce Required Headers: X-Telar-Signature, uid, X-Timestamp │
│ ├── Validate Timestamp: ±5 minute window                       │
│ ├── Build Canonical String: METHOD\nPATH\nQUERY\nSHA256(BODY)\nUID\nTIMESTAMP │
│ ├── Validate HMAC-SHA256 signature                             │
│ └── Continue to handler with validated user context            │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ AdminHandler.Signup(c *fiber.Ctx)                              │
│ ├── Parse form data: c.FormValue("email"), c.FormValue("pass") │
│ ├── Fallback: JSON body parsing for SPA requests               │
│ ├── Call: h.adminService.CreateAdmin(ctx, "admin", email, pwd) │
│ ├── Success: c.Status(201).JSON({"token": token})              │
│ └── Error: errors.HandleServiceError(c, err)                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Service.CreateAdmin(ctx, fullName, email, password)            │
│ ├── 1. Validation: Check email and password required           │
│ ├── 2. Transaction Start: s.base.Repository.WithTransaction()  │
│ ├── 3. Duplicate Check: FindOne({username: email, role: admin})│
│ ├── 4. Password Hash: bcrypt.GenerateFromPassword()            │
│ ├── 5. Create UserAuth: Save to "userAuth" collection          │
│ ├── 6. Create UserProfile: Save to "userProfile" collection    │
│ ├── 7. Transaction Commit                                      │
│ ├── 8. Token Generation: createTelarToken() (after transaction) │
│ └── Return token or error                                       │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Transaction Operations:                                         │
│ ├── 1. Check Existing: FindOne("userAuth", {username, role})   │
│ ├── 2. Insert UserAuth: Save("userAuth", userAuthDoc)          │
│ │   └── Fields: {objectId, username, hashedPassword, role,     │
│ │               emailVerified: true, createdDate, lastUpdated} │
│ ├── 3. Insert UserProfile: Save("userProfile", profileDoc)     │
│ │   └── Fields: {objectId, fullName, socialName, email,        │
│ │               avatar, banner, createdDate, lastUpdated}      │
│ └── 4. Commit or Rollback based on success                     │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   TOKEN GENERATION                             │
├─────────────────────────────────────────────────────────────────┤
│ createTelarToken(profileInfo, claim)                           │
│ ├── Profile Info: {id, login, name, audience}                  │
│ ├── Claim Data: {displayName, socialName, email, uid, role,    │
│ │                createdDate}                                  │
│ ├── JWT Creation: tokens.CreateTokenWithKey()                  │
│ ├── Algorithm: ES256 (private key signing)                     │
│ └── Return: Signed JWT token                                   │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Status: 201 Created
├── Headers: {Content-Type: "application/json"}
└── Body: {"token": "eyJ0eXAiOiJKV1QiLCJhbGc..."}
```

#### Code Trace
1. **Input Processing**: Handler accepts both form data and JSON body
2. **Service Layer**: `CreateAdmin()` uses database transactions for data consistency
3. **Duplicate Prevention**: Checks for existing admin before creation
4. **Password Security**: Uses bcrypt with default cost for password hashing
5. **Data Creation**: Creates both `userAuth` and `userProfile` records atomically
6. **Token Generation**: Creates JWT token with ES256 algorithm for immediate login

---

### 3. Admin Login Route

**Route:** `POST /auth/admin/login`  
**Middleware:** HMAC Authentication + Rate Limiting  
**Purpose:** Authenticate admin users

**Security Update**: Now requires HMAC signing with rate limiting and timestamp validation

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/admin/login
    │   └── Body: {"email": "admin@example.com", "password": "secretpass"}
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ AdminHandler.Login(c *fiber.Ctx)                               │
│ ├── Parse: JSON body parser + form fallback                    │
│ ├── Call: h.adminService.Login(ctx, email, password)           │
│ ├── Success: c.JSON({"token": token})                          │
│ └── Error: errors.HandleServiceError(c, err)                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Service.Login(ctx, email, password)                            │
│ ├── 1. Find Admin: FindOne({username: email, role: "admin"})   │
│ ├── 2. Verify Password: utils.CompareHash(stored, provided)    │
│ ├── 3. Generate Token: createTelarToken(profile, claim)        │
│ └── Return: JWT token string                                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ FindOne("userAuth", {username: email, role: "admin"})          │
│ ├── Query Result: userAuth document with hashed password       │
│ ├── Password Verification: bcrypt.CompareHashAndPassword()     │
│ ├── Success: Generate admin token                              │
│ └── Failure: Return authentication error                       │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Status: 200 OK
├── Body: {"token": "eyJ0eXAiOiJKV1QiLCJhbGc..."}
└── Error Cases:
    ├── 404: Admin not found
    ├── 401: Password mismatch
    └── 500: Database error
```

---

### 4. User Signup Route

**Route:** `POST /auth/signup`  
**Purpose:** Register new users with email/phone verification

**Security Update**: Now returns secure verificationId instead of JWT tokens with plaintext passwords

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/signup
    │   └── Body: {
    │         "fullName": "John Doe",
    │         "email": "john@example.com", 
    │         "newPassword": "strongpass123",
    │         "verifyType": "email",
    │         "g-recaptcha-response": "captcha_token",
    │         "responseType": "spa"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Handler.Handle(c *fiber.Ctx) - GET/POST handling               │
│ ├── GET Request: Return HTML signup form (SSR)                 │
│ └── POST Request: Process signup data                          │
│     ├── Parse: Form values (fullName, email, password, etc.)   │
│     ├── Validation: Required fields check                      │
│     ├── Password Strength: zxcvbn score >= 3, entropy >= 37    │
│     ├── Recaptcha: h.recaptchaVerifier.Verify()               │
│     ├── Generate: UUID for new user                            │
│     └── Create Verification Token                              │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SECURE VERIFICATION FLOW                     │
├─────────────────────────────────────────────────────────────────┤
│ Email Verification (verifyType="email"):                       │
│ ├── h.svc.InitiateEmailVerification() - SECURE METHOD          │
│ ├── Generate: 6-digit verification code                        │
│ ├── Generate: Secure verificationId (UUID)                     │
│ ├── Hash: User password with bcrypt for secure storage         │
│ ├── Save: UserVerification record with expiry (15 min)         │
│ ├── SECURITY: NO JWT tokens - only verificationId returned     │
│ └── Email: Send verification code to user (planned)            │
│                                                                 │
│ Phone Verification (verifyType="phone"):                       │
│ ├── h.svc.InitiatePhoneVerification() - SECURE METHOD          │
│ ├── Generate: 6-digit SMS code                                 │
│ ├── Generate: Secure verificationId (UUID)                     │
│ ├── Save: UserVerification with phone number                   │
│ ├── SECURITY: NO JWT tokens - only verificationId returned     │
│ └── SMS: Send code via provider (planned)                      │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Save("userVerification", verificationDoc)                      │
│ ├── Fields: {objectId: verifyId, userId: newUserId,            │
│ │           code: "123456", target: email, targetType: "email",│
│ │           hashedPassword: hash, expiresAt: now+15min,        │
│ │           used: false, isVerified: false}                    │
│ ├── Purpose: Secure server-side storage of verification data   │
│ └── Security: Password is hashed, not stored in JWT            │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── SPA Response: {
│   "verificationId": "123e4567-e89b-12d3-a456-426614174000",
│   "expiresAt": 1642782134,
│   "message": "Verification code sent to your email"
│ }
├── SSR Response: Same as SPA (no JWT tokens)
└── Error Cases:
    ├── 400: Missing required fields
    ├── 400: Weak password (score < 3)
    ├── 400: Invalid recaptcha
    └── 500: Database/email service error

**SECURITY IMPROVEMENT**: No JWT tokens containing plaintext passwords
```

#### Code Trace
1. **Input Validation**: Checks required fields (fullName, email, password)
2. **Password Strength**: Uses zxcvbn library with score ≥ 3 and entropy ≥ 37
3. **Recaptcha Verification**: Validates Google reCAPTCHA response
4. **Verification Record**: Creates secure verification record in database
5. **JWT Token**: Returns verification token for subsequent verification step

---

### 5. Signup Verification Route

**Route:** `POST /auth/signup/verify`  
**Purpose:** Verify signup code and create user account

**Security Update**: Now uses secure verificationId instead of JWT tokens

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/signup/verify
    │   └── Body: {
    │         "code": "123456",
    │         "verificationId": "123e4567-e89b-12d3-a456-426614174000",
    │         "responseType": "spa"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Handler.Handle(c *fiber.Ctx)                                   │
│ ├── Parse: Body + form fallback for all fields                 │
│ ├── Extract: verificationId and code from request              │
│ ├── SECURITY: No JWT token validation - uses verificationId    │
│ ├── Lookup: Verification record by verificationId              │
│ └── Branch: SPA vs SSR response handling                       │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SECURE VERIFICATION LOOKUP                   │
├─────────────────────────────────────────────────────────────────┤
│ SECURITY IMPROVEMENT: No JWT token validation                  │
│ ├── Direct Lookup: Find verification record by verificationId  │
│ ├── Security: verificationId is opaque UUID, not JWT           │
│ ├── Validation: Record exists, not expired, not used           │
│ ├── Extract: userId, hashedPassword from verification record   │
│ └── NO PLAINTEXT PASSWORDS in verification process             │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ h.svc.FindUserVerification(ctx, {objectId: verificationId})    │
│ ├── SECURITY CHECKS (Enhanced):                                │
│ │   ├── Record exists and not used                             │
│ │   ├── Not expired (expiresAt check)                          │
│ │   ├── IP address matches (optional)                          │
│ │   └── verificationId is valid UUID format                    │
│ ├── Code Verification: h.svc.verifyUserByCode()               │
│ ├── User Creation: h.svc.createUserAuth()                     │
│ ├── Profile Creation: h.svc.createUserProfile()               │
│ ├── Mark Verification Used: {used: true, isVerified: true}     │
│ └── SECURITY: Use stored hashedPassword, not from request      │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE OPERATIONS                          │
├─────────────────────────────────────────────────────────────────┤
│ 1. Verification Lookup:                                        │
│    └── FindOne("userVerification", {objectId: verificationId}) │
│                                                                 │
│ 2. Code Validation:                                            │
│    ├── Compare provided code with stored code                  │
│    ├── Check expiry: now > expiresAt                           │
│    ├── Verify IP address match (optional)                     │
│    ├── SECURITY: Extract hashedPassword from verification      │
│    └── Update: {isVerified: true, used: true}                  │
│                                                                 │
│ 3. User Account Creation:                                      │
│    ├── Save("userAuth", {objectId: userId, username: email,    │
│    │       password: hashedPassword, role: "user",             │
│    │       emailVerified: true, createdDate})                  │
│    └── Save("userProfile", {objectId: userId, fullName,        │
│           socialName, email, avatar, banner, createdDate})     │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   RESPONSE GENERATION                          │
├─────────────────────────────────────────────────────────────────┤
│ SPA Response (responseType="spa"):                             │
│ ├── Return: 200 OK (account created successfully)              │
│ └── Client handles success state                               │
│                                                                 │
│ SSR Response (default):                                        │
│ ├── Generate: JWT session token                                │
│ ├── Claims: {displayName, socialName, email, uid, role}        │
│ └── Return: {"accessToken": token, "tokenType": "Bearer",      │
│             "user": claimData}                                  │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── SPA: 200 OK (no body)
├── SSR: {"accessToken": "jwt_token", "tokenType": "Bearer", 
│         "user": {...}}
└── Error Cases:
    ├── 400: Invalid or expired token
    ├── 400: Wrong verification code
    ├── 400: Already verified/used
    └── 500: Database error
```

#### Code Trace
1. **Token Validation**: Verifies JWT token signature and extracts claims
2. **Security Verification**: Checks verification record exists, not expired, not used
3. **Code Matching**: Compares provided code with stored verification code
4. **Account Creation**: Creates both userAuth and userProfile records
5. **Response Handling**: Different responses for SPA vs SSR clients

---

### 6. User Login Route

**Route:** `POST /auth/login`  
**Purpose:** Authenticate users and issue session tokens

#### Flow Diagram
```
HTTP Request
    │
    ├── POST /auth/login (GET returns login page HTML)
    │   └── Body: {
    │         "username": "john@example.com",
    │         "password": "userpass123",
    │         "responseType": "spa",
    │         "state": "optional_state"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Handler.Handle(c *fiber.Ctx)                                   │
│ ├── Method Check: GET → return HTML page, POST → process login │
│ ├── Content Type: Accept JSON and form-encoded data            │
│ ├── Validation: username and password required                 │
│ ├── Find User: h.svc.FindUserByUsername()                     │
│ ├── Verify User: Email/phone verification check                │
│ ├── Password Check: h.svc.ComparePassword()                   │
│ ├── Profile Lookup: h.svc.ReadProfileAndLanguage()           │
│ └── Token Generation: JWT with user claims                     │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ 1. User Lookup:                                                │
│    └── FindUserByUsername(ctx, username)                       │
│        └── FindOne("userAuth", {username: email})              │
│                                                                 │
│ 2. Verification Check:                                         │
│    ├── foundUser.EmailVerified || foundUser.PhoneVerified      │
│    └── Reject unverified users                                 │
│                                                                 │
│ 3. Password Verification:                                      │
│    └── ComparePassword(stored, provided)                       │
│        └── utils.CompareHash(hashedPassword, plaintext)        │
│                                                                 │
│ 4. Profile Data:                                               │
│    └── ReadProfileAndLanguage(ctx, foundUser)                  │
│        └── FindOne("userProfile", {objectId: user.ObjectId})   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   TOKEN GENERATION                             │
├─────────────────────────────────────────────────────────────────┤
│ tokenutil.CreateTokenWithKey()                                 │
│ ├── Algorithm: ES256 (private key signing)                     │
│ ├── Token Model: {                                             │
│ │   claim: {                                                   │
│ │     displayName: profile.FullName,                           │
│ │     socialName: profile.SocialName,                          │
│ │     email: profile.Email,                                    │
│ │     avatar: profile.Avatar,                                  │
│ │     uid: foundUser.ObjectId,                                 │
│ │     role: foundUser.Role,                                    │
│ │     createdDate: profile.CreatedDate                         │
│ │   }                                                          │
│ │ }                                                            │
│ ├── Profile Info: {id, login, name, audience}                  │
│ └── Return: Signed JWT access token                            │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Success: {
│   "user": {profile_data},
│   "accessToken": "eyJ0eXAiOiJKV1Q...",
│   "tokenType": "Bearer",
│   "expires_in": "0"
│ }
└── Error Cases:
    ├── 404: User not found
    ├── 400: User not verified  
    ├── 401: Password mismatch
    ├── 500: Database/profile error
```

#### Code Trace
1. **User Lookup**: Searches userAuth collection by username (email)
2. **Verification Status**: Ensures user has verified email or phone
3. **Password Verification**: Uses bcrypt to compare hashed passwords
4. **Profile Retrieval**: Gets complete user profile for token claims
5. **JWT Creation**: Generates ES256-signed token with comprehensive user data

---

### 7. Password Change Route

**Route:** `PUT /auth/password/change`  
**Middleware:** JWT Authentication  
**Purpose:** Change user password (authenticated users)

#### Flow Diagram
```
HTTP Request
    │
    ├── PUT /auth/password/change
    │   ├── Headers: {Authorization: "Bearer jwt_token"}
    │   └── Body: {
    │         "currentPassword": "oldpass",
    │         "newPassword": "newpass123",
    │         "confirmPassword": "newpass123"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   MIDDLEWARE LAYER                             │
├─────────────────────────────────────────────────────────────────┤
│ authJWTMiddleware(config)                                       │
│ ├── Extract: Bearer token from Authorization header            │
│ ├── Validate: JWT signature using PublicKey                    │
│ ├── Decode: User claims and set c.Locals("user")              │
│ └── Continue: If token valid, proceed to handler               │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ PasswordHandler.Change(c *fiber.Ctx)                           │
│ ├── Extract User: c.Locals("user").(types.UserContext)         │
│ ├── Parse Input: JSON body + form fallback                     │
│ ├── Validation:                                                │
│ │   ├── currentPassword, newPassword required                  │
│ │   └── newPassword == confirmPassword                         │
│ ├── Verify Current: Load user auth and compare password        │
│ └── Update: h.service.UpdatePasswordByUserId()                │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Current Password Verification:                                 │
│ ├── FindOne("userAuth", {objectId: current.UserID})           │
│ ├── Decode: Extract stored password hash                       │
│ ├── Compare: utils.CompareHash(stored, currentPassword)        │
│ └── Reject: If current password doesn't match                  │
│                                                                 │
│ Password Update:                                               │
│ ├── Hash: utils.Hash(newPassword) with bcrypt                  │
│ ├── Update: Repository.Update("userAuth", filter, data)        │
│ └── Return: Success or error                                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ 1. Current Password Check:                                     │
│    └── FindOne("userAuth", {objectId: userId})                 │
│        └── Returns: {password: bcrypt_hash}                    │
│                                                                 │
│ 2. Password Update:                                            │
│    └── Update("userAuth", {objectId: userId},                  │
│               {$set: {password: newBcryptHash}})               │
│        ├── MongoDB: db.userAuth.updateOne()                    │
│        └── PostgreSQL: UPDATE userAuth SET password = ?        │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Success: 200 OK (no body)
└── Error Cases:
    ├── 401: Invalid JWT token
    ├── 400: Missing required fields
    ├── 400: Password confirmation mismatch
    ├── 401: Current password incorrect
    └── 500: Database error
```

#### Code Trace
1. **JWT Authentication**: Middleware validates token and extracts user context
2. **Input Validation**: Ensures all required fields and password confirmation match
3. **Current Password Verification**: Loads stored hash and compares with provided password
4. **Password Update**: Hashes new password and updates database record
5. **Security**: Uses bcrypt for hashing, validates current password before update

---

### 8. Password Reset Flow

**Routes:** `GET/POST /auth/password/forget` and `GET/POST /auth/password/reset/:verifyId`  
**Purpose:** Reset forgotten passwords via email verification

#### Flow Diagram
```
STEP 1: Forget Password Request
HTTP Request
    │
    ├── POST /auth/password/forget
    │   └── Body: {"email": "user@example.com"}
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   HANDLER LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│ PasswordHandler.ForgetForm(c *fiber.Ctx)                       │
│ ├── Extract: userEmail from form                               │
│ ├── Email Config: h.refEmail, h.refEmailPass, h.smtpEmail      │
│ ├── Create Verification: h.service.PrepareResetVerification()  │
│ ├── Generate Token: tokenutil.GenerateResetPasswordToken()     │
│ ├── Send Email: Reset link with token                          │
│ └── Response: 200 OK                                           │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ PrepareResetVerification(ctx, email, ip):                      │
│ ├── Find User: FindOne("userAuth", {username: email})          │
│ ├── Create Verification Record: {                              │
│ │   objectId: verifyId, userId: user.ObjectId,                 │
│ │   code: "0", target: email, targetType: "email",             │
│ │   remoteIpAddress: ip                                        │
│ │ }                                                            │
│ ├── Save: Repository.Save("userVerification", doc)             │
│ └── Return: verifyId string                                    │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   EMAIL SERVICE                                │
├─────────────────────────────────────────────────────────────────┤
│ Email Delivery:                                                │
│ ├── Link: baseURL + "/auth/password/reset/" + resetToken       │
│ ├── Subject: "Reset Password"                                  │
│ ├── Body: HTML with reset link                                 │
│ ├── SMTP: Using configured email service                       │
│ └── Delivery: To user's email address                          │
└─────────────────────────────────────────────────────────────────┘

STEP 2: Reset Password Form Submission
HTTP Request
    │
    ├── POST /auth/password/reset/TOKEN_HERE
    │   └── Body: {
    │         "newPassword": "newpass123",
    │         "confirmPassword": "newpass123"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   HANDLER LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│ PasswordHandler.ResetForm(c *fiber.Ctx)                        │
│ ├── Extract: verifyToken from URL params                       │
│ ├── Parse: newPassword, confirmPassword from form              │
│ ├── Validate: newPassword == confirmPassword                   │
│ ├── Decode Token: tokenutil.DecodeResetPasswordToken()         │
│ ├── Find User: h.service.FindUserIdByVerifyId()               │
│ ├── Update Password: h.service.UpdatePasswordByUserId()        │
│ └── Response: 200 OK                                           │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Token Processing:                                              │
│ ├── DecodeResetPasswordToken: Extract verifyId from JWT        │
│ ├── FindUserIdByVerifyId: Lookup verification record           │
│ └── UpdatePasswordByUserId: Hash and update password           │
│                                                                 │
│ Database Operations:                                           │
│ ├── FindOne("userVerification", {objectId: verifyId})          │
│ ├── Extract: userId from verification record                   │
│ ├── Hash: utils.Hash(newPassword)                             │
│ └── Update("userAuth", {objectId: userId}, {password: hash})   │
└─────────────────────────────────────────────────────────────────┘

Combined Response Flow:
├── Step 1 Response: 200 OK (email sent)
├── Step 2 Response: 200 OK (password reset)
└── Error Cases:
    ├── 400: Email not found
    ├── 400: Invalid/expired reset token
    ├── 400: Password confirmation mismatch
    └── 500: Email service/database error
```

#### Code Trace
1. **Forget Request**: Creates verification record and sends email with reset link
2. **Email Service**: Uses configured SMTP settings to send reset link
3. **Reset Form**: Validates token, confirms password match, updates database
4. **Security**: Reset tokens are time-limited JWTs, verification records track usage

---

### 9. Profile Update Route

**Route:** `PUT /auth/profile`  
**Middleware:** JWT Authentication  
**Purpose:** Update user profile information

#### Flow Diagram
```
HTTP Request
    │
    ├── PUT /auth/profile
    │   ├── Headers: {Authorization: "Bearer jwt_token"}
    │   └── Body: {
    │         "fullName": "John Smith",
    │         "avatar": "https://example.com/avatar.jpg",
    │         "banner": "https://example.com/banner.jpg",
    │         "tagLine": "Software Developer",
    │         "socialName": "johnsmith"
    │       }
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   MIDDLEWARE LAYER                             │
├─────────────────────────────────────────────────────────────────┤
│ authJWTMiddleware(config)                                       │
│ ├── Validate: JWT token signature                              │
│ ├── Extract: User context from token claims                    │
│ └── Set: c.Locals("user", userContext)                         │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ ProfileHandler.Handle(c *fiber.Ctx)                            │
│ ├── Parse: JSON body to ProfileUpdateModel                     │
│ ├── Validate: Required fields and data types                   │
│ ├── Call: h.service.UpdateProfile(ctx, fields...)              │
│ ├── Success: c.SendStatus(200)                                 │
│ └── Error: Return 500 with error message                       │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SERVICE LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ Service.UpdateProfile(ctx, fullName, avatar, banner,           │
│                      tagLine, socialName)                      │
│ ├── Current Implementation: No-op for compatibility            │
│ ├── Future Implementation:                                     │
│ │   ├── Get User ID from context                               │
│ │   ├── Build update document with provided fields             │
│ │   ├── Repository.Update("userProfile", filter, updates)      │
│ │   └── Return success or error                                │
│ └── Return: nil (success)                                      │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   DATABASE LAYER (Future)                      │
├─────────────────────────────────────────────────────────────────┤
│ Update("userProfile", {objectId: userId}, {                    │
│   $set: {                                                      │
│     fullName: "John Smith",                                    │
│     avatar: "https://example.com/avatar.jpg",                  │
│     banner: "https://example.com/banner.jpg",                  │
│     tagLine: "Software Developer",                             │
│     socialName: "johnsmith",                                   │
│     lastUpdated: currentTimestamp                              │
│   }                                                            │
│ })                                                             │
│ ├── MongoDB: db.userProfile.updateOne()                        │
│ └── PostgreSQL: UPDATE userProfile SET ... WHERE objectId = ?  │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Success: 200 OK (no body)
└── Error Cases:
    ├── 401: Invalid JWT token
    ├── 400: Invalid JSON body
    ├── 500: Database error (when implemented)
```

#### Code Trace
1. **JWT Authentication**: Middleware validates token and extracts user context
2. **Input Parsing**: Handler parses JSON body into ProfileUpdateModel struct
3. **Service Call**: Currently no-op, designed for future profile update implementation
4. **Database Update**: Planned to update userProfile collection with new values

---

### 10. OAuth Flow Routes

**Routes:** `GET /auth/login/github`, `GET /auth/login/google`, `GET /auth/oauth2/authorized`  
**Purpose:** Handle OAuth authentication with external providers

#### Flow Diagram
```
STEP 1: OAuth Initiation
HTTP Request
    │
    ├── GET /auth/login/github (or /google)
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   HANDLER LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│ Login Handler GitHub/Google Methods:                           │
│ ├── Handler.Github(c): Redirect to GitHub OAuth               │
│ │   └── c.Redirect("https://github.com/login/oauth/authorize") │
│ └── Handler.Google(c): Redirect to Google OAuth               │
│     └── c.Redirect("https://accounts.google.com/o/oauth2/...")  │
└─────────────────────────────────────────────────────────────────┘

STEP 2: OAuth Callback
HTTP Request (from OAuth provider)
    │
    ├── GET /auth/oauth2/authorized?code=AUTH_CODE&state=STATE
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   HANDLER LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│ OAuthHandler.Authorized(c *fiber.Ctx)                          │
│ ├── Extract: code, state, provider parameters                  │
│ ├── Validate: state parameter and retrieve PKCE data          │
│ ├── Exchange: Authorization code for access token              │
│ ├── Get User Info: Fetch user profile from OAuth provider     │
│ ├── Find/Create User: Process user account                     │
│ ├── Generate Token: Create JWT session token                   │
│ └── Return: JSON response with access token                    │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   TOKEN GENERATION                             │
├─────────────────────────────────────────────────────────────────┤
│ tokenutil.CreateTokenWithKey():                                │
│ ├── Real OAuth Claims: {                                       │
│ │   displayName: userProfile.FullName,                         │
│ │   socialName: userProfile.SocialName,                        │
│ │   email: userProfile.Email,                                  │
│ │   avatar: userProfile.Avatar,                                │
│ │   uid: userAuth.ObjectId.String(),                           │
│ │   role: userAuth.Role,                                       │
│ │   createdDate: userProfile.CreatedDate,                      │
│ │   provider: provider                                         │
│ │ }                                                            │
│ ├── Profile Info: {id, login, name, audience}                  │
│ └── JWT Token: Signed with private key                         │
└─────────────────────────────────────────────────────────────────┘

Current Implementation Flow:
┌─────────────────────────────────────────────────────────────────┐
│                   FULL OAUTH FLOW (IMPLEMENTED)                │
├─────────────────────────────────────────────────────────────────┤
│ 1. Authorization Code Exchange:                                │
│    ├── Extract: code parameter from callback                   │
│    ├── Exchange: POST to provider token endpoint               │
│    └── Receive: Access token and user info                     │
│                                                                 │
│ 2. User Data Retrieval:                                       │
│    ├── API Call: GET user profile from provider                │
│    ├── Extract: email, name, avatar from provider response     │
│    └── Normalize: Convert to internal user format              │
│                                                                 │
│ 3. User Account Handling:                                     │
│    ├── Lookup: Existing user by email                          │
│    ├── Create: New user if not found                           │
│    ├── Link: OAuth account to existing user                    │
│    └── Update: Profile data from provider                      │
│                                                                 │
│ 4. Session Creation:                                           │
│    ├── Generate: JWT with user claims                          │
│    ├── Store: OAuth tokens for future API calls               │
│    └── Return: Session token to client                         │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Current: {
│   "accessToken": "jwt_token",
│   "tokenType": "Bearer", 
│   "user": {real_oauth_user_claims},
│   "provider": "github|google"
│ }
└── Error Cases:
    ├── 400: Missing authorization code
    ├── 400: Invalid or expired state parameter
    ├── 401: Provider authentication failed
    └── 500: Token generation error
```

#### Code Trace
1. **OAuth Initiation**: Simple redirects to provider authorization URLs
2. **Callback Handling**: Full OAuth flow with PKCE support and real user creation
3. **Token Generation**: Creates JWT with real OAuth user data
4. **User Management**: Finds or creates users based on OAuth provider data

---

### 11. JWKS Endpoint

**Route:** `GET /auth/.well-known/jwks.json`  
**Middleware:** None (Public endpoint)  
**Purpose:** Provide JSON Web Key Set for JWT validation

#### Flow Diagram
```
HTTP Request
    │
    ├── GET /auth/.well-known/jwks.json
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   HANDLER LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│ JWKSHandler.Handle(c *fiber.Ctx)                              │
│ ├── Parse: Public key from configuration                      │
│ ├── Decode: PEM-encoded ECDSA public key                      │
│ ├── Convert: ECDSA key to JWK format                         │
│ ├── Generate: JWKS structure with key metadata                │
│ └── Return: JSON Web Key Set                                 │
└─────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│                   JWK GENERATION                               │
├─────────────────────────────────────────────────────────────────┤
│ JWK Structure: {                                               │
│   "keys": [{                                                   │
│     "kty": "EC",                                               │
│     "use": "sig",                                              │
│     "kid": "key-id",                                           │
│     "alg": "ES256",                                            │
│     "crv": "P-256",                                            │
│     "x": "base64url-encoded-x-coordinate",                     │
│     "y": "base64url-encoded-y-coordinate"                      │
│   }]                                                           │
│ }                                                              │
└─────────────────────────────────────────────────────────────────┘

HTTP Response
├── Status: 200 OK
├── Content-Type: application/json
├── Body: {
│   "keys": [{
│     "kty": "EC",
│     "use": "sig", 
│     "kid": "telar-key-1",
│     "alg": "ES256",
│     "crv": "P-256",
│     "x": "base64url-encoded-x",
│     "y": "base64url-encoded-y"
│   }]
│ }
└── Error Cases:
    ├── 500: Failed to parse public key
    ├── 500: Public key is not ECDSA
    └── 500: Key parsing error
```

#### Code Trace
1. **Key Parsing**: Decodes PEM-encoded public key from configuration
2. **Type Validation**: Ensures the key is ECDSA P-256
3. **JWK Conversion**: Converts ECDSA key to JWK format with base64url encoding
4. **Response Generation**: Returns standardized JWKS JSON structure

---

## 🔧 Database Collections

### userAuth Collection
```json
{
  "objectId": "uuid",
  "username": "user@example.com",
  "password": "bcrypt_hash",
  "role": "user|admin", 
  "emailVerified": true,
  "phoneVerified": false,
  "createdDate": 1634567890,
  "lastUpdated": 1634567890
}
```

### userProfile Collection
```json
{
  "objectId": "uuid",
  "fullName": "John Doe",
  "socialName": "johndoe123",
  "email": "user@example.com",
  "avatar": "https://util.telar.dev/api/avatars/uuid",
  "banner": "https://picsum.photos/id/1/900/300/?blur",
  "tagLine": "Software Developer",
  "createdDate": 1634567890,
  "lastUpdated": 1634567890
}
```

### userVerification Collection (Phase 1 Enhanced)
```json
{
  "objectId": "uuid", // Used as verificationId (NO JWT tokens)
  "userId": "uuid",
  "code": "123456",
  "target": "user@example.com",
  "targetType": "email|phone",
  "counter": 1,
  "createdDate": 1634567890,
  "remoteIpAddress": "192.168.1.1",
  "isVerified": false,
  "lastUpdated": 1634567890,
  "hashedPassword": "bcrypt_hash", // Securely hashed, not in JWT
  "expiresAt": 1634568790, // 15 minute expiry
  "used": false // Prevents replay attacks
}
```

## 🛡️ Security Measures

### Authentication Mechanisms (Phase 1 Refactored)
1. **HMAC Authentication**: Admin routes use HMAC signing with rate limiting
   - Format: `METHOD\nPATH\nQUERY\nSHA256(BODY)\nUID\nTIMESTAMP`
   - Required headers: `X-Telar-Signature`, `uid`, `X-Timestamp`
   - Timestamp validation: ±5 minute window for replay attack prevention
   - Rate limiting: Login rate limits applied to admin endpoints
2. **JWT Authentication**: User routes use ES256 JWT tokens (session tokens only)
3. **Password Hashing**: bcrypt with default cost (10)
4. **Secure Verification**: verificationId system replaces JWT verification tokens
5. **Rate Limiting**: Comprehensive rate limiting across all endpoints
6. **OAuth 2.0**: Full OAuth flow with PKCE support and real user management
7. **JWKS**: Public key distribution for JWT validation

### Validation & Protection
1. **Input Validation**: Required field checks and data type validation
2. **Password Strength**: zxcvbn library with score ≥ 3 and entropy ≥ 37
3. **Recaptcha**: Google reCAPTCHA validation for signup
4. **IP Tracking**: Verification records include IP address for security
5. **Rate Limiting**: 
   - 5 login attempts per 15 minutes per IP
   - 10 signups per hour per IP
   - 10 verification attempts per verification ID
   - 3 password resets per hour per IP

### Database Security (Enhanced)
1. **Transactions**: Admin creation uses database transactions with proper rollback
2. **Secure Storage**: Passwords hashed before database storage using bcrypt
3. **Verification Security**: 
   - Codes stored server-side with secure verificationId lookup
   - NO JWT tokens containing plaintext passwords (ELIMINATED)
   - Verification records expire in 15 minutes
   - Secure reset tokens with high entropy (32 bytes)
4. **Duplicate Prevention**: Checks existing users before creation
5. **HMAC Security**: Canonical signing prevents signature forgery and replay attacks
6. **Rate Limiting**: Database-backed rate limiting for security
7. **Audit Logging**: Comprehensive security event logging

---

## 🚀 Future Enhancements

### Planned Features
1. **Multi-Factor Authentication**: SMS and app-based 2FA
2. **Social Profile Integration**: Link multiple social accounts
3. **Advanced Password Policies**: Configurable complexity requirements
4. **Session Management**: Token refresh and revocation
5. **Advanced Rate Limiting**: Adaptive rate limiting based on user behavior
6. **Biometric Authentication**: Fingerprint and face recognition support

### Architecture Improvements
1. **Event Sourcing**: Track all authentication events
2. **CQRS Pattern**: Separate read/write operations
3. **Circuit Breakers**: Handle external service failures
4. **Distributed Tracing**: Track requests across services
