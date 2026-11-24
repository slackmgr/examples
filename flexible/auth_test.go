package main

import (
	"strings"
	"testing"

	api "github.com/slackmgr/core/restapi"
	"github.com/slackmgr/examples/flexible/config"
	common "github.com/slackmgr/types"
)

func TestConfigureAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		auth    config.AuthConfig
		wantErr string
	}{
		{name: "off", auth: config.AuthConfig{Mode: config.AuthModeOff}},
		{name: "api keys", auth: config.AuthConfig{Mode: config.AuthModeAPIKeys}},
		{
			name:    "oidc with a plain http issuer",
			auth:    config.AuthConfig{Mode: config.AuthModeOIDC, OIDCIssuerURL: "http://login.example.com", OIDCAudience: "slackmgr-api"},
			wantErr: "failed to create the OIDC authenticator",
		},
		{
			name:    "both without issuer",
			auth:    config.AuthConfig{Mode: config.AuthModeBoth, OIDCAudience: "slackmgr-api"},
			wantErr: "failed to create the OIDC authenticator",
		},
		{name: "invalid mode", auth: config.AuthConfig{Mode: "basic"}, wantErr: `invalid API_AUTH_MODE "basic"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{Auth: tt.auth}
			server := api.New(nil, &common.NoopLogger{}, nil)

			err := configureAuth(t.Context(), cfg, server, &common.NoopLogger{})

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetAPICfgAuth(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{RestPort: "8080", Auth: config.AuthConfig{Permissive: true, ClientBinding: "override"}}
	apiCfg := cfg.GetAPICfg()

	if apiCfg.Auth == nil || !apiCfg.Auth.Permissive || apiCfg.Auth.ClientBinding != "override" {
		t.Fatalf("unexpected auth config: %+v", apiCfg.Auth)
	}
}
