package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func Call[Req any, Res any](method, url, token string, payload Req) (Res, error) {
	var result Res

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return result, fmt.Errorf("error serializing payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("unexpected error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return result, fmt.Errorf("erro deserializing response: %w", err)
	}

	return result, nil
}