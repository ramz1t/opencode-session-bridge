package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	BaseURL  string
	Username string
	Password string
}

var urlPattern = regexp.MustCompile(`https?://\S+`)

// Discover finds the running OpenCode service. Environment variables take
// precedence over what the local service reports.
func Discover(ctx context.Context) (Config, error) {
	baseURL := os.Getenv("OPENCODE_BASE_URL")
	if baseURL == "" {
		url, err := serviceURL(ctx)
		if err != nil {
			return Config{}, err
		}
		baseURL = url
	}

	password := os.Getenv("OPENCODE_SERVER_PASSWORD")
	if password == "" {
		p, err := servicePassword()
		if err != nil {
			return Config{}, err
		}
		password = p
	}

	username := os.Getenv("OPENCODE_SERVER_USERNAME")
	if username == "" {
		username = "opencode"
	}

	return Config{BaseURL: baseURL, Username: username, Password: password}, nil
}

func serviceURL(ctx context.Context) (string, error) {
	for _, bin := range binaryCandidates() {
		out, err := exec.CommandContext(ctx, bin, "service", "status").Output()
		if err != nil {
			continue
		}
		if url := urlPattern.FindString(string(out)); url != "" {
			return strings.TrimRight(url, "/"), nil
		}
	}
	return "", errors.New("no running OpenCode service found: start OpenCode or set OPENCODE_BASE_URL")
}

func binaryCandidates() []string {
	candidates := []string{"opencode"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".opencode", "bin", "opencode"))
	}
	return candidates
}

func servicePassword() (string, error) {
	path := filepath.Join(configDir(), "opencode", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("cannot read " + path + ": set OPENCODE_SERVER_PASSWORD or start OpenCode")
	}
	var service struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(data, &service); err != nil {
		return "", err
	}
	if service.Password == "" {
		return "", errors.New(path + " has no password: set OPENCODE_SERVER_PASSWORD")
	}
	return service.Password, nil
}

func configDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
