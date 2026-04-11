package build

import (
	"encoding/json"
	"net/http"
	"time"
)

// Version is the current evolutionary state of Nido.
// This is normally injected at build time, but we provide a default.
var Version = "v4.5.18"

// GetLatestVersion fetches the latest release tag from the GitHub mother nest.
func GetLatestVersion() (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Josepavese/nido/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &http.ProtocolError{ErrorString: "unexpected HTTP status from GitHub"}
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", &http.ProtocolError{ErrorString: "missing tag_name in GitHub response"}
	}
	return release.TagName, nil
}
