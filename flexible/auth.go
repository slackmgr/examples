package main

import (
	"context"
	"fmt"

	api "github.com/slackmgr/core/restapi"
	"github.com/slackmgr/examples/flexible/config"
	"github.com/slackmgr/plugins/oidc"
	common "github.com/slackmgr/types"
)

// configureAuth enables authentication of the REST API according to cfg.Auth.Mode (API_AUTH_MODE):
//
//   - "off" (default): no authentication, all endpoints are open
//   - "apikeys": API keys, configured as SHA-256 hashes with scopes in the API settings file (apiKeys). Keys can
//     be added, rotated and revoked at runtime, as the settings file is hot-reloaded
//   - "oidc": OIDC/OAuth2 access tokens issued by OIDC_ISSUER_URL for the audience OIDC_AUDIENCE, validated by
//     the github.com/slackmgr/plugins/oidc authenticator
//   - "both": API keys first, then OIDC/OAuth2 access tokens
//
// With authentication enabled, all endpoints except GET /ping and GET /openapi.yaml require credentials and a
// scope (alerts:write, alerts:test or config:read), and the channel access rules in the API settings file
// (accessRules, defaultChannelAccess) restrict the channels each client may send alerts to.
func configureAuth(ctx context.Context, cfg *config.Config, apiServer *api.Server, logger common.Logger) error {
	switch cfg.Auth.Mode {
	case config.AuthModeOff:
		return nil
	case config.AuthModeAPIKeys:
		apiServer.WithAPIKeyAuth()
		return nil
	case config.AuthModeOIDC, config.AuthModeBoth:
		// New downloads the discovery document and key set of the issuer, so it fails if the identity provider
		// is unreachable or misconfigured.
		authenticator, err := oidc.New(ctx, cfg.Auth.OIDCIssuerURL,
			oidc.WithAudience(cfg.Auth.OIDCAudience),
			oidc.WithLogger(logger),
		)
		if err != nil {
			return fmt.Errorf("failed to create the OIDC authenticator: %w", err)
		}

		if cfg.Auth.Mode == config.AuthModeBoth {
			apiServer.WithAPIKeyAuth()
		}

		apiServer.WithAuthenticator(authenticator)

		return nil
	default:
		return fmt.Errorf("invalid API_AUTH_MODE %q: must be %q, %q, %q or %q", cfg.Auth.Mode,
			config.AuthModeOff, config.AuthModeAPIKeys, config.AuthModeOIDC, config.AuthModeBoth)
	}
}
