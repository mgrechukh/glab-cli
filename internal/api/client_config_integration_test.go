//go:build !integration

package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/config"
)

func TestNewClientFromConfigWithClientCertificates(t *testing.T) {
	// Generate test certificates
	certFile, keyFile := generateTestCert(t)

	// Create a test config with client certificates
	configStr := `
hosts:
  example.com:
    api_protocol: https
    api_host: example.com
    token: test-token
    client_cert: ` + certFile + `
    client_key: ` + keyFile + `
`

	cfg := config.NewFromString(configStr)

	// Create client from config
	client, err := NewClientFromConfig("example.com", cfg, false, "test-agent")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify that client certificate files are set
	require.Equal(t, certFile, client.clientCertFile)
	require.Equal(t, keyFile, client.clientKeyFile)

	// Verify that HTTP client is initialized with certificates
	require.NotNil(t, client.httpClient)
}

func TestNewClientFromConfigWithoutClientCertificates(t *testing.T) {
	// Create a test config without client certificates
	configStr := `
hosts:
  example.com:
    api_protocol: https
    api_host: example.com
    token: test-token
`

	cfg := config.NewFromString(configStr)

	// Create client from config
	client, err := NewClientFromConfig("example.com", cfg, false, "test-agent")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify that client certificate files are not set
	require.Empty(t, client.clientCertFile)
	require.Empty(t, client.clientKeyFile)

	// Verify that HTTP client is still initialized
	require.NotNil(t, client.httpClient)
}

func TestNewClientFromConfigWithPartialCertificates(t *testing.T) {
	tests := []struct {
		name      string
		configStr func(certFile, keyFile string) string
	}{
		{
			name: "only client_cert specified",
			configStr: func(certFile, keyFile string) string {
				return `
hosts:
  example.com:
    api_protocol: https
    api_host: example.com
    token: test-token
    client_cert: ` + certFile + `
`
			},
		},
		{
			name: "only client_key specified",
			configStr: func(certFile, keyFile string) string {
				return `
hosts:
  example.com:
    api_protocol: https
    api_host: example.com
    token: test-token
    client_key: ` + keyFile + `
`
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			certFile, keyFile := generateTestCert(t)
			cfg := config.NewFromString(tc.configStr(certFile, keyFile))

			// Create client from config - should succeed but not load certificates
			client, err := NewClientFromConfig("example.com", cfg, false, "test-agent")
			require.NoError(t, err)
			require.NotNil(t, client)
			require.NotNil(t, client.httpClient)
		})
	}
}

func TestClientCertificatesFromEnvironment(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	// Set environment variables
	t.Setenv("CLIENT_CERT", certFile)
	t.Setenv("CLIENT_KEY", keyFile)

	// Create a minimal config
	configStr := `
hosts:
  example.com:
    api_protocol: https
    api_host: example.com
    token: test-token
`

	cfg := config.NewFromString(configStr)

	// Create client from config - should pick up env vars
	client, err := NewClientFromConfig("example.com", cfg, false, "test-agent")
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify that client certificate files are set from environment
	require.Equal(t, certFile, client.clientCertFile)
	require.Equal(t, keyFile, client.clientKeyFile)
}
