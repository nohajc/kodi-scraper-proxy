package api

import "io"

// ResponseAdapter is an interface for plugins
// which define HTTP response body filters
type ResponseAdapter interface {
	AppliesTo(requestHost string) bool
	// current limitation is that you must always consume the entire input
	ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestHost string, requestPath string)
}
