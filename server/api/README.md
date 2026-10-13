# Fantasy FRC Machine API

This package implements the `/api/v1` machine-to-machine API for Fantasy FRC.

## Authentication

API clients authenticate using OAuth2-style client credentials:

```bash
curl -X POST https://example.com/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "client_credentials",
    "client_id": "ffrc_xxxxxxxx",
    "client_secret": "yyyyyyyy"
  }'
```

The response contains a short-lived JWT access token:

```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

Include the token in the `Authorization` header for subsequent requests:

```bash
curl https://example.com/api/v1/drafts \
  -H "Authorization: Bearer eyJ..."
```

## Scopes

JWT tokens include a `scopes` claim that controls access to API endpoints.
The following scopes are supported:

| Scope | Access |
|-------|--------|
| `drafts:read` | Read drafts, picks, and scores |
| `drafts:write` | Create and modify drafts and picks |
| `invites:read` | List invites |
| `invites:write` | Create, accept, and decline invites |
| `integrations:read` | List own API keys |
| `integrations:write` | Create and revoke own API keys |
| `admin` | Admin draft actions (skip, extend, make, undo picks) |
| `full_access` | Grants all permissions (default for web-created keys) |

## OpenAPI

The OpenAPI specification is served at `/api/v1/openapi.yaml`.

## Creating an Integration

```bash
curl -X POST https://example.com/api/v1/integrations \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "My Bot",
    "scopes": ["drafts:read", "drafts:write"]
  }'
```
