module github.com/localitas/localitas-app-mmoney

go 1.26.3

require (
	github.com/grandcat/zeroconf v1.0.0
	github.com/localitas/localitas-go v0.0.0-00010101000000-000000000000
	github.com/mattn/go-sqlite3 v1.14.45
	github.com/pquerna/otp v1.5.0
	github.com/urfave/cli/v3 v3.9.1
)

require (
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/miekg/dns v1.1.27 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/localitas/localitas-go => ../localitas-go
