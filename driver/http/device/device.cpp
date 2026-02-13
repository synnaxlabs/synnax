// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <mutex>
#include <string>
#include <utility>

#include <curl/curl.h>

#include "glog/logging.h"

#include "driver/http/device/device.h"
#include "driver/http/errors/errors.h"

namespace driver::http::device {
namespace {
std::once_flag curl_init_flag;

void ensure_curl_initialized() {
    std::call_once(curl_init_flag, [] { curl_global_init(CURL_GLOBAL_DEFAULT); });
}

size_t write_callback(char *ptr, size_t size, size_t nmemb, void *userdata) {
    auto *response = static_cast<std::string *>(userdata);
    response->append(ptr, size * nmemb);
    return size * nmemb;
}

std::string build_url(
    const std::string &base_url,
    const std::string &path,
    const std::map<std::string, std::string> &query_params
) {
    std::string url = base_url;
    if (!url.empty() && url.back() == '/') url.pop_back();
    if (!path.empty()) {
        if (path.front() != '/') url += '/';
        url += path;
    }
    if (!query_params.empty()) {
        url += '?';
        bool first = true;
        for (const auto &[k, v]: query_params) {
            if (!first) url += '&';
            url += k + "=" + v;
            first = false;
        }
    }
    return url;
}

x::errors::Error parse_curl_error(CURLcode code) {
    switch (code) {
        case CURLE_OK:
            return x::errors::NIL;
        case CURLE_COULDNT_CONNECT:
        case CURLE_COULDNT_RESOLVE_HOST:
        case CURLE_COULDNT_RESOLVE_PROXY:
        case CURLE_OPERATION_TIMEDOUT:
            return x::errors::Error(http::errors::UNREACHABLE_ERROR, curl_easy_strerror(code));
        default:
            return x::errors::Error(http::errors::CLIENT_ERROR, curl_easy_strerror(code));
    }
}

struct EasyHandle {
    CURL *handle = nullptr;
    struct curl_slist *headers = nullptr;
    std::string response_body;

    EasyHandle() = default;

    ~EasyHandle() {
        if (headers != nullptr) curl_slist_free_all(headers);
        if (handle != nullptr) curl_easy_cleanup(handle);
    }

    EasyHandle(const EasyHandle &) = delete;
    EasyHandle &operator=(const EasyHandle &) = delete;

    EasyHandle(EasyHandle &&other) noexcept:
        handle(other.handle),
        headers(other.headers),
        response_body(std::move(other.response_body)) {
        other.handle = nullptr;
        other.headers = nullptr;
    }

    EasyHandle &operator=(EasyHandle &&other) noexcept {
        if (this != &other) {
            if (headers != nullptr) curl_slist_free_all(headers);
            if (handle != nullptr) curl_easy_cleanup(handle);
            handle = other.handle;
            headers = other.headers;
            response_body = std::move(other.response_body);
            other.handle = nullptr;
            other.headers = nullptr;
        }
        return *this;
    }
};

void apply_auth(CURL *handle, const AuthConfig &auth, struct curl_slist **slist) {
    if (auth.type == "bearer") {
        const std::string hdr = "Authorization: Bearer " + auth.token;
        *slist = curl_slist_append(*slist, hdr.c_str());
    } else if (auth.type == "basic") {
        curl_easy_setopt(handle, CURLOPT_HTTPAUTH, CURLAUTH_BASIC);
        const std::string userpwd = auth.username + ":" + auth.password;
        curl_easy_setopt(handle, CURLOPT_USERPWD, userpwd.c_str());
    } else if (auth.type == "api_key") {
        const std::string hdr = auth.header + ": " + auth.key;
        *slist = curl_slist_append(*slist, hdr.c_str());
    }
}

EasyHandle create_easy_handle(
    const ConnectionConfig &config,
    const Request &req
) {
    EasyHandle eh;
    eh.handle = curl_easy_init();
    if (eh.handle == nullptr) return eh;

    const auto url = build_url(config.base_url, req.path, req.query_params);
    curl_easy_setopt(eh.handle, CURLOPT_URL, url.c_str());
    curl_easy_setopt(
        eh.handle, CURLOPT_TIMEOUT_MS,
        static_cast<long>(config.timeout_ms)
    );
    curl_easy_setopt(eh.handle, CURLOPT_WRITEFUNCTION, write_callback);

    if (!config.verify_ssl) {
        curl_easy_setopt(eh.handle, CURLOPT_SSL_VERIFYPEER, 0L);
        curl_easy_setopt(eh.handle, CURLOPT_SSL_VERIFYHOST, 0L);
    }

    switch (req.method) {
        case Method::POST:
            curl_easy_setopt(eh.handle, CURLOPT_POST, 1L);
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDS, req.body.c_str());
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDSIZE, req.body.size());
            break;
        case Method::PUT:
            curl_easy_setopt(eh.handle, CURLOPT_CUSTOMREQUEST, "PUT");
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDS, req.body.c_str());
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDSIZE, req.body.size());
            break;
        case Method::DELETE:
            curl_easy_setopt(eh.handle, CURLOPT_CUSTOMREQUEST, "DELETE");
            break;
        case Method::PATCH:
            curl_easy_setopt(eh.handle, CURLOPT_CUSTOMREQUEST, "PATCH");
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDS, req.body.c_str());
            curl_easy_setopt(eh.handle, CURLOPT_POSTFIELDSIZE, req.body.size());
            break;
        case Method::GET:
            break;
    }

    apply_auth(eh.handle, config.auth, &eh.headers);

    for (const auto &[k, v]: config.headers) {
        const std::string hdr = k + ": " + v;
        eh.headers = curl_slist_append(eh.headers, hdr.c_str());
    }
    for (const auto &[k, v]: req.headers) {
        const std::string hdr = k + ": " + v;
        eh.headers = curl_slist_append(eh.headers, hdr.c_str());
    }

    if (!req.body.empty()) {
        const std::string ct = "Content-Type: " + req.content_type;
        eh.headers = curl_slist_append(eh.headers, ct.c_str());
    }

    if (eh.headers != nullptr)
        curl_easy_setopt(eh.handle, CURLOPT_HTTPHEADER, eh.headers);

    return eh;
}
} // namespace

Client::Client(ConnectionConfig config): config_(std::move(config)) {
    ensure_curl_initialized();
    multi_handle_ = curl_multi_init();
}

Client::~Client() {
    if (multi_handle_ != nullptr) curl_multi_cleanup(multi_handle_);
}

std::pair<std::vector<Response>, x::errors::Error>
Client::request(const std::vector<Request> &requests) {
    std::lock_guard lock(mu_);

    std::vector<EasyHandle> handles;
    handles.reserve(requests.size());

    auto *multi = static_cast<CURLM *>(multi_handle_);

    for (const auto &req: requests) {
        auto eh = create_easy_handle(config_, req);
        if (eh.handle == nullptr)
            return {{}, x::errors::Error(http::errors::CLIENT_ERROR, "failed to create curl handle")};
        curl_multi_add_handle(multi, eh.handle);
        handles.push_back(std::move(eh));
        curl_easy_setopt(
            handles.back().handle, CURLOPT_WRITEDATA,
            &handles.back().response_body
        );
    }

    const auto start = x::telem::TimeStamp::now();

    int still_running = 0;
    do {
        const CURLMcode mc = curl_multi_perform(multi, &still_running);
        if (mc != CURLM_OK)
            break;
        if (still_running > 0) curl_multi_wait(multi, nullptr, 0, 1000, nullptr);
    } while (still_running > 0);

    const auto end = x::telem::TimeStamp::now();

    std::vector<Response> responses;
    responses.reserve(handles.size());

    x::errors::Error first_err = x::errors::NIL;

    CURLMsg *msg;
    int msgs_left;
    while ((msg = curl_multi_info_read(multi, &msgs_left)) != nullptr) {
        if (msg->msg != CURLMSG_DONE) continue;
        if (msg->data.result != CURLE_OK && !first_err)
            first_err = parse_curl_error(msg->data.result);
    }

    for (auto &eh: handles) {
        long status_code = 0;
        curl_easy_getinfo(eh.handle, CURLINFO_RESPONSE_CODE, &status_code);
        responses.push_back(Response{
            .status_code = static_cast<int>(status_code),
            .body = std::move(eh.response_body),
            .time_range = {start, end},
        });
        curl_multi_remove_handle(multi, eh.handle);
    }

    return {std::move(responses), first_err};
}

std::pair<std::shared_ptr<Client>, x::errors::Error>
Manager::acquire(const ConnectionConfig &config) {
    std::lock_guard lock(mu_);
    auto it = clients_.find(config.base_url);
    if (it != clients_.end()) {
        if (auto existing = it->second.lock()) return {existing, x::errors::NIL};
        clients_.erase(it);
    }
    auto client = std::make_shared<Client>(config);
    clients_[config.base_url] = client;
    return {client, x::errors::NIL};
}
}
