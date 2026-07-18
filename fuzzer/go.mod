module fuzzer

go 1.26.5

replace server => ../server/

require (
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	server v0.0.0-00010101000000-000000000000
)
