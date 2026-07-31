package dockercredentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func readDockerConfigForTest(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	return got
}

func TestConfigureDockerCredentialHelperCreatesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker", "config.json")

	require.NoError(t, SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com"))

	got := readDockerConfigForTest(t, path)
	require.Equal(t, map[string]any{
		"123.containers.us-west-2.cloud.databricks.com": HelperName,
	}, got["credHelpers"])

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestConfigureDockerCredentialHelperPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "auths": {
    "registry.example.com": {"auth": "abc"}
  },
  "credsStore": "desktop",
  "credHelpers": {
    "registry.example.com": "desktop"
  },
  "experimental": "enabled"
}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com"))

	got := readDockerConfigForTest(t, path)
	require.Equal(t, "desktop", got["credsStore"])
	require.Equal(t, "enabled", got["experimental"])
	require.Equal(t, map[string]any{
		"registry.example.com":                          "desktop",
		"123.containers.us-west-2.cloud.databricks.com": HelperName,
	}, got["credHelpers"])
	require.Contains(t, got, "auths")
}

func TestConfigureDockerCredentialHelperIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "credHelpers": {
    "123.containers.us-west-2.cloud.databricks.com": "databricks"
  }
}`), 0o600))

	before, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com"))

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestConfigureDockerCredentialHelperReplacesExistingHelper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "credHelpers": {
    "123.containers.us-west-2.cloud.databricks.com": "desktop"
  }
}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com"))

	got := readDockerConfigForTest(t, path)
	require.Equal(t, map[string]any{
		"123.containers.us-west-2.cloud.databricks.com": HelperName,
	}, got["credHelpers"])
}

func TestConfigureDockerCredentialHelperTreatsNullCredHelpersAsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"credHelpers": null}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com"))

	got := readDockerConfigForTest(t, path)
	require.Equal(t, map[string]any{
		"123.containers.us-west-2.cloud.databricks.com": HelperName,
	}, got["credHelpers"])
}

func TestConfigureDockerCredentialHelperRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0o600))

	err := SetCredentialHelper(path, "123.containers.us-west-2.cloud.databricks.com")
	require.ErrorContains(t, err, "read Docker config")
}
