package apimodel

import "time"

// IntegrationResponse is the JSON representation of a machine API key.
type IntegrationResponse struct {
	Id          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	ClientId    string    `json:"client_id"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at,omitempty"`
}

// CreateIntegrationRequest is the body for POST /api/v1/integrations.
type CreateIntegrationRequest struct {
	DisplayName string `json:"display_name"`
}

// CreateIntegrationResponse returns the new integration credentials.
type CreateIntegrationResponse struct {
	Integration IntegrationResponse `json:"integration"`
	ClientId    string              `json:"client_id"`
	ClientSecret string             `json:"client_secret"`
}
