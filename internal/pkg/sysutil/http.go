package sysutil

import "net/http"

// HTTPClient is the portable network contract used by metadata and image
// adapters. Requests carry cancellation/deadlines; callers close response bodies.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
