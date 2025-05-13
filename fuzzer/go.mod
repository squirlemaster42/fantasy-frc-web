module fuzzer

go 1.23.0

toolchain go1.23.9

replace server => ../server/

require server v0.0.0-00010101000000-000000000000

require (
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
)
