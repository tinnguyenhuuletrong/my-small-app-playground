package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/http-redirect-plugin/protocol"
)

func logRequest(req *http.Request) {
	result, err := httputil.DumpRequest(req, true)
	if err != nil {
		slog.Error("Failed to print request", "err", err)
		return
	}
	slog.Info("Request sent:", "req", result)
}

func logRequestLikeCUrl(req *http.Request) {
	panic("Unimplemented!")
}

func main() { /*empty because it does nothing*/ }

type PluginStr struct{}

// Compile time check for
// PreRequestHook implements protocol.HttpRedirectPlugin.
var _ protocol.HttpRedirectPlugin = PluginStr{}

// PreRequestHook implements protocol.HttpRedirectPlugin.
func (p PluginStr) PreRequestHook(req *http.Request) {
	logRequest(req)
}

var Plugin = PluginStr{}
