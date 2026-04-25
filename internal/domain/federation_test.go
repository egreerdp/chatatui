package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFederatedIdentity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FederatedIdentity
		wantErr bool
	}{
		{
			name:  "valid identity",
			input: "alice@server-a.com",
			want:  FederatedIdentity{Username: "alice", Domain: "server-a.com"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "missing @",
			input:   "aliceserver-a.com",
			wantErr: true,
		},
		{
			name:    "leading @ — empty username",
			input:   "@server-a.com",
			wantErr: true,
		},
		{
			name:    "trailing @ — empty domain",
			input:   "alice@",
			wantErr: true,
		},
		{
			name:    "multiple @ signs",
			input:   "alice@server@a.com",
			wantErr: true,
		},
		{
			name:  "leading and trailing whitespace trimmed",
			input: "  alice@server-a.com  ",
			want:  FederatedIdentity{Username: "alice", Domain: "server-a.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFederatedIdentity(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFederatedIdentity_String(t *testing.T) {
	f := FederatedIdentity{Username: "alice", Domain: "server-a.com"}
	assert.Equal(t, "alice@server-a.com", f.String())
}

func TestFederatedIdentity_RoundTrip(t *testing.T) {
	inputs := []string{
		"alice@server-a.com",
		"bob@localhost:8080",
		"user123@my.server.example",
	}
	for _, s := range inputs {
		t.Run(s, func(t *testing.T) {
			f, err := ParseFederatedIdentity(s)
			require.NoError(t, err)
			assert.Equal(t, s, f.String())
		})
	}
}

func TestFederatedIdentity_IsLocal(t *testing.T) {
	tests := []struct {
		name        string
		identity    FederatedIdentity
		localDomain string
		want        bool
	}{
		{
			name:        "same domain is local",
			identity:    FederatedIdentity{Username: "alice", Domain: "server-a.com"},
			localDomain: "server-a.com",
			want:        true,
		},
		{
			name:        "different domain is not local",
			identity:    FederatedIdentity{Username: "alice", Domain: "server-a.com"},
			localDomain: "server-b.com",
			want:        false,
		},
		{
			name:        "subdomain is not local",
			identity:    FederatedIdentity{Username: "alice", Domain: "chat.server-a.com"},
			localDomain: "server-a.com",
			want:        false,
		},
		{
			name:        "empty local domain never matches",
			identity:    FederatedIdentity{Username: "alice", Domain: "server-a.com"},
			localDomain: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.identity.IsLocal(tt.localDomain))
		})
	}
}

func TestParseFederatedRoom(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FederatedRoom
		wantErr bool
	}{
		{
			name:  "valid room",
			input: "general@server-b.com",
			want:  FederatedRoom{RoomName: "general", Domain: "server-b.com"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "missing @",
			input:   "generalserver-b.com",
			wantErr: true,
		},
		{
			name:    "leading @ — empty room name",
			input:   "@server-b.com",
			wantErr: true,
		},
		{
			name:    "trailing @ — empty domain",
			input:   "general@",
			wantErr: true,
		},
		{
			name:    "multiple @ signs",
			input:   "general@server@b.com",
			wantErr: true,
		},
		{
			name:  "leading and trailing whitespace trimmed",
			input: "  general@server-b.com  ",
			want:  FederatedRoom{RoomName: "general", Domain: "server-b.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFederatedRoom(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFederatedRoom_String(t *testing.T) {
	f := FederatedRoom{RoomName: "general", Domain: "server-b.com"}
	assert.Equal(t, "general@server-b.com", f.String())
}

func TestFederatedRoom_RoundTrip(t *testing.T) {
	inputs := []string{
		"general@server-b.com",
		"lobby@localhost:8081",
		"dev-chat@my.server.example",
	}
	for _, s := range inputs {
		t.Run(s, func(t *testing.T) {
			f, err := ParseFederatedRoom(s)
			require.NoError(t, err)
			assert.Equal(t, s, f.String())
		})
	}
}
