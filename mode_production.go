//go:build production

package main

// isProduction is true in `wails build` output, which Wails compiles with the
// "production" build tag. It turns on the Content-Security-Policy.
const isProduction = true
