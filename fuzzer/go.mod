module fuzzer

go 1.24.0

replace server => ../server/

require (
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	server v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/labstack/echo/v4 v4.13.3 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)
