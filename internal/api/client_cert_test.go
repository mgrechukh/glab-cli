//go:build !integration

package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// generateTestCert generates a self-signed certificate for testing
func generateTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "Test Client",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Write certificate to temp file
	certFile = t.TempDir() + "/test-client.crt"
	certOut, err := os.Create(certFile)
	require.NoError(t, err)
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	require.NoError(t, err)
	certOut.Close()

	// Write private key to temp file
	keyFile = t.TempDir() + "/test-client.key"
	keyOut, err := os.Create(keyFile)
	require.NoError(t, err)
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	require.NoError(t, err)
	keyOut.Close()

	return certFile, keyFile
}

func TestWithClientCertificate(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	// Test that WithClientCertificate option sets the fields correctly
	client := &Client{}
	option := WithClientCertificate(certFile, keyFile)
	err := option(client)
	require.NoError(t, err)

	require.Equal(t, certFile, client.clientCertFile)
	require.Equal(t, keyFile, client.clientKeyFile)
}

func TestClientCertificateLoading(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	// Create a client with client certificate
	client := &Client{
		baseURL:        "https://example.com",
		clientCertFile: certFile,
		clientKeyFile:  keyFile,
	}

	// Initialize the HTTP client
	err := client.initializeHTTPClient()
	require.NoError(t, err)
	require.NotNil(t, client.httpClient)

	// Verify that the TLS config has the client certificate loaded
	transport, ok := client.httpClient.Transport.(*debugTransport)
	var tlsConfig *tls.Config
	if ok {
		// If debug transport is wrapped, get the inner transport
		innerTransport, ok := transport.rt.(*http.Transport)
		require.True(t, ok, "expected http.Transport")
		tlsConfig = innerTransport.TLSClientConfig
	} else {
		// Direct transport
		httpTransport, ok := client.httpClient.Transport.(*http.Transport)
		require.True(t, ok, "expected http.Transport")
		tlsConfig = httpTransport.TLSClientConfig
	}

	require.NotNil(t, tlsConfig)
	require.Len(t, tlsConfig.Certificates, 1, "should have one client certificate loaded")
}

func TestClientCertificateWithInvalidFiles(t *testing.T) {
	tests := []struct {
		name     string
		certFile string
		keyFile  string
	}{
		{
			name:     "non-existent cert file",
			certFile: "/nonexistent/cert.pem",
			keyFile:  "/tmp/key.pem",
		},
		{
			name:     "non-existent key file",
			certFile: "/tmp/cert.pem",
			keyFile:  "/nonexistent/key.pem",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				baseURL:        "https://example.com",
				clientCertFile: tc.certFile,
				clientKeyFile:  tc.keyFile,
			}

			err := client.initializeHTTPClient()
			require.Error(t, err, "should fail with invalid certificate files")
		})
	}
}

func TestClientCertificateOptional(t *testing.T) {
	// Test that client works fine without client certificates
	client := &Client{
		baseURL: "https://example.com",
	}

	err := client.initializeHTTPClient()
	require.NoError(t, err)
	require.NotNil(t, client.httpClient)
}

func TestNewClientWithClientCertificate(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	// Test creating a new client with client certificate option
	client, err := NewClient(
		func(c *http.Client) (gitlab.AuthSource, error) {
			return gitlab.AccessTokenAuthSource{Token: "test-token"}, nil
		},
		WithBaseURL("https://example.com"),
		WithClientCertificate(certFile, keyFile),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, certFile, client.clientCertFile)
	require.Equal(t, keyFile, client.clientKeyFile)
}
