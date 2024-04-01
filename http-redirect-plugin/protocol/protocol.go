package protocol

import "net/http"

// Plugins should export a variable called "Plugin" which implements this interface
type HttpRedirectPlugin interface {
	PreRequestHook(*http.Request)
}
