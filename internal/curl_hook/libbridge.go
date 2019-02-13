package main

// #include <stdlib.h>
// #define INVOKE(return_type, callback, ...) typedef return_type (*callback ## _ptr_t)(__VA_ARGS__); \
//	inline return_type invoke_ ## callback(callback ## _ptr_t fn, ## __VA_ARGS__)
//
// INVOKE(size_t, write_callback, char *ptr, size_t size, size_t nmemb, void *userdata) {
// 	return fn(ptr, size, nmemb, userdata);
// }
//
// INVOKE(void, close_callback, void *hnd) {
//	return fn(hnd);
// }
import "C"

import (
	"io"
	"log"
	"os"
	"plugin"
	"sync"
	"unsafe"

	"github.com/nohajc/kodi-scraper-proxy/pkg/api"
)

type context struct {
	ReadEnd  io.ReadCloser
	WriteEnd io.WriteCloser
}

var (
	reqMap    = make(map[uintptr](*context))
	reqMapMtx sync.Mutex
)

// NopResponseAdapter does nothing
type NopResponseAdapter struct{}

// Host retuns empty string so that no valid request can match this filter
func (*NopResponseAdapter) Host() string {
	return ""
}

// ResponseBodyFilter is never called because empty host is invalid
func (*NopResponseAdapter) ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestURL string) {
}

var gAdapter api.ResponseAdapter = &NopResponseAdapter{}

func init() {
	filterPluginPath := os.Getenv("FILTER_PLUGIN")
	if filterPluginPath == "" {
		log.Println("Error: FILTER_PLUGIN has to be specified")
		return
	}

	filterPlugin, err := plugin.Open(filterPluginPath)
	if err != nil {
		log.Printf("Error loading plugin: %v\n", err)
		return
	}

	adapterSym, err := filterPlugin.Lookup("PluginResponseAdapter")
	if err != nil {
		log.Printf("Warning: %v\n", err)
	} else {
		adapterPtr := adapterSym.(*api.ResponseAdapter)
		/*if !ok {
			log.Printf("Error: ResponseAdapter doesn't have the right type")
			return
		}*/
		// could be nil if there was an error during plugin initialization
		if adapterPtr != nil {
			gAdapter = *adapterPtr
		}
	}
}

// CallbackWriteCloser delegates its Write and Close functions
// to callbacks obtained from the user of this library
type CallbackWriteCloser struct {
	Hnd      unsafe.Pointer
	Userdata unsafe.Pointer
	WriteCB  C.write_callback_ptr_t
	CloseCB  C.close_callback_ptr_t
}

// Write writes bytes
func (cb *CallbackWriteCloser) Write(data []byte) (int, error) {
	//log.Printf("CallbackWriteCloser Write with handle %v\n", cb.Hnd)
	rawBytes := C.CBytes(data)
	defer C.free(rawBytes)
	return int(C.invoke_write_callback(cb.WriteCB, (*C.char)(rawBytes), 1, C.size_t(len(data)), cb.Userdata)), nil
}

// Close closes the WriteCloser
func (cb *CallbackWriteCloser) Close() error {
	//log.Printf("CallbackWriteCloser Close with handle %v\n", cb.Hnd)
	C.invoke_close_callback(cb.CloseCB, cb.Hnd)

	reqMapMtx.Lock()
	delete(reqMap, uintptr(cb.Hnd))
	reqMapMtx.Unlock()
	return nil
}

// FilterRequest registers curl handle
//export FilterRequest
func FilterRequest(
	hnd unsafe.Pointer,
	urlHost string, urlPath string,
	writeCallback C.write_callback_ptr_t,
	closeCallback C.close_callback_ptr_t,
	userdata unsafe.Pointer) {

	//log.Printf("FilterRequest with handle %v\n", hnd)

	ctx := &context{}

	reqMapMtx.Lock()
	reqMap[uintptr(hnd)] = ctx
	reqMapMtx.Unlock()

	callbackWriteCloser := &CallbackWriteCloser{
		Hnd:      hnd,
		Userdata: userdata,
		WriteCB:  writeCallback,
		CloseCB:  closeCallback,
	}

	if gAdapter.Host() == urlHost {
		r, w := io.Pipe()
		ctx.ReadEnd = r
		ctx.WriteEnd = w
		log.Printf("Applying response filter to %s%s", urlHost, urlPath)
		gAdapter.ResponseBodyFilter(ctx.ReadEnd, callbackWriteCloser, urlPath)
	} else {
		// if adapter host does not match url host, do not call BodyFilter at all;
		// set ctx.WriteEnd to CallbackWriteCloser instead for direct copy
		ctx.WriteEnd = callbackWriteCloser
	}
}

// ResponseWrite takes curl handle and data. It sends the data to the corresponding request pipe.
//export ResponseWrite
func ResponseWrite(hnd unsafe.Pointer, data []byte) C.size_t {
	//log.Printf("ResponseWrite with handle %v\n", hnd)
	reqMapMtx.Lock()
	ctx := reqMap[uintptr(hnd)]
	if ctx == nil {
		panic("ResponseWrite: got nil context!")
	}
	reqMapMtx.Unlock()

	written, _ := ctx.WriteEnd.Write(data)
	return C.size_t(written)
}

// ResponseClose signals that the response is complete, i. e. ResponseWrite won't be called again
//export ResponseClose
func ResponseClose(hnd unsafe.Pointer) {
	//log.Printf("ResponseClose with handle %v\n", hnd)
	reqMapMtx.Lock()
	ctx := reqMap[uintptr(hnd)]
	reqMapMtx.Unlock()

	ctx.WriteEnd.Close()
}

func main() {}
