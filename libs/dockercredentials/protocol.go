package dockercredentials

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type TokenFunc func(context.Context, string) (string, error)

type ProfileFunc func(context.Context, Registry) (string, error)

const maxProtocolInputBytes = 64 * 1024

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
		_, err := readProtocolInput(opts.In)
		if err != nil {
			writeProtocolError(opts.Err, err)
		}
		return err
	case "list":
		return json.NewEncoder(opts.Out).Encode(map[string]string{})
	default:
		err := fmt.Errorf("unsupported Docker credential helper action %q", action)
		writeProtocolError(opts.Err, err)
		return err
	}
}

func handleGet(ctx context.Context, opts ProtocolOptions) error {
	raw, err := readProtocolInput(opts.In)
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	if opts.ResolveProfile == nil {
		err := errors.New("Docker credential profile resolver is not configured")
		writeProtocolError(opts.Err, err)
		return err
	}
	registry, err := ParseRegistryHost(string(raw))
	if err != nil {
		writeProtocolError(opts.Err, err)
		return err
	}
	profileName, err := opts.ResolveProfile(ctx, registry)
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

func readProtocolInput(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxProtocolInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxProtocolInputBytes {
		return nil, errors.New("Docker credential helper input is too large")
	}
	return raw, nil
}

func writeProtocolError(w io.Writer, err error) {
	if w == nil || err == nil {
		return
	}
	_, _ = fmt.Fprintln(w, err)
}
