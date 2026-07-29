package dockercredentials

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryHost(t *testing.T) {
	got, err := RegistryHost("123456789", "us-west-2")
	require.NoError(t, err)
	require.Equal(t, "123456789.containers.us-west-2.cloud.databricks.com", got)
}

func TestRegistryHostRejectsEmptyParts(t *testing.T) {
	_, err := RegistryHost("", "us-west-2")
	require.ErrorContains(t, err, "workspace ID is required")

	_, err = RegistryHost("123456789", "")
	require.ErrorContains(t, err, "region is required")
}

func TestParseRegistryHost(t *testing.T) {
	cases := []string{
		"123456789.containers.us-west-2.cloud.databricks.com",
		"https://123456789.containers.us-west-2.cloud.databricks.com",
		"123456789.containers.us-west-2.cloud.databricks.com/v2/",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := ParseRegistryHost(input)
			require.NoError(t, err)
			require.Equal(t, Registry{
				WorkspaceID: "123456789",
				Region:      "us-west-2",
				Host:        "123456789.containers.us-west-2.cloud.databricks.com",
			}, got)
		})
	}
}

func TestParseRegistryHostRejectsNonDARHost(t *testing.T) {
	_, err := ParseRegistryHost("registry.example.com")
	require.ErrorContains(t, err, "is not a Databricks Artifact Registry host")
}
