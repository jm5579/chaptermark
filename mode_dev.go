//go:build !production

package main

// isProduction is false under `wails dev` and `go test`; the dev server's hot
// reload needs inline scripts, which the Content-Security-Policy would block.
const isProduction = false
