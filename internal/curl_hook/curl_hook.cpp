#include <curl/curl.h>
#include <dlfcn.h>
#include <cstdio>
#include <cstdarg>
#include <cassert>
#include <utility>
#include <unordered_map>
#include <memory>
#include <string>
#include <future>
#include <mutex>

#include "libbridge.h"

#undef curl_easy_setopt


template <typename F>
class func_ptr {
    F* ptr;
    const char* func_name;
public:
    func_ptr(F*, const char* name) : ptr(nullptr), func_name(name) {}

    F* get() {
        if (!ptr) {
            ptr = reinterpret_cast<F*>(dlsym(RTLD_NEXT, func_name));
        }
        return ptr;
    }

    template <typename... Args>
    decltype(auto) operator()(Args&&... args) {
        return get()(std::forward<Args>(args)...);
    }

    (*operator void())(...) {
        return reinterpret_cast<void(*)(...)>(get());
    }
};

#define ORIG(func) func_ptr orig_ ## func(func, #func)


ORIG(curl_easy_setopt);
ORIG(curl_easy_init);
ORIG(curl_easy_reset);
ORIG(curl_easy_cleanup);
ORIG(curl_easy_perform);
ORIG(curl_multi_add_handle);
//ORIG(curl_multi_perform);
ORIG(curl_multi_info_read);


extern "C" {
    CURLcode curl_easy_setopt(CURL* handle, CURLoption option, ...);
    CURL* curl_easy_init();
    void curl_easy_reset(CURL* handle);
    void curl_easy_cleanup(CURL* handle);
    CURLcode curl_easy_perform(CURL* handle);
    CURLMcode curl_multi_add_handle(CURLM* multi_handle, CURL* easy_handle);
    //CURLMcode curl_multi_perform(CURLM* multi_handle, int* running_handles);
    CURLMsg* curl_multi_info_read(CURLM* multi_handle, int* msgs_in_queue);
}


//typedef size_t (*write_callback_ptr_t)(char *ptr, size_t size, size_t nmemb, void *userdata);

struct url_components {
    std::string host;
    std::string path;
};

class handle_ctx {
    std::promise<void> complete_prom;
    std::future<void> complete_fut = complete_prom.get_future();

public:
    CURL* handle;
    write_callback_ptr_t orig_write_callback = (write_callback_ptr_t)fwrite;
    void* userdata = nullptr;
    url_components request_url;
    bool easy_perform_called = false;

    handle_ctx(CURL* h) : handle(h) {}

    void complete() {
        complete_prom.set_value();
    }

    void wait_for_completion() {
        complete_fut.get();
        // reset for possible next connection
        complete_prom = {};
        complete_fut = complete_prom.get_future();
    }
};

std::unordered_map<CURL*, std::unique_ptr<handle_ctx>> g_contextForHandle;
std::mutex g_contextForHandleMutex;

handle_ctx* create_context(CURL* handle) {
    std::lock_guard<std::mutex> lock(g_contextForHandleMutex);
    return (g_contextForHandle[handle] = std::make_unique<handle_ctx>(handle)).get();
}

void destroy_context(CURL* handle) {
    std::lock_guard<std::mutex> lock(g_contextForHandleMutex);
    g_contextForHandle.erase(handle);
}

handle_ctx* get_context(CURL* handle) {
    std::lock_guard<std::mutex> lock(g_contextForHandleMutex);
    return g_contextForHandle[handle].get();
}

url_components getURLComponents(const std::string& url) {
    auto hostPos = url.find("://");
    std::string fromHost;
    if (hostPos == std::string::npos) {
        fromHost = url;
    }
    else {
        fromHost = url.substr(hostPos + 3);
    }
    auto pathPos = fromHost.find("/");
    if (pathPos == std::string::npos) {
        return {fromHost, "/"};
    }

    return {fromHost.substr(0, pathPos), fromHost.substr(pathPos)};
}

GoString to_go_string_view(const std::string& str) {
    return{ &str[0], static_cast<GoInt>(str.size()) };
}

class trace_call {
    const char* func_name;
    const char* option;
    CURL* handle;
public:
    trace_call(const char* func, CURL* hnd) : func_name(func), option(nullptr), handle(hnd) {
        if (handle) {
            fprintf(stderr, "-> %s with handle %p\n", func_name, handle);
        }
        else {
            fprintf(stderr, "-> %s\n", func_name);
        }
    }

    trace_call(const char* func, const char* opt, CURL* hnd) : func_name(func), option(opt), handle(hnd) {
        fprintf(stderr, "-> %s %s with handle %p\n", func_name, option, handle);
    }
    ~trace_call() {
        if (handle) {
            if (option) {
                fprintf(stderr, "<- %s %s with handle %p\n", func_name, option, handle);
            }
            else {
                fprintf(stderr, "<- %s with handle %p\n", func_name, handle);
            }
        }
        else {
            fprintf(stderr, "<- %s\n", func_name);
        }
    }
};

#define FUNC_WITH(option) __FUNCTION__ #option

#ifdef DEBUG
#define TRACE_CALL(handle) trace_call _call(__FUNCTION__, handle);
#define TRACE_CALL_WITH(option, handle) trace_call _call(__FUNCTION__, #option, handle)
#define DLOG(fmt, ...) fprintf(stderr, fmt, ## __VA_ARGS__)
#else
#define TRACE_CALL(handle)
#define TRACE_CALL_WITH(option, handle)
#define DLOG(fmt, ...)
#endif

static size_t write_callback_hook(char *ptr, size_t size, size_t nmemb, handle_ctx *context) {
    TRACE_CALL(context->handle);
    GoSlice data{ ptr, static_cast<GoInt>(nmemb), static_cast<GoInt>(nmemb) };
    return ResponseWrite(context, data);
}

CURL* curl_easy_init() {
    TRACE_CALL(nullptr);
    auto handle = orig_curl_easy_init();
    DLOG("   Creating handle %p\n", handle);
    create_context(handle);
    return handle;
}

CURLcode curl_easy_setopt(CURL *handle, CURLoption option, ...) {
    void* args_copy = __builtin_apply_args();
    va_list args;

    auto context = get_context(handle);
    assert(context != nullptr);

    switch (option) {
    case CURLOPT_URL:
    {
        TRACE_CALL_WITH(CURLOPT_URL, handle);
        va_start(args, option);
        auto url_str = va_arg(args, char*);
        context->request_url = getURLComponents(url_str);
        va_end(args);
        return orig_curl_easy_setopt(handle, option, url_str);
    }
    case CURLOPT_WRITEDATA:
    {
        TRACE_CALL_WITH(CURLOPT_WRITEDATA, handle);
        va_start(args, option);
        auto data = va_arg(args, void*);
        context->userdata = data;
        va_end(args);
        return orig_curl_easy_setopt(handle, option, context);
    }
    case CURLOPT_WRITEFUNCTION:
    {
        TRACE_CALL_WITH(CURLOPT_WRITEFUNCTION, handle);
        va_start(args, option);
        auto write_callback_ptr = va_arg(args, write_callback_ptr_t);
        context->orig_write_callback = write_callback_ptr;
        va_end(args);
        // First we need to set default WRITEDATA
        // in case the client does not use userdata
        // because we always need access to context.
        orig_curl_easy_setopt(handle, CURLOPT_WRITEDATA, context);
        return orig_curl_easy_setopt(handle, option, write_callback_hook);
    }
    default:;
    }

    void* ret = __builtin_apply(orig_curl_easy_setopt, args_copy, 128);
    __builtin_return(ret);
}

/*static void log_response(CURL* handle) {
    auto it = g_contextForHandle.find(handle);
    if (it != g_contextForHandle.end()) {
        auto context = it->second.get();
        auto& resp = context->response_body;
        fprintf(stderr, "   Client received %d bytes from %s: '%s'\n", resp.size(), context->request_url.c_str(), resp.c_str());
    }
}*/

void curl_easy_reset(CURL* handle) {
    TRACE_CALL(handle);
    //log_response(handle);
    create_context(handle);
    orig_curl_easy_reset(handle);
}

void curl_easy_cleanup(CURL* handle) {
    TRACE_CALL(handle);
    //log_response(handle);
    destroy_context(handle);
    orig_curl_easy_cleanup(handle);
}

static void close_callback(void* ctx) {
    DLOG("-> close_callback called with context %p\n", ctx);
    auto context = reinterpret_cast<handle_ctx*>(ctx);
    assert(context != nullptr);
    context->complete();
    DLOG("<- close_callback\n");
}

void do_filter_request(handle_ctx* context) {
    // These string references will be valid until handle cleanup
    // so it should be OK to pass them to Golang as GoStrings
    auto& urlHost = context->request_url.host;
    auto& urlPath = context->request_url.path;

    FilterRequest(
        context, to_go_string_view(urlHost), to_go_string_view(urlPath),
        context->orig_write_callback, close_callback, context->userdata);
}

CURLcode curl_easy_perform(CURL* handle) {
    TRACE_CALL(handle);
    auto context = get_context(handle);
    assert(context != nullptr);
    context->easy_perform_called = true;
    do_filter_request(context);

    auto code = orig_curl_easy_perform(handle);
    // signal that the response is complete
    ResponseClose(context);
    context->wait_for_completion();

    return code;
}

CURLMcode curl_multi_add_handle(CURLM* multi_handle, CURL* easy_handle) {
    TRACE_CALL(easy_handle);
    auto context = get_context(easy_handle);
    assert(context != nullptr);
    if (!context->easy_perform_called) {
        do_filter_request(context);
    }
    return orig_curl_multi_add_handle(multi_handle, easy_handle);
}

/*CURLMcode curl_multi_perform(CURLM *multi_handle, int *running_handles) {
    //TRACE_CALL(nullptr);
    return orig_curl_multi_perform(multi_handle, running_handles);
}*/

CURLMsg* curl_multi_info_read(CURLM* multi_handle, int* msgs_in_queue) {
    TRACE_CALL(nullptr);

    auto msg = orig_curl_multi_info_read(multi_handle, msgs_in_queue);

    if (msg && msg->msg == CURLMSG_DONE) {
        DLOG("\twith handle %p\n", msg->easy_handle);
        auto context = get_context(msg->easy_handle);
        assert(context != nullptr);
        if (!context->easy_perform_called) {
            // signal that the response is complete
            ResponseClose(context);
            context->wait_for_completion();
        }
    }

    return msg;
}
