package apihandler

import (
	"net/http"

	apiopenapi "server/api/openapi"

	"github.com/labstack/echo/v5"
)

// ServeOpenAPI serves the embedded OpenAPI specification.
func ServeOpenAPI(c *echo.Context) error {
	spec, err := apiopenapi.Spec.ReadFile("openapi.yaml")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load OpenAPI specification")
	}
	c.Response().Header().Set("Content-Type", "application/yaml")
	return c.String(http.StatusOK, string(spec))
}
