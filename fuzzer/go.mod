module fuzzer

go 1.27.4

replace server => ../server/

require (
	github.com/lib/pq v1.12.3
	server v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
)
