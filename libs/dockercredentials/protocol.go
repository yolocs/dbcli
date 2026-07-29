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

// ProtocolOptions contains the streams and callbacks for the Docker credential
// helper protocol. The get action reads a registry host from In, resolves it to
// a profile, and writes Docker JSON to Out. Errors are written to both Out and
// Err so Docker and direct invocations both surface the actionable message.
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

// HandleProtocol handles Docker credential helper actions: get, store, erase,
// and list. Store and erase drain bounded input and intentionally persist
// nothing; list returns an empty JSON object.
func HandleProtocol(ctx context.Context, action string, opts ProtocolOptions) error {
	switch action {
	case "get":
		return handleGet(ctx, opts)
	case "store", "erase":
		_, err := readProtocolInput(opts.In)
		if err != nil {
			writeProtocolError(opts.Out, opts.Err, err)
		}
		return err
	case "list":
		err := json.NewEncoder(opts.Out).Encode(map[string]string{})
		if err != nil {
			writeProtocolError(opts.Out, opts.Err, err)
		}
		return err
	default:
		err := fmt.Errorf("unsupported Docker credential helper action %q", action)
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
}

func handleGet(ctx context.Context, opts ProtocolOptions) error {
	raw, err := readProtocolInput(opts.In)
	if err != nil {
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	if opts.ResolveProfile == nil {
		err := errors.New("Docker credential profile resolver is not configured")
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	registry, err := ParseRegistryHost(string(raw))
	if err != nil {
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	profileName, err := opts.ResolveProfile(ctx, registry)
	if err != nil {
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	if opts.Token == nil {
		err := errors.New("Docker credential token source is not configured")
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	token, err := opts.Token(ctx, profileName)
	if err != nil {
		writeProtocolError(opts.Out, opts.Err, err)
		return err
	}
	err = json.NewEncoder(opts.Out).Encode(credentialGetResponse{
		Username: OAuthTokenUsername,
		Secret:   token,
	})
	if err != nil {
		writeProtocolError(opts.Out, opts.Err, err)
	}
	return err
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

func writeProtocolError(out, errOut io.Writer, err error) {
	if err == nil {
		return
	}
	if out != nil {
		_, _ = fmt.Fprintln(out, err)
	}
	if errOut != nil {
		_, _ = fmt.Fprintln(errOut, err)
	}
}
