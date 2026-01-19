# Mutual TLS (mTLS) Support in glab-cli

## Overview

The `glab-cli` project fully supports mutual TLS (mTLS) authentication for connecting to GitLab instances that require client certificate authentication. This document describes the implementation and how to use this feature.

## Configuration

### Config File

To configure mTLS, add the `client_cert` and `client_key` parameters to your host configuration in `~/.config/glab-cli/config.yml`:

```yaml
hosts:
    git.your-domain.com:
        api_protocol: https
        api_host: git.your-domain.com
        token: xxxxxxxxxxxxxxxxxxxxxxxxx
        client_cert: /path/to/client.crt
        client_key: /path/to/client.key
        ca_cert: /path/to/ca-chain.pem  # Optional, for custom CA
```

### Environment Variables

The client certificate and key can also be configured via environment variables:

- `CLIENT_CERT`: Path to the client certificate file
- `CLIENT_KEY`: Path to the client key file

In CI/CD environments with GitLab's autologin feature enabled (`GLAB_ENABLE_CI_AUTOLOGIN=true` and `GITLAB_CI=true`), the following GitLab CI/CD predefined variables are automatically used:

- `CI_SERVER_TLS_CERT_FILE`: Path to the client certificate
- `CI_SERVER_TLS_KEY_FILE`: Path to the client key

## Implementation Details

### Configuration Layer (`internal/config`)

The configuration system handles mTLS parameters through:

1. **Config Mapping** (`config_mapping.go`):
   - Maps `client_cert` and `client_key` to environment variables
   - Supports both standard env vars and GitLab CI variables
   - Falls through ConfigKeyEquivalence to return keys as-is

2. **Config Access** (`config.go`):
   - Retrieves certificate paths from config files or environment
   - Supports host-specific configuration

### API Client Layer (`internal/api`)

The API client implements mTLS through:

1. **Client Structure** (`client.go`):
   - `clientCertFile`: Path to the client certificate file
   - `clientKeyFile`: Path to the client key file

2. **Client Options**:
   - `WithClientCertificate(certFile, keyFile)`: Sets certificate files

3. **TLS Configuration** (`initializeHTTPClient`):
   - Loads certificates using `tls.LoadX509KeyPair()`
   - Configures TLS client config with the certificates
   - Handles errors gracefully if files are invalid

4. **Config Integration** (`NewClientFromConfig`):
   - Reads `client_cert` and `client_key` from config
   - Applies certificates if both are specified
   - Works seamlessly with other TLS options (CA cert, skip verify)

## Usage Examples

### Basic mTLS Configuration

```yaml
hosts:
    secure.gitlab.example.com:
        api_protocol: https
        api_host: secure.gitlab.example.com
        token: your-access-token
        client_cert: /home/user/.certs/client.crt
        client_key: /home/user/.certs/client.key
```

### mTLS with Custom CA

```yaml
hosts:
    secure.gitlab.example.com:
        api_protocol: https
        api_host: secure.gitlab.example.com
        token: your-access-token
        client_cert: /home/user/.certs/client.crt
        client_key: /home/user/.certs/client.key
        ca_cert: /home/user/.certs/ca-chain.pem
```

### Using Environment Variables

```bash
export CLIENT_CERT=/path/to/client.crt
export CLIENT_KEY=/path/to/client.key
glab api projects
```

## Testing

The implementation includes comprehensive tests:

### Unit Tests (`client_cert_test.go`)

- `TestWithClientCertificate`: Validates option function
- `TestClientCertificateLoading`: Verifies TLS config setup
- `TestClientCertificateWithInvalidFiles`: Tests error handling
- `TestClientCertificateOptional`: Ensures client works without certs
- `TestNewClientWithClientCertificate`: Tests full initialization

### Integration Tests (`client_config_integration_test.go`)

- `TestNewClientFromConfigWithClientCertificates`: Config file integration
- `TestNewClientFromConfigWithoutClientCertificates`: Optional certificates
- `TestNewClientFromConfigWithPartialCertificates`: Partial config handling
- `TestClientCertificatesFromEnvironment`: Environment variable support

### Configuration Tests (`config_mapping_test.go`)

- `TestEnvKeyEquivalence`: Validates environment variable mapping
- Tests for both standard and CI/CD autologin scenarios

## Security Considerations

1. **File Permissions**: Client certificate and key files should have restricted permissions (e.g., 0600)
2. **Path Security**: Always use absolute paths for certificate files
3. **Certificate Validation**: The client validates certificates during TLS handshake
4. **Key Protection**: Private keys are loaded into memory securely using Go's crypto/tls package

## Error Handling

The implementation handles various error scenarios:

- **Missing Files**: Returns error if certificate or key files don't exist
- **Invalid Format**: Returns error if files aren't valid PEM format
- **Mismatched Pair**: Returns error if cert and key don't match
- **Partial Configuration**: Only applies certificates if both cert and key are specified

## Related Files

- `internal/config/config_mapping.go`: Environment variable mapping
- `internal/config/config.go`: Config retrieval logic
- `internal/api/client.go`: Client implementation with mTLS
- `internal/api/client_cert_test.go`: Unit tests
- `internal/api/client_config_integration_test.go`: Integration tests
- `internal/config/config_mapping_test.go`: Config tests
- `README.md`: User-facing documentation

## References

- [GitLab Documentation - Mutual TLS](https://docs.gitlab.com/)
- [Go crypto/tls Package](https://pkg.go.dev/crypto/tls)
- [GitLab CI/CD Variables](https://docs.gitlab.com/ee/ci/variables/predefined_variables.html)
