package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
	}{
		{
			name: "federation enabled with domain set is valid",
			cfg: ServerConfig{
				Federation: FederationConfig{Enabled: true, Domain: "example.com"},
			},
		},
		{
			name: "federation disabled with empty domain is valid",
			cfg: ServerConfig{
				Federation: FederationConfig{Enabled: false, Domain: ""},
			},
		},
		{
			name: "federation disabled with domain set is valid",
			cfg: ServerConfig{
				Federation: FederationConfig{Enabled: false, Domain: "example.com"},
			},
		},
		{
			name:    "federation enabled with empty domain is invalid",
			cfg:     ServerConfig{Federation: FederationConfig{Enabled: true, Domain: ""}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "server.federation.domain")
				return
			}
			require.NoError(t, err)
		})
	}
}
