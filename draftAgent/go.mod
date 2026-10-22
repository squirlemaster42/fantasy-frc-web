module draftAgent

go 1.27.3

require (
	draftclient v0.0.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/net v0.61.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace draftclient => ../draftClient
