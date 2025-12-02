package content

import (
	"fmt"
	"net/url"
	"strings"
)

func GetDomainFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid hostname: %s", u.Hostname())
	}
	domain := parts[len(parts)-2] + "." + parts[len(parts)-1]
	return domain, nil
}
