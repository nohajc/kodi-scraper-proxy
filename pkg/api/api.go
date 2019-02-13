package api

import "io"

// ResponseAdapter is an interface for plugins
// which define HTTP response body filters
type ResponseAdapter interface {
	Host() string
	ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestURL string)
}
