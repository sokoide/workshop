package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/secret_management/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretRepository is a test double for domain.SecretRepository.
type mockSecretRepository struct {
	secrets map[string]domain.Secret
	getErr  error
	putErr  error
	listErr error
}

func (m *mockSecretRepository) GetSecret(ctx context.Context, key string) (*domain.Secret, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if secret, ok := m.secrets[key]; ok {
		return &secret, nil
	}
	return nil, errors.New("secret not found")
}

func (m *mockSecretRepository) PutSecret(ctx context.Context, key, value string) error {
	if m.putErr != nil {
		return m.putErr
	}
	m.secrets[key] = domain.Secret{
		Key:       key,
		Value:     value,
		Version:   1,
		CreatedAt: time.Now(),
	}
	return nil
}

func (m *mockSecretRepository) ListSecrets(ctx context.Context) ([]domain.SecretMetadata, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []domain.SecretMetadata
	for _, s := range m.secrets {
		result = append(result, domain.SecretMetadata{
			Key:       s.Key,
			Version:   s.Version,
			CreatedAt: s.CreatedAt,
		})
	}
	return result, nil
}

func TestSecretManager_RetrieveSecret(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setupMock   func(*mockSecretRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			key:  "test-key",
			setupMock: func(m *mockSecretRepository) {
				m.secrets = map[string]domain.Secret{
					"test-key": {
						Key:       "test-key",
						Value:     "test-value",
						Version:   1,
						CreatedAt: time.Now(),
					},
				}
			},
			wantErr: false,
		},
		{
			name:        "empty key",
			key:         "",
			setupMock:   func(m *mockSecretRepository) {},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "secret not found",
			key:  "nonexistent",
			setupMock: func(m *mockSecretRepository) {
				m.secrets = make(map[string]domain.Secret)
			},
			wantErr:     true,
			errContains: "failed to retrieve",
		},
		{
			name: "repository error",
			key:  "test-key",
			setupMock: func(m *mockSecretRepository) {
				m.getErr = errors.New("repository error")
			},
			wantErr:     true,
			errContains: "failed to retrieve",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSecretRepository{}
			tt.setupMock(mock)

			sm := NewSecretManager(mock)
			ctx := context.Background()

			secret, err := sm.RetrieveSecret(ctx, tt.key)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, secret)
			} else {
				require.NoError(t, err)
				require.NotNil(t, secret)
				assert.Equal(t, tt.key, secret.Key)
			}
		})
	}
}

func TestSecretManager_StoreSecret(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		setupMock   func(*mockSecretRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:  "success",
			key:   "new-key",
			value: "new-value",
			setupMock: func(m *mockSecretRepository) {
				m.secrets = make(map[string]domain.Secret)
			},
			wantErr: false,
		},
		{
			name:        "empty key",
			key:         "",
			value:       "value",
			setupMock:   func(m *mockSecretRepository) {},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "empty value",
			key:         "key",
			value:       "",
			setupMock:   func(m *mockSecretRepository) {},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:  "repository error",
			key:   "key",
			value: "value",
			setupMock: func(m *mockSecretRepository) {
				m.putErr = errors.New("repository error")
			},
			wantErr:     true,
			errContains: "failed to store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSecretRepository{}
			tt.setupMock(mock)

			sm := NewSecretManager(mock)
			ctx := context.Background()

			err := sm.StoreSecret(ctx, tt.key, tt.value)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify secret was stored
				secret, err := mock.GetSecret(ctx, tt.key)
				require.NoError(t, err)
				assert.Equal(t, tt.key, secret.Key)
				assert.Equal(t, tt.value, secret.Value)
			}
		})
	}
}

func TestSecretManager_ListAllSecrets(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockSecretRepository)
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name: "success with secrets",
			setupMock: func(m *mockSecretRepository) {
				m.secrets = map[string]domain.Secret{
					"key1": {Key: "key1", Value: "val1", Version: 1, CreatedAt: time.Now()},
					"key2": {Key: "key2", Value: "val2", Version: 2, CreatedAt: time.Now()},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "success empty",
			setupMock: func(m *mockSecretRepository) {
				m.secrets = make(map[string]domain.Secret)
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "repository error",
			setupMock: func(m *mockSecretRepository) {
				m.listErr = errors.New("repository error")
			},
			wantErr:     true,
			errContains: "failed to list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSecretRepository{}
			tt.setupMock(mock)

			sm := NewSecretManager(mock)
			ctx := context.Background()

			secrets, err := sm.ListAllSecrets(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, secrets, tt.wantCount)
			}
		})
	}
}

func TestSecretManager_NilRepository(t *testing.T) {
	assert.Panics(t, func() {
		NewSecretManager(nil)
	}, "Expected panic when repository is nil")
}
