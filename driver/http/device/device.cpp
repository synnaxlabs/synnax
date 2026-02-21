// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <utility>

#include "glog/logging.h"
#include <curl/curl.h>

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"

namespace driver::http::device {
namespace {
struct CurlGlobal {
    CurlGlobal() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~CurlGlobal() { curl_global_cleanup(); }
};

void ensure_curl_initialized() {
    static CurlGlobal instance;
}

size_t write_callback(char *ptr, size_t size, size_t nmemb, void *userdata) {
    auto *response = static_cast<std::string *>(userdata);
    response->append(ptr, size * nmemb);
    return size * nmemb;
}

x::errors::Error parse_curl_error(CURLcode code) {
    if (code == CURLE_OK) return x::errors::NIL;
    const auto code_str = std::to_string(static_cast<int>(code));
    switch (code) {
        case CURLE_COULDNT_CONNECT:
        case CURLE_COULDNT_RESOLVE_HOST:
        case CURLE_COULDNT_RESOLVE_PROXY:
        case CURLE_OPERATION_TIMEDOUT:
            return x::errors::Error(
                http::errors::UNREACHABLE_ERROR.sub(code_str),
                curl_easy_strerror(code)
            );
        default:
            return x::errors::Error(
                http::errors::CLIENT_ERROR.sub(code_str),
                curl_easy_strerror(code)
            );
    }
}
}

struct MultiHandle {
    CURLM *handle = nullptr;

    MultiHandle(): handle(curl_multi_init()) {}

    ~MultiHandle() {
        if (handle != nullptr) curl_multi_cleanup(handle);
    }

    MultiHandle(const MultiHandle &) = delete;
    MultiHandle &operator=(const MultiHandle &) = delete;

    MultiHandle(MultiHandle &&other) noexcept: handle(other.handle) {
        other.handle = nullptr;
    }

    MultiHandle &operator=(MultiHandle &&other) noexcept {
        if (this != &other) {
            if (handle != nullptr) curl_multi_cleanup(handle);
            handle = other.handle;
            other.handle = nullptr;
        }
        return *this;
    }
};

/// @brief internal handle that wraps a pre-configured curl easy handle.
struct Handle {
    CURL *handle = nullptr;
    struct curl_slist *headers = nullptr;
    std::string response_body;
    Method method;
    std::string expected_content_type;
    CURLcode result_code;

    Handle() = default;

    ~Handle() {
        if (headers != nullptr) curl_slist_free_all(headers);
        if (handle != nullptr) curl_easy_cleanup(handle);
    }

    Handle(const Handle &) = delete;
    Handle &operator=(const Handle &) = delete;

    Handle(Handle &&other) noexcept:
        handle(other.handle),
        headers(other.headers),
        response_body(std::move(other.response_body)),
        method(other.method),
        expected_content_type(std::move(other.expected_content_type)),
        result_code(other.result_code) {
        other.handle = nullptr;
        other.headers = nullptr;
    }
};

namespace {
/// @brief sets the request body on a handle. CURLOPT_POSTFIELDS does not copy — it
/// stores the pointer, so body must outlive the perform call. Methods that don't accept
/// bodies silently skip body setting.
void set_body(Handle &h, const std::string &body) {
    if (!has_request_body(h.method)) return;
    if (!body.empty()) {
        curl_easy_setopt(h.handle, CURLOPT_POSTFIELDS, body.c_str());
        curl_easy_setopt(h.handle, CURLOPT_POSTFIELDSIZE, body.size());
    } else {
        curl_easy_setopt(h.handle, CURLOPT_POSTFIELDS, nullptr);
        curl_easy_setopt(h.handle, CURLOPT_POSTFIELDSIZE, 0L);
    }
}

/// @brief builds a Response + Error pair from a completed handle whose result_code has
/// already been set by curl_easy_perform or via CURLOPT_PRIVATE.
std::pair<Response, x::errors::Error>
build_result(Handle &h, x::telem::TimeStamp start) {
    long status_code = 0;
    curl_easy_getinfo(h.handle, CURLINFO_RESPONSE_CODE, &status_code);
    double total_secs = 0;
    curl_easy_getinfo(h.handle, CURLINFO_TOTAL_TIME, &total_secs);
    const auto elapsed = x::telem::TimeSpan(static_cast<int64_t>(total_secs * 1e9));
    if (!has_response_body(h.method)) h.response_body.clear();
    x::errors::Error err = x::errors::NIL;
    if (h.result_code != CURLE_OK) {
        err = parse_curl_error(h.result_code);
    } else if (!h.expected_content_type.empty()) {
        char *ct = nullptr;
        curl_easy_getinfo(h.handle, CURLINFO_CONTENT_TYPE, &ct);
        if (ct != nullptr) {
            const std::string_view actual(ct);
            const auto n = h.expected_content_type.size();
            if (!actual.starts_with(h.expected_content_type) ||
                (actual.size() > n && actual[n] != ';'))
                err = x::errors::Error(
                    http::errors::PARSE_ERROR,
                    "expected content type '" + h.expected_content_type + "', got '" +
                        std::string(actual) + "'"
                );
        }
    }
    return {
        Response{
            .status_code = static_cast<int>(status_code),
            .body = std::move(h.response_body),
            .time_range = {start, start + elapsed},
        },
        err,
    };
}
}

Client::Client() = default;

Client::Client(Client &&other) noexcept:
    multi_handle_(std::move(other.multi_handle_)),
    handles_(std::move(other.handles_)) {}

std::pair<Client, x::errors::Error>
Client::create(ConnectionConfig config, const std::vector<RequestConfig> &requests) {
    for (const auto &req: requests) {
        if (!has_response_body(req.method) && !req.response_content_type.empty())
            return {
                Client(),
                x::errors::Error(
                    http::errors::CLIENT_ERROR,
                    std::string(to_string(req.method)) +
                        " requests must not set response_content_type"
                ),
            };
    }
    return {Client(std::move(config), requests), x::errors::NIL};
}

Client::Client(ConnectionConfig config, const std::vector<RequestConfig> &requests) {
    ensure_curl_initialized();
    multi_handle_ = std::make_unique<MultiHandle>();
    handles_.reserve(requests.size());

    for (const auto &req: requests) {
        Handle h;
        h.handle = curl_easy_init();
        if (h.handle == nullptr) continue;

        // URL (static per handle).
        CURLU *u = curl_url();
        curl_url_set(u, CURLUPART_URL, config.base_url.c_str(), 0);
        if (!req.path.empty()) {
            std::string path = req.path;
            if (path.front() != '/') path.insert(path.begin(), '/');
            curl_url_set(u, CURLUPART_PATH, path.c_str(), 0);
        }
        for (const auto &[k, v]: req.query_params) {
            const auto param = k + "=" + v;
            curl_url_set(
                u,
                CURLUPART_QUERY,
                param.c_str(),
                CURLU_APPENDQUERY | CURLU_URLENCODE
            );
        }
        char *url = nullptr;
        curl_url_get(u, CURLUPART_URL, &url, 0);
        curl_easy_setopt(h.handle, CURLOPT_URL, url);
        curl_free(url);
        curl_url_cleanup(u);

        // Timeout (static per handle).
        curl_easy_setopt(
            h.handle,
            CURLOPT_TIMEOUT_MS,
            static_cast<long>(config.timeout.milliseconds())
        );

        // Write callback (static).
        curl_easy_setopt(h.handle, CURLOPT_WRITEFUNCTION, write_callback);

        // SSL verification (static).
        if (!config.verify_ssl) {
            curl_easy_setopt(h.handle, CURLOPT_SSL_VERIFYPEER, 0L);
            curl_easy_setopt(h.handle, CURLOPT_SSL_VERIFYHOST, 0L);
        }

        // HTTP method (static per handle).
        h.method = req.method;
        if (h.method == Method::HEAD)
            curl_easy_setopt(h.handle, CURLOPT_NOBODY, 1L);
        else if (h.method == Method::POST)
            curl_easy_setopt(h.handle, CURLOPT_POST, 1L);
        else if (h.method != Method::GET)
            curl_easy_setopt(h.handle, CURLOPT_CUSTOMREQUEST, to_string(h.method));

        // Auth headers (static).
        if (config.auth.type == "bearer") {
            const std::string hdr = "Authorization: Bearer " + config.auth.token;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        } else if (config.auth.type == "basic") {
            curl_easy_setopt(h.handle, CURLOPT_HTTPAUTH, CURLAUTH_BASIC);
            const std::string userpwd = config.auth.username + ":" +
                                        config.auth.password;
            curl_easy_setopt(h.handle, CURLOPT_USERPWD, userpwd.c_str());
        } else if (config.auth.type == "api_key") {
            const std::string hdr = config.auth.header + ": " + config.auth.key;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Connection-level headers (static).
        for (const auto &[k, v]: config.headers) {
            const std::string hdr = k + ": " + v;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Per-request headers (static).
        for (const auto &[k, v]: req.headers) {
            const std::string hdr = k + ": " + v;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Content-Type header (static). An empty entry suppresses curl's default.
        if (!req.request_content_type.empty()) {
            const std::string ct_hdr = "Content-Type: " + req.request_content_type;
            h.headers = curl_slist_append(h.headers, ct_hdr.c_str());
        } else if (has_request_body(req.method)) {
            h.headers = curl_slist_append(h.headers, "Content-Type:");
        }

        // Accept header and expected content type validation (static).
        if (!req.response_content_type.empty()) {
            h.expected_content_type = req.response_content_type;
            const std::string accept_hdr = "Accept: " + req.response_content_type;
            h.headers = curl_slist_append(h.headers, accept_hdr.c_str());
        }

        if (h.headers != nullptr)
            curl_easy_setopt(h.handle, CURLOPT_HTTPHEADER, h.headers);

        handles_.push_back(std::move(h));
        // Set WRITEDATA and PRIVATE after push_back so pointers target the handle's
        // final location in the vector (reserve prevents reallocation).
        auto &back = handles_.back();
        curl_easy_setopt(back.handle, CURLOPT_WRITEDATA, &back.response_body);
        curl_easy_setopt(back.handle, CURLOPT_PRIVATE, reinterpret_cast<char *>(&back));
    }
}

Client::~Client() = default;

std::pair<std::vector<std::pair<Response, x::errors::Error>>, x::errors::Error>
Client::execute_requests(
    const std::vector<std::string> &bodies,
    const x::telem::TimeSpan poll_timeout
) {
    static const std::string empty;

    // Single-handle fast path: use curl_easy_perform directly.
    if (handles_.size() == 1) {
        auto &h = handles_[0];
        h.response_body.clear();
        set_body(h, !bodies.empty() ? bodies[0] : empty);
        const auto start = x::telem::TimeStamp::now();
        h.result_code = curl_easy_perform(h.handle);
        return {{build_result(h, start)}, x::errors::NIL};
    }

    // Multi-handle path.
    auto *multi = multi_handle_->handle;

    for (size_t i = 0; i < handles_.size(); i++) {
        auto &h = handles_[i];
        h.response_body.clear();
        h.result_code = CURLE_OK;
        set_body(h, i < bodies.size() ? bodies[i] : empty);
        curl_multi_add_handle(multi, h.handle);
    }

    const auto start = x::telem::TimeStamp::now();

    int still_running = 0;
    const int timeout_ms = static_cast<int>(poll_timeout.milliseconds());
    do {
        const auto mc = curl_multi_perform(multi, &still_running);
        if (mc != CURLM_OK) {
            for (auto &h: handles_)
                curl_multi_remove_handle(multi, h.handle);
            return {
                {},
                x::errors::Error(http::errors::CLIENT_ERROR, curl_multi_strerror(mc)),
            };
        }
        if (still_running > 0) curl_multi_poll(multi, nullptr, 0, timeout_ms, nullptr);
    } while (still_running > 0);

    CURLMsg *msg;
    int msgs_left;
    while ((msg = curl_multi_info_read(multi, &msgs_left)) != nullptr) {
        if (msg->msg != CURLMSG_DONE) continue;
        char *private_ptr;
        curl_easy_getinfo(msg->easy_handle, CURLINFO_PRIVATE, &private_ptr);
        reinterpret_cast<Handle *>(private_ptr)->result_code = msg->data.result;
    }

    std::vector<std::pair<Response, x::errors::Error>> results;
    results.reserve(handles_.size());

    for (auto &h: handles_) {
        results.push_back(build_result(h, start));
        curl_multi_remove_handle(multi, h.handle);
    }

    return {std::move(results), x::errors::NIL};
}

std::pair<ConnectionConfig, x::errors::Error> retrieve_connection(
    const synnax::device::Client &devices,
    const std::string &device_key
) {
    auto [dev, dev_err] = devices.retrieve(device_key);
    if (dev_err)
        return {
            ConnectionConfig(x::json::Parser(x::json::json::object())),
            dev_err,
        };
    auto props = x::json::json(dev.properties);
    const bool secure = props.value("secure", true);
    const std::string protocol = secure ? "https://" : "http://";
    props["base_url"] = protocol + dev.location;
    auto parser = x::json::Parser(props);
    auto conn = ConnectionConfig(parser);
    return {std::move(conn), parser.error()};
}

}
