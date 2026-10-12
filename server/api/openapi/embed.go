package apiopenapi

import (
	"embed"
)

//go:embed openapi.yaml
var Spec embed.FS
