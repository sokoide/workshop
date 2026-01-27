package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/sokoide/workshop/secret_management/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient is a test double for HTTPClient.
type mockHTTPClient struct {
	statusCode    int
	responseBody  string
	err           error
	calledWithURL string
	calledWithKey string
}

func (m *mockHTTPClient) Get(url, apiKey string) (int, string, error) {
	m.calledWithURL = url
	m.calledWithKey = apiKey
	return m.statusCode, m.responseBody, m.err
}

// mockSecretManager is a test double for SecretManager.
type mockSecretManager struct {
	secret *domain.Secret
	err    error
}

func (m *mockSecretManager) RetrieveSecret(ctx context.Context, key string) (*domain.Secret, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.secret, nil
}

func (m *mockSecretManager) StoreSecret(ctx context.Context, key, value string) error {
	return nil
}

func (m *mockSecretManager) ListAllSecrets(ctx context.Context) ([]domain.SecretMetadata, error) {
	return nil, nil
}

func TestAPIClient_CallAPI(t *testing.T) {
	tests := []struct {
		name        string
		secretKey   string
		apiEndpoint string
		setupSecret func(*mockSecretManager)
		setupHTTP   func(*mockHTTPClient)
		wantErr     bool
		errContains string
	}{
		{
			name:        "success",
			secretKey:   "api-key",
			apiEndpoint: "https://api.example.com/data",
			setupSecret: func(m *mockSecretManager) {
				m.secret = &domain.Secret{
					Key:     "api-key",
					Value:   "sk-test-12345",
					Version: 1,
				}
			},
			setupHTTP: func(h *mockHTTPClient) {
				h.statusCode = 200
				h.responseBody = `{"status": "ok"}`
			},
			wantErr: false,
		},
		{
			name:        "secret retrieval fails",
			secretKey:   "api-key",
			apiEndpoint: "https://api.example.com/data",
			setupSecret: func(m *mockSecretManager) {
				m.err = errors.New("vault unavailable")
			},
			setupHTTP:   func(h *mockHTTPClient) {},
			wantErr:     true,
			errContains: "failed to retrieve API key",
		},
		{
			name:        "http call fails",
			secretKey:   "api-key",
			apiEndpoint: "https://api.example.com/data",
			setupSecret: func(m *mockSecretManager) {
				m.secret = &domain.Secret{
					Key:     "api-key",
					Value:   "sk-test-12345",
					Version: 1,
				}
			},
			setupHTTP: func(h *mockHTTPClient) {
				h.err = errors.New("network error")
			},
			wantErr:     true,
			errContains: "API call failed",
		},
		{
			name:        "api returns error status",
			secretKey:   "api-key",
			apiEndpoint: "https://api.example.com/data",
			setupSecret: func(m *mockSecretManager) {
				m.secret = &domain.Secret{
					Key:     "api-key",
					Value:   "sk-test-12345",
					Version: 1,
				}
			},
			setupHTTP: func(h *mockHTTPClient) {
				h.statusCode = 401
				h.responseBody = "Unauthorized"
			},
			wantErr: false, // APIClient doesn't treat 4xx/5xx as errors, it logs them
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := &mockSecretManager{}
			tt.setupSecret(mockSM)

			mockHTTP := &mockHTTPClient{}
			tt.setupHTTP(mockHTTP)

			client := &apiClient{
				secretManager: mockSM,
				httpClient:    mockHTTP,
			}
			ctx := context.Background()

			err := client.CallAPI(ctx, tt.secretKey, tt.apiEndpoint)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify HTTP client was called with the API key from the secret
				if mockSM.secret != nil {
					assert.Equal(t, tt.apiEndpoint, mockHTTP.calledWithURL)
					assert.Equal(t, mockSM.secret.Value, mockHTTP.calledWithKey)
				}
			}
		})
	}
}
