package dockercredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type nonComparableWriter []byte

func (nonComparableWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func TestHandleProtocolGet(t *testing.T) {
	var stdout bytes.Buffer
	err := HandleProtocol(t.Context(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &stdout,
		Err: &bytes.Buffer{},
		ResolveProfile: func(_ context.Context, registry Registry) (string, error) {
			require.Equal(t, "123.containers.us-west-2.cloud.databricks.com", registry.Host)
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
			err := HandleProtocol(t.Context(), action, ProtocolOptions{
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
	err := HandleProtocol(t.Context(), "list", ProtocolOptions{
		In:  &bytes.Buffer{},
		Out: &stdout,
		Err: &bytes.Buffer{},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, stdout.String())
}

func TestHandleProtocolListWriteErrorIsPrinted(t *testing.T) {
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "list", ProtocolOptions{
		In:  &bytes.Buffer{},
		Out: failWriter{},
		Err: &stderr,
	})
	require.ErrorContains(t, err, "write failed")
	require.Contains(t, stderr.String(), "write failed")
}

func TestHandleProtocolGetWriteErrorIsPrinted(t *testing.T) {
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: failWriter{},
		Err: &stderr,
		ResolveProfile: func(context.Context, Registry) (string, error) {
			return "dev", nil
		},
		Token: func(context.Context, string) (string, error) {
			return "access-token", nil
		},
	})
	require.ErrorContains(t, err, "write failed")
	require.Contains(t, stderr.String(), "write failed")
}

func TestHandleProtocolStoreReadErrorIsPrinted(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "store", ProtocolOptions{
		In:  errorReader{},
		Out: &stdout,
		Err: &stderr,
	})
	require.ErrorContains(t, err, "read failed")
	require.Contains(t, stdout.String(), "read failed")
	require.Contains(t, stderr.String(), "read failed")
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

var _ io.Reader = errorReader{}

func TestHandleProtocolUnknownAction(t *testing.T) {
	var stdout bytes.Buffer
	err := HandleProtocol(t.Context(), "bad", ProtocolOptions{
		In:  &bytes.Buffer{},
		Out: &stdout,
		Err: &bytes.Buffer{},
	})
	require.ErrorContains(t, err, `unsupported Docker credential helper action "bad"`)
	require.Contains(t, stdout.String(), `unsupported Docker credential helper action "bad"`)
}

func TestHandleProtocolErrorDoesNotCompareWriters(t *testing.T) {
	w := nonComparableWriter{}
	require.NotPanics(t, func() {
		err := HandleProtocol(t.Context(), "bad", ProtocolOptions{
			In:  &bytes.Buffer{},
			Out: w,
			Err: w,
		})
		require.ErrorContains(t, err, `unsupported Docker credential helper action "bad"`)
	})
}

func TestHandleProtocolGetResolveError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &stdout,
		Err: &stderr,
		ResolveProfile: func(context.Context, Registry) (string, error) {
			return "", errors.New("registry is not configured")
		},
	})
	require.ErrorContains(t, err, "registry is not configured")
	require.Contains(t, stdout.String(), "registry is not configured")
	require.Contains(t, stderr.String(), "registry is not configured")
}

func TestHandleProtocolGetTokenError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "get", ProtocolOptions{
		In:  bytes.NewBufferString("123.containers.us-west-2.cloud.databricks.com"),
		Out: &stdout,
		Err: &stderr,
		ResolveProfile: func(context.Context, Registry) (string, error) {
			return "dev", nil
		},
		Token: func(context.Context, string) (string, error) {
			return "", errors.New("token expired")
		},
	})
	require.ErrorContains(t, err, "token expired")
	require.Contains(t, stdout.String(), "token expired")
	require.Contains(t, stderr.String(), "token expired")
}

func TestHandleProtocolGetRejectsLargeInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := HandleProtocol(t.Context(), "get", ProtocolOptions{
		In:  strings.NewReader(strings.Repeat("a", maxProtocolInputBytes+1)),
		Out: &stdout,
		Err: &stderr,
	})
	require.ErrorContains(t, err, "docker credential helper input is too large")
	require.Contains(t, stdout.String(), "docker credential helper input is too large")
	require.Contains(t, stderr.String(), "docker credential helper input is too large")
}

func TestHandleProtocolStoreRejectsLargeInput(t *testing.T) {
	err := HandleProtocol(t.Context(), "store", ProtocolOptions{
		In:  strings.NewReader(strings.Repeat("a", maxProtocolInputBytes+1)),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	})
	require.ErrorContains(t, err, "docker credential helper input is too large")
}
