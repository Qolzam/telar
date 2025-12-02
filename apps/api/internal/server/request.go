package server

import (
	"bytes"
	"fmt"
	"net/http"
)

// Call makes an HTTP POST request to the specified URL with the given data
func Call(gatewayURL, url string, data []byte) error {
	c := http.Client{}
	reader := bytes.NewBuffer(data)
	fullURL := fmt.Sprintf("%s/%s", gatewayURL, url)
	req, _ := http.NewRequest(http.MethodPost, fullURL, reader)
	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call %s: %w", fullURL, err)
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	return nil
}
