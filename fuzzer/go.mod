module fuzzer

go 1.26.2

replace server => ../server/

require (
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	server v0.0.0-00010101000000-000000000000
)
