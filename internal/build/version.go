package build

import (
	"encoding/json"
	"net/http"
	"time"
)

// Version is the current evolutionary state of Nido.
// This is normally injected at build time, but we provide a default.
var Version = "v4.4.3"

// GetLatestVersion fetches the latest release tag from the GitHub mother nest.
func GetLatestVersion() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Josepavese/nido/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}
