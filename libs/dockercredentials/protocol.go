package dockercredentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type TokenFunc func(context.Context, string) (string, error)

type ProfileFunc func(context.Context, string) (string, error)

type ProtocolOptions struct {
	In             io.Reader
	Out            io.Writer
	Err            io.Writer
	ResolveProfile ProfileFunc
	Token          TokenFunc
}

type credentialGetResponse struct {
	Username string `json:"Username"`
	Secret   string `json:"Secret"`
}

func HandleProtocol(ctx context.Context, action string, opts ProtocolOptions) error {
	switch action {
	case "get":
		return handleGet(ctx, opts)
	case "store", "erase":
		_, err := io.Copy(io.Discard, opts.In)
		return err
	case "list":
		return json.NewEncoder(opts.Out).Encode(map[string]string{})
	default:
		return fmt.Errorf("unsupported Docker credential helper action %q", action)
	}
}

func handleGet(ctx context.Context, opts ProtocolOptions) error {
	if opts.ResolveProfile == nil {
		err := errors.New("Docker credential profile resolver is not configured")
		writeProtocolError(opts.Err, err)
		return err
	}
	raw, err := io.ReadAll(opts.In)
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	registry, err := ParseRegistryHost(string(raw))
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	profileName, err := opts.ResolveProfile(ctx, registry.Host)
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	if opts.Token == nil {
		err := errors.New("Docker credential token source is not configured")
		writeProtocolError(opts.Err, err)
		return err
	}
	token, err := opts.Token(ctx, profileName)
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	return json.NewEncoder(opts.Out).Encode(credentialGetResponse{
		Username: OAuthTokenUsername,
		Secret:   token,
	})
}

func writeProtocolError(w io.Writer, err error) {
	if w == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintln(w, err)
}
