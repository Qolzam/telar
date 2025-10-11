package types

// HTTP Header Constants
const (
	HeaderHMACAuthenticate = "X-Telar-Signature"
	HeaderTimestamp        = "X-Timestamp"
	HeaderUID              = "uid"
	HeaderAuthorization    = "Authorization"
	HeaderContentType      = "Content-Type"
)

// Authentication Constants
const (
	BearerPrefix = "Bearer "
	HMACPrefix   = "sha256="
)

// Common Values
const (
	UserRole  = "user"
	AdminRole = "admin"
)
