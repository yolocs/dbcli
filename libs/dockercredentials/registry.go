package dockercredentials

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
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
	workspaceID = strings.TrimSpace(workspaceID)
	region = strings.TrimSpace(region)
	if workspaceID == "" {
		return "", errors.New("workspace ID is required")
	}
	if region == "" {
		return "", errors.New("region is required")
	}
	if !isDNSLabel(workspaceID) {
		return "", fmt.Errorf("invalid workspace ID %q", workspaceID)
	}
	if !isDNSLabel(region) {
		return "", fmt.Errorf("invalid region %q", region)
	}
	return fmt.Sprintf("%s.containers.%s.cloud.databricks.com", workspaceID, region), nil
}

func normalizeServerAddress(raw string) (string, error) {
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

	if host, port, ok, err := splitOptionalPort(value); err != nil {
		return "", err
	} else if ok {
		value = host
		if err := validatePort(port); err != nil {
			return "", err
		}
	}

	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if value == "" {
		return "", errors.New("server address is required")
	}
	return value, nil
}

func ParseRegistryHost(raw string) (Registry, error) {
	host, err := normalizeServerAddress(raw)
	if err != nil {
		return Registry{}, err
	}

	const suffix = ".cloud.databricks.com"
	if !strings.HasSuffix(host, suffix) {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}

	trimmed := strings.TrimSuffix(host, suffix)
	workspaceID, region, ok := strings.Cut(trimmed, ".containers.")
	if !ok || !isDNSLabel(workspaceID) || !isDNSLabel(region) {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}

	return Registry{
		WorkspaceID: workspaceID,
		Region:      region,
		Host:        host,
	}, nil
}

func splitOptionalPort(value string) (host, port string, ok bool, err error) {
	host, port, err = net.SplitHostPort(value)
	if err == nil {
		return host, port, true, nil
	}

	if strings.Count(value, ":") == 1 {
		host, port, found := strings.Cut(value, ":")
		if found && port != "" {
			return host, port, true, nil
		}
	}

	return "", "", false, nil
}

func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid registry port %q", port)
	}
	return nil
}

func isDNSLabel(label string) bool {
	if label == "" || len(label) > 63 {
		return false
	}
	for i, r := range label {
		if r > unicode.MaxASCII {
			return false
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if r == '-' && i > 0 && i < len(label)-1 {
			continue
		}
		return false
	}
	return true
}
