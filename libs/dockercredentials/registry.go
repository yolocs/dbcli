package dockercredentials

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

const (
	HelperName         = "databricks"
	OAuthTokenUsername = "oauthtoken"
)

type Registry struct {
	WorkspaceID string
	Region      string
	Host        string
}

func RegistryHost(workspaceID, region string) (string, error) {
	workspaceID = strings.ToLower(strings.TrimSpace(workspaceID))
	region = strings.ToLower(strings.TrimSpace(region))
	if workspaceID == "" {
		return "", errors.New("workspace ID is required")
	}
	if region == "" {
		return "", errors.New("region is required")
	}
	if strings.Contains(workspaceID, ".") {
		return "", errors.New("workspace ID must not contain dots")
	}
	if strings.Contains(region, ".") {
		return "", errors.New("region must not contain dots")
	}
	if !isDNSLabel(workspaceID) {
		return "", errors.New("workspace ID must be a valid DNS label")
	}
	if !isDNSLabel(region) {
		return "", errors.New("region must be a valid DNS label")
	}
	return fmt.Sprintf("%s.containers.%s.cloud.databricks.com", workspaceID, region), nil
}

func NormalizeServerAddress(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("server address is required")
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("parse server address %q: %w", raw, err)
		}
		if u.User != nil {
			return "", errors.New("server address must not contain user info")
		}
		value = u.Host
	} else if i := strings.IndexByte(value, '/'); i >= 0 {
		value = value[:i]
	}
	if strings.Contains(value, "@") {
		return "", errors.New("server address must not contain user info")
	}
	if strings.Contains(value, ":") {
		host, port, err := net.SplitHostPort(value)
		if err != nil {
			return "", fmt.Errorf("parse server address %q: %w", raw, err)
		}
		if !isNumericPort(port) {
			return "", fmt.Errorf("server address port %q is invalid", port)
		}
		value = host
	}
	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if value == "" {
		return "", errors.New("server address is required")
	}
	return value, nil
}

func ParseRegistryHost(raw string) (Registry, error) {
	host, err := NormalizeServerAddress(raw)
	if err != nil {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host: %w", strings.TrimSpace(raw), err)
	}
	parts := strings.Split(host, ".")
	if len(parts) != 6 || parts[1] != "containers" || parts[3] != "cloud" || parts[4] != "databricks" || parts[5] != "com" {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}
	workspaceID := parts[0]
	region := parts[2]
	if !isDNSLabel(workspaceID) || !isDNSLabel(region) {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}
	return Registry{WorkspaceID: workspaceID, Region: region, Host: host}, nil
}

func isNumericPort(port string) bool {
	if port == "" {
		return false
	}
	for _, ch := range port {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isDNSLabel(label string) bool {
	if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, ch := range label {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return true
}
