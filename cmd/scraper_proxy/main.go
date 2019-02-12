package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/nohajc/kodi-scraper-proxy/internal/filter"
)

// NewProxyWithScraperAdapters returns new HTTP proxy server with the given scraper adapters
func NewProxyWithScraperAdapters(adapters ...filter.ScraperAdapter) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		var reqBodyBuf bytes.Buffer
		io.Copy(&reqBodyBuf, ctx.Req.Body)
		reqBody, _ := ioutil.ReadAll(&reqBodyBuf)
		log.Printf("Request: %s, %v\n", ctx.Req.URL.Path, reqBody)

		for _, adp := range adapters {
			if ctx.Req.Host == adp.Host() {
				return ResponseFilter(adp, resp, ctx)
			}
		}

		return resp
	})

	return proxy
}

// ResponseFilter modifies response from the scraper source to apply the new ordering
func ResponseFilter(adp filter.ScraperAdapter, resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	reader, writer := io.Pipe()

	adp.ResponseBodyFilter(resp.Body, writer, ctx.Req.URL.Path)
	resp.Body = reader

	return resp
}

func main() {
	shows := filter.LoadConfig("ordering.yaml")
	log.Printf("%+v\n", shows)

	proxy := NewProxyWithScraperAdapters(&filter.TMDBScraperOrderingAdapter{filter.NewOfflineOrderingMap(shows)})
	proxy.Verbose = false

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
