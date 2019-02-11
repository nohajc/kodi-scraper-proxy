#include <curl/curl.h>
#include <dlfcn.h>
#include <cstdio>
#include <cstdarg>
#include <utility>
#include <unordered_map>
#include <memory>
#include <string>

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


extern "C" {
    CURLcode curl_easy_setopt(CURL* handle, CURLoption option, ...);
    CURL* curl_easy_init();
    void curl_easy_reset(CURL* handle);
    void curl_easy_cleanup(CURL* handle);
    CURLcode curl_easy_perform(CURL* handle);
}


typedef size_t (*write_callback_ptr_t)(char *ptr, size_t size, size_t nmemb, void *userdata);


struct handle_ctx {
    CURL *handle;
    write_callback_ptr_t orig_write_callback = (write_callback_ptr_t)fwrite;
    void* userdata;
    std::string request_url;
    std::string response_body;
};

// TODO: thread-safe access
std::unordered_map<CURL*, std::unique_ptr<handle_ctx>> g_contextForHandle;

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

#define TRACE_CALL(handle) trace_call _call(__FUNCTION__, handle);
#define TRACE_CALL_WITH(option, handle) trace_call _call(__FUNCTION__, #option, handle)

static size_t write_callback_hook(char *ptr, size_t size, size_t nmemb, handle_ctx *context) {
    TRACE_CALL(context->handle);
    auto bytesProcessed = context->orig_write_callback(ptr, size, nmemb, context->userdata);

    context->response_body.append(ptr, bytesProcessed);

    return bytesProcessed;
}

CURL* curl_easy_init() {
    TRACE_CALL(nullptr)
    auto handle = orig_curl_easy_init();
    fprintf(stderr, "   Creating handle %p\n", handle);
    g_contextForHandle[handle] = std::make_unique<handle_ctx>(handle_ctx{handle});
    return handle;
}

CURLcode curl_easy_setopt(CURL *handle, CURLoption option, ...) {
    void* args_copy = __builtin_apply_args();
    va_list args;

    auto it = g_contextForHandle.find(handle);
    if (it != g_contextForHandle.end()) {
        auto context = it->second.get();

        switch (option) {
        case CURLOPT_URL:
        {
            //fprintf(stderr, "Called curl_easy_setopt CURLOPT_URL with handle %p\n", handle);
            TRACE_CALL_WITH(CURLOPT_URL, handle);
            va_start(args, option);
            auto url = va_arg(args, char*);
            context->request_url = url;
            va_end(args);
            return orig_curl_easy_setopt(handle, option, url);
        }
        case CURLOPT_WRITEDATA:
        {
            //fprintf(stderr, "Called curl_easy_setopt CURLOPT_WRITEDATA with handle %p\n", handle);
            TRACE_CALL_WITH(CURLOPT_WRITEDATA, handle);
            va_start(args, option);
            auto data = va_arg(args, void*);
            context->userdata = data;
            va_end(args);
            return orig_curl_easy_setopt(handle, option, context);
        }
        case CURLOPT_WRITEFUNCTION:
        {
            //fprintf(stderr, "Called curl_easy_setopt CURLOPT_WRITEFUNCTION with handle %p\n", handle);
            TRACE_CALL_WITH(CURLOPT_WRITEFUNCTION, handle);
            va_start(args, option);
            auto write_callback_ptr = va_arg(args, write_callback_ptr_t);
            context->orig_write_callback = write_callback_ptr;
            va_end(args);
            return orig_curl_easy_setopt(handle, option, write_callback_hook);
        }
        default:;
        }
    }

    void* ret = __builtin_apply(orig_curl_easy_setopt, args_copy, 128);
    __builtin_return(ret);
}

static void log_response(CURL* handle) {
    auto it = g_contextForHandle.find(handle);
    if (it != g_contextForHandle.end()) {
        auto context = it->second.get();
        auto& resp = context->response_body;
        fprintf(stderr, "   Client received %d bytes from %s: '%s'\n", resp.size(), context->request_url.c_str(), resp.c_str());
    }
}

void curl_easy_reset(CURL* handle) {
    TRACE_CALL(handle);
    log_response(handle);
    g_contextForHandle.erase(handle);
    orig_curl_easy_reset(handle);
}

void curl_easy_cleanup(CURL* handle) {
    TRACE_CALL(handle);
    log_response(handle);
    g_contextForHandle.erase(handle);
    orig_curl_easy_cleanup(handle);
}

CURLcode curl_easy_perform(CURL* handle) {
    TRACE_CALL(handle);
    return orig_curl_easy_perform(handle);
}
