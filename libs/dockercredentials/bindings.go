package dockercredentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

type Binding struct {
	RegistryHost  string `json:"registry_host"`
	Profile       string `json:"profile"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceHost string `json:"workspace_host"`
}

type bindingFile struct {
	Registries map[string]Binding `json:"registries"`
}

func BindingPath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "docker-credential-databricks.json"), nil
}

func SaveBinding(ctx context.Context, binding Binding) error {
	path, err := BindingPath(ctx)
	if err != nil {
		return err
	}
	host, err := NormalizeServerAddress(binding.RegistryHost)
	if err != nil {
		return err
	}
	binding.RegistryHost = host

	file, err := loadBindingFile(path)
	if err != nil {
		return err
	}
	if file.Registries == nil {
		file.Registries = map[string]Binding{}
	}
	file.Registries[host] = binding
	return writeBindingFile(path, file)
}

func LoadBinding(ctx context.Context, registryHost string) (Binding, bool, error) {
	path, err := BindingPath(ctx)
	if err != nil {
		return Binding{}, false, err
	}
	host, err := NormalizeServerAddress(registryHost)
	if err != nil {
		return Binding{}, false, err
	}
	file, err := loadBindingFile(path)
	if err != nil {
		return Binding{}, false, err
	}
	binding, ok := file.Registries[host]
	return binding, ok, nil
}

func loadBindingFile(path string) (bindingFile, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return bindingFile{}, nil
	}
	if err != nil {
		return bindingFile{}, err
	}
	var file bindingFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return bindingFile{}, fmt.Errorf("parse Docker credential bindings %s: %w", path, err)
	}
	return file, nil
}

func writeBindingFile(path string, file bindingFile) error {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".docker-credential-databricks-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
