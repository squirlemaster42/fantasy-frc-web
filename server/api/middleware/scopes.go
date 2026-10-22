package apimiddleware

// Scope defines a machine API permission.
type Scope string

const (
	// ScopeDraftsRead allows reading draft data.
	ScopeDraftsRead Scope = "drafts:read"
	// ScopeDraftsWrite allows creating and modifying drafts and picks.
	ScopeDraftsWrite Scope = "drafts:write"
	// ScopeInvitesRead allows reading invites.
	ScopeInvitesRead Scope = "invites:read"
	// ScopeInvitesWrite allows creating, accepting, and declining invites.
	ScopeInvitesWrite Scope = "invites:write"
	// ScopeIntegrationsRead allows reading own API keys.
	ScopeIntegrationsRead Scope = "integrations:read"
	// ScopeIntegrationsWrite allows creating and revoking own API keys.
	ScopeIntegrationsWrite Scope = "integrations:write"
	// ScopeAdmin allows administrative draft actions.
	ScopeAdmin Scope = "admin"
)
