package dockercredentials

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

const HelperName = "databricks"
const OAuthTokenUsername = "oauthtoken"

type Registry struct {
	WorkspaceID string
	Region      string
	Host        string
}

func RegistryHost(workspaceID, region string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	region = strings.TrimSpace(region)
	if workspaceID == "" {
		return "", errors.New("workspace ID is required")
	}
	if region == "" {
		return "", errors.New("region is required")
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
		value = u.Host
	} else if i := strings.IndexByte(value, '/'); i >= 0 {
		value = value[:i]
	}
	host, port, err := net.SplitHostPort(value)
	if err == nil && port != "" {
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
		return Registry{}, err
	}
	const suffix = ".cloud.databricks.com"
	if !strings.HasSuffix(host, suffix) {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}
	trimmed := strings.TrimSuffix(host, suffix)
	workspaceID, region, ok := strings.Cut(trimmed, ".containers.")
	if !ok || workspaceID == "" || region == "" {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}
	return Registry{WorkspaceID: workspaceID, Region: region, Host: host}, nil
}
