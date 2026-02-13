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

#include <curl/curl.h>

#include "glog/logging.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"

namespace driver::http::device {
namespace {
struct CurlGlobal {
    CurlGlobal() { curl_global_init(CURL_GLOBAL_DEFAULT); }
    ~CurlGlobal() { curl_global_cleanup(); }
};

void ensure_curl_initialized() { static CurlGlobal instance; }

size_t write_callback(char *ptr, size_t size, size_t nmemb, void *userdata) {
    auto *response = static_cast<std::string *>(userdata);
    response->append(ptr, size * nmemb);
    return size * nmemb;
}

x::errors::Error parse_curl_error(CURLcode code) {
    switch (code) {
        case CURLE_OK:
            return x::errors::NIL;
        case CURLE_COULDNT_CONNECT:
        case CURLE_COULDNT_RESOLVE_HOST:
        case CURLE_COULDNT_RESOLVE_PROXY:
        case CURLE_OPERATION_TIMEDOUT:
            return x::errors::Error(
                http::errors::UNREACHABLE_ERROR, curl_easy_strerror(code)
            );
        default:
            return x::errors::Error(
                http::errors::CLIENT_ERROR, curl_easy_strerror(code)
            );
    }
}
} // namespace

/// @brief internal handle that wraps a pre-configured curl easy handle.
struct Client::Handle {
    CURL *handle = nullptr;
    struct curl_slist *headers = nullptr;
    std::string response_body;
    bool accepts_body = false;

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
        accepts_body(other.accepts_body) {
        other.handle = nullptr;
        other.headers = nullptr;
    }

    Handle &operator=(Handle &&other) noexcept {
        if (this != &other) {
            if (headers != nullptr) curl_slist_free_all(headers);
            if (handle != nullptr) curl_easy_cleanup(handle);
            handle = other.handle;
            headers = other.headers;
            response_body = std::move(other.response_body);
            accepts_body = other.accepts_body;
            other.handle = nullptr;
            other.headers = nullptr;
        }
        return *this;
    }
};

Client::Client(
    ConnectionConfig config,
    const std::vector<RequestConfig> &requests
): config_(std::move(config)) {
    ensure_curl_initialized();
    multi_handle_ = curl_multi_init();
    handles_.reserve(requests.size());

    for (const auto &req: requests) {
        Handle h;
        h.handle = curl_easy_init();
        if (h.handle == nullptr) continue;

        // URL (static per handle).
        CURLU *u = curl_url();
        std::string base = config_.base_url;
        if (!base.empty() && base.back() == '/') base.pop_back();
        if (!req.path.empty()) {
            if (req.path.front() != '/') base += '/';
            base += req.path;
        }
        curl_url_set(u, CURLUPART_URL, base.c_str(), 0);
        for (const auto &[k, v]: req.query_params) {
            const auto param = k + "=" + v;
            curl_url_set(
                u, CURLUPART_QUERY, param.c_str(),
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
            h.handle, CURLOPT_TIMEOUT_MS,
            static_cast<long>(config_.timeout_ms)
        );

        // Write callback (static).
        curl_easy_setopt(h.handle, CURLOPT_WRITEFUNCTION, write_callback);

        // SSL verification (static).
        if (!config_.verify_ssl) {
            curl_easy_setopt(h.handle, CURLOPT_SSL_VERIFYPEER, 0L);
            curl_easy_setopt(h.handle, CURLOPT_SSL_VERIFYHOST, 0L);
        }

        // HTTP method (static per handle).
        h.accepts_body = req.method != Method::GET &&
                         req.method != Method::DELETE;
        switch (req.method) {
            case Method::POST:
                curl_easy_setopt(h.handle, CURLOPT_POST, 1L);
                break;
            case Method::PUT:
                curl_easy_setopt(h.handle, CURLOPT_CUSTOMREQUEST, "PUT");
                break;
            case Method::DELETE:
                curl_easy_setopt(h.handle, CURLOPT_CUSTOMREQUEST, "DELETE");
                break;
            case Method::PATCH:
                curl_easy_setopt(h.handle, CURLOPT_CUSTOMREQUEST, "PATCH");
                break;
            case Method::GET:
                break;
        }

        // Auth headers (static).
        if (config_.auth.type == "bearer") {
            const std::string hdr =
                "Authorization: Bearer " + config_.auth.token;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        } else if (config_.auth.type == "basic") {
            curl_easy_setopt(h.handle, CURLOPT_HTTPAUTH, CURLAUTH_BASIC);
            const std::string userpwd =
                config_.auth.username + ":" + config_.auth.password;
            curl_easy_setopt(h.handle, CURLOPT_USERPWD, userpwd.c_str());
        } else if (config_.auth.type == "api_key") {
            const std::string hdr =
                config_.auth.header + ": " + config_.auth.key;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Connection-level headers (static).
        for (const auto &[k, v]: config_.headers) {
            const std::string hdr = k + ": " + v;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Per-request headers (static).
        for (const auto &[k, v]: req.headers) {
            const std::string hdr = k + ": " + v;
            h.headers = curl_slist_append(h.headers, hdr.c_str());
        }

        // Content-Type header (static).
        if (h.accepts_body)
            h.headers = curl_slist_append(
                h.headers, "Content-Type: application/json"
            );

        if (h.headers != nullptr)
            curl_easy_setopt(h.handle, CURLOPT_HTTPHEADER, h.headers);

        handles_.push_back(std::move(h));
        // Set WRITEDATA after push_back to avoid dangling pointer.
        curl_easy_setopt(
            handles_.back().handle, CURLOPT_WRITEDATA,
            &handles_.back().response_body
        );
    }
}

Client::~Client() {
    if (multi_handle_ != nullptr) curl_multi_cleanup(multi_handle_);
}

std::pair<std::vector<Response>, x::errors::Error>
Client::request(const std::vector<std::string> &bodies) {
    auto *multi = static_cast<CURLM *>(multi_handle_);

    // Set bodies and add handles to multi.
    for (size_t i = 0; i < handles_.size(); i++) {
        auto &h = handles_[i];
        h.response_body.clear();
        if (h.accepts_body) {
            if (i < bodies.size() && !bodies[i].empty()) {
                curl_easy_setopt(
                    h.handle, CURLOPT_POSTFIELDS, bodies[i].c_str()
                );
                curl_easy_setopt(
                    h.handle, CURLOPT_POSTFIELDSIZE, bodies[i].size()
                );
            } else {
                curl_easy_setopt(h.handle, CURLOPT_POSTFIELDS, nullptr);
                curl_easy_setopt(h.handle, CURLOPT_POSTFIELDSIZE, 0L);
            }
        }
        curl_multi_add_handle(multi, h.handle);
    }

    const auto start = x::telem::TimeStamp::now();

    int still_running = 0;
    do {
        const CURLMcode mc = curl_multi_perform(multi, &still_running);
        if (mc != CURLM_OK) break;
        if (still_running > 0)
            curl_multi_wait(multi, nullptr, 0, 1000, nullptr);
    } while (still_running > 0);

    const auto end = x::telem::TimeStamp::now();

    std::vector<Response> responses;
    responses.reserve(handles_.size());

    x::errors::Error first_err = x::errors::NIL;

    CURLMsg *msg;
    int msgs_left;
    while ((msg = curl_multi_info_read(multi, &msgs_left)) != nullptr) {
        if (msg->msg != CURLMSG_DONE) continue;
        if (msg->data.result != CURLE_OK && !first_err)
            first_err = parse_curl_error(msg->data.result);
    }

    for (auto &h: handles_) {
        long status_code = 0;
        curl_easy_getinfo(h.handle, CURLINFO_RESPONSE_CODE, &status_code);
        responses.push_back(Response{
            .status_code = static_cast<int>(status_code),
            .body = std::move(h.response_body),
            .time_range = {start, end},
        });
        curl_multi_remove_handle(multi, h.handle);
    }

    return {std::move(responses), first_err};
}

}
