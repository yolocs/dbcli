package dockercredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleProtocolGet(t *testing.T) {
	var stdout bytes.Buffer
	err := HandleProtocol(context.Background(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &stdout,
		Err: &bytes.Buffer{},
		ResolveProfile: func(context.Context, string) (string, error) {
			return "dev", nil
		},
		Token: func(_ context.Context, profileName string) (string, error) {
			require.Equal(t, "dev", profileName)
			return "access-token", nil
		},
	})
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, "oauthtoken", got["Username"])
	require.Equal(t, "access-token", got["Secret"])
}

func TestHandleProtocolStoreEraseNoop(t *testing.T) {
	for _, action := range []string{"store", "erase"} {
		t.Run(action, func(t *testing.T) {
			var stdout bytes.Buffer
			err := HandleProtocol(context.Background(), action, ProtocolOptions{
				In:  bytes.NewBufferString(`{"ServerURL":"123.containers.us-west-2.cloud.databricks.com"}`),
				Out: &stdout,
				Err: &bytes.Buffer{},
			})
			require.NoError(t, err)
			require.Empty(t, stdout.String())
		})
	}
}

func TestHandleProtocolList(t *testing.T) {
	var stdout bytes.Buffer
	err := HandleProtocol(context.Background(), "list", ProtocolOptions{
		In:  &bytes.Buffer{},
		Out: &stdout,
		Err: &bytes.Buffer{},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, stdout.String())
}

func TestHandleProtocolUnknownAction(t *testing.T) {
	err := HandleProtocol(context.Background(), "bad", ProtocolOptions{
		In:  &bytes.Buffer{},
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	})
	require.ErrorContains(t, err, `unsupported Docker credential helper action "bad"`)
}

func TestHandleProtocolGetResolveError(t *testing.T) {
	var stderr bytes.Buffer
	err := HandleProtocol(context.Background(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &bytes.Buffer{},
		Err: &stderr,
		ResolveProfile: func(context.Context, string) (string, error) {
			return "", errors.New("registry is not configured")
		},
	})
	require.ErrorContains(t, err, "registry is not configured")
	require.Contains(t, stderr.String(), "registry is not configured")
}

func TestHandleProtocolGetTokenError(t *testing.T) {
	var stderr bytes.Buffer
	err := HandleProtocol(context.Background(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &bytes.Buffer{},
		Err: &stderr,
		ResolveProfile: func(context.Context, string) (string, error) {
			return "dev", nil
		},
		Token: func(context.Context, string) (string, error) {
			return "", errors.New("token expired")
		},
	})
	require.ErrorContains(t, err, "token expired")
	require.Contains(t, stderr.String(), "token expired")
}
