package api

import "io"

// ResponseAdapter is an interface for plugins
// which define HTTP response body filters
type ResponseAdapter interface {
	// hostname which this filter applies to
	Host() string
	// current limitation is that you must always consume the entire input
	ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestURL string)
}
