package dockercredentials

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadBinding(t *testing.T) {
	ctx := env.WithUserHomeDir(context.Background(), t.TempDir())
	binding := Binding{
		RegistryHost:  "123.containers.us-west-2.cloud.databricks.com",
		Profile:       "dev",
		WorkspaceID:   "123",
		WorkspaceHost: "https://workspace.example.com",
	}

	require.NoError(t, SaveBinding(ctx, binding))
	got, ok, err := LoadBinding(ctx, "https://123.containers.us-west-2.cloud.databricks.com/v2/")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, binding, got)
}

func TestLoadBindingMissingFile(t *testing.T) {
	ctx := env.WithUserHomeDir(context.Background(), t.TempDir())
	_, ok, err := LoadBinding(ctx, "123.containers.us-west-2.cloud.databricks.com")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestBindingPath(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(context.Background(), home)
	got, err := BindingPath(ctx)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".databricks", "docker-credential-databricks.json"), got)
}

func TestSaveBindingPreservesExistingRegistries(t *testing.T) {
	ctx := env.WithUserHomeDir(context.Background(), t.TempDir())
	first := Binding{
		RegistryHost:  "111.containers.us-west-2.cloud.databricks.com",
		Profile:       "first",
		WorkspaceID:   "111",
		WorkspaceHost: "https://first.example.com",
	}
	second := Binding{
		RegistryHost:  "222.containers.us-east-1.cloud.databricks.com",
		Profile:       "second",
		WorkspaceID:   "222",
		WorkspaceHost: "https://second.example.com",
	}

	require.NoError(t, SaveBinding(ctx, first))
	require.NoError(t, SaveBinding(ctx, second))

	gotFirst, ok, err := LoadBinding(ctx, first.RegistryHost)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, first, gotFirst)

	gotSecond, ok, err := LoadBinding(ctx, second.RegistryHost)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, second, gotSecond)
}

func TestLoadBindingInvalidJSON(t *testing.T) {
	ctx := env.WithUserHomeDir(context.Background(), t.TempDir())
	path, err := BindingPath(ctx)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("{not-json"), 0o600))

	_, _, err = LoadBinding(ctx, "123.containers.us-west-2.cloud.databricks.com")
	require.ErrorContains(t, err, "parse Docker credential bindings")
}
