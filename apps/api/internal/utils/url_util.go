package utils

import (
	"fmt"
)

// GetPrettyURL returns the base route (deprecated - use GetPrettyURLf with baseRoute parameter)
func GetPrettyURL() string {
	return "/api/v1" // Default base route
}

// GetPrettyURLf formats according to pretty URL from (baseRoute+url) and returns the resulting string.
func GetPrettyURLf(url string) string {
	return fmt.Sprintf("%s%s", "/api/v1", url)
}

// GetPrettyURLWithBase formats according to pretty URL from (baseRoute+url) and returns the resulting string.
func GetPrettyURLWithBase(baseRoute, url string) string {
	return fmt.Sprintf("%s%s", baseRoute, url)
}
