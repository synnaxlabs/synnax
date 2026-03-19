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

#include "driver/http/errors/errors.h"
#include "driver/http/processor/processor.h"

namespace driver::http {
namespace {
/// @brief maximum time the event loop blocks when no transfers are active. Only affects
/// shutdown latency — new submissions interrupt the wait immediately via
/// curl_multi_wakeup.
const auto IDLE_POLL_TIMEOUT = static_cast<long>(x::telem::SECOND.milliseconds());

/// @brief maximum time the event loop blocks between I/O checks while transfers are
/// in-flight. Caps how long newly submitted requests wait to be picked up when no
/// socket activity occurs.
const auto ACTIVE_POLL_TIMEOUT = static_cast<long>(
    x::telem::MILLISECOND.milliseconds()
);
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
                http::errors::CRITICAL_ERROR.sub(code_str),
                curl_easy_strerror(code)
            );
    }
}

}

std::pair<Response, x::errors::Error>
Processor::build_result(CURL *handle, CURLcode result_code, ActiveTransfer &t) {
    long status_code = 0;
    curl_easy_getinfo(handle, CURLINFO_RESPONSE_CODE, &status_code);
    double total_secs = 0;
    curl_easy_getinfo(handle, CURLINFO_TOTAL_TIME, &total_secs);
    const auto elapsed = x::telem::TimeSpan(static_cast<int64_t>(total_secs * 1e9));
    if (!has_response_body(t.method)) t.response_body.clear();

    x::errors::Error err = x::errors::NIL;
    if (result_code != CURLE_OK) err = parse_curl_error(result_code);
    return {
        Response{
            .status_code = static_cast<int>(status_code),
            .body = std::move(t.response_body),
            .time_range = {t.start, t.start + elapsed},
        },
        err,
    };
}

CURL *Processor::create_handle(const Request &req, ActiveTransfer &t) {
    CURL *handle = curl_easy_init();
    if (handle == nullptr) return nullptr;

    curl_easy_setopt(handle, CURLOPT_URL, req.url.c_str());
    curl_easy_setopt(handle, CURLOPT_NOSIGNAL, 1L);
    curl_easy_setopt(handle, CURLOPT_FOLLOWLOCATION, 1L);

    curl_easy_setopt(
        handle,
        CURLOPT_TIMEOUT_MS,
        static_cast<long>(req.timeout.milliseconds())
    );

    // Write callback — WRITEDATA is set after the transfer is in its final location
    // (see run()) to avoid a dangling pointer from std::move.
    curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, write_callback);

    if (!req.verify_ssl) {
        curl_easy_setopt(handle, CURLOPT_SSL_VERIFYPEER, 0L);
        curl_easy_setopt(handle, CURLOPT_SSL_VERIFYHOST, 0L);
    }

    t.method = req.method;
    if (req.method == Method::HEAD)
        curl_easy_setopt(handle, CURLOPT_NOBODY, 1L);
    else if (req.method == Method::POST)
        curl_easy_setopt(handle, CURLOPT_POST, 1L);
    else if (req.method != Method::GET)
        curl_easy_setopt(handle, CURLOPT_CUSTOMREQUEST, to_string(req.method));

    for (const auto &hdr: req.headers)
        t.headers = curl_slist_append(t.headers, hdr.c_str());

    if (t.headers != nullptr) curl_easy_setopt(handle, CURLOPT_HTTPHEADER, t.headers);

    if (has_request_body(req.method)) {
        if (!req.body.empty()) {
            curl_easy_setopt(handle, CURLOPT_POSTFIELDS, req.body.c_str());
            curl_easy_setopt(
                handle,
                CURLOPT_POSTFIELDSIZE,
                static_cast<long>(req.body.size())
            );
        } else {
            curl_easy_setopt(handle, CURLOPT_POSTFIELDS, nullptr);
            curl_easy_setopt(handle, CURLOPT_POSTFIELDSIZE, 0L);
        }
    }

    return handle;
}

Processor::Processor() {
    ensure_curl_initialized();
    this->multi = curl_multi_init();
    this->io_thread = std::thread([this] { run(); });
}

Processor::~Processor() {
    this->running.store(false);
    curl_multi_wakeup(this->multi);
    if (this->io_thread.joinable()) this->io_thread.join();
    for (auto &[handle, transfer]: this->active) {
        curl_multi_remove_handle(this->multi, handle);
        if (transfer.headers != nullptr) curl_slist_free_all(transfer.headers);
        curl_easy_cleanup(handle);
    }
    curl_multi_cleanup(this->multi);
}

void Processor::run() {
    while (this->running.load()) {
        // Dequeue pending requests under the lock.
        {
            std::lock_guard lock(this->queue_mutex);
            while (!this->pending.empty()) {
                auto &p = this->pending.front();
                ActiveTransfer t;
                t.start = x::telem::TimeStamp::now();
                t.promise = std::move(p.promise);
                CURL *handle = create_handle(*p.request, t);
                if (handle == nullptr) {
                    t.promise.set_value({
                        Response{},
                        x::errors::Error(
                            http::errors::CRITICAL_ERROR,
                            "failed to create curl handle"
                        ),
                    });
                    this->pending.pop_front();
                    continue;
                }
                auto [it, _] = this->active.emplace(handle, std::move(t));
                // Set WRITEDATA after emplace so it points to the response_body at its
                // final address in the map.
                curl_easy_setopt(handle, CURLOPT_WRITEDATA, &it->second.response_body);
                curl_multi_add_handle(this->multi, handle);
                this->pending.pop_front();
            }
        }

        if (this->active.empty()) {
            curl_multi_poll(this->multi, nullptr, 0, IDLE_POLL_TIMEOUT, nullptr);
            continue;
        }

        int still_running = 0;
        const auto mc = curl_multi_perform(this->multi, &still_running);
        if (mc != CURLM_OK) {
            LOG(ERROR) << "[http.processor] curl_multi_perform error: "
                       << curl_multi_strerror(mc);
            const auto err = x::errors::Error(
                http::errors::CRITICAL_ERROR,
                curl_multi_strerror(mc)
            );
            for (auto &[handle, transfer]: this->active) {
                transfer.promise.set_value({Response{}, err});
                curl_multi_remove_handle(this->multi, handle);
                if (transfer.headers != nullptr) curl_slist_free_all(transfer.headers);
                curl_easy_cleanup(handle);
            }
            this->active.clear();
            continue;
        }

        // Check for completed transfers.
        CURLMsg *msg;
        int msgs_left;
        while ((msg = curl_multi_info_read(this->multi, &msgs_left)) != nullptr) {
            if (msg->msg != CURLMSG_DONE) continue;
            CURL *handle = msg->easy_handle;
            auto it = this->active.find(handle);
            if (it == this->active.end()) continue;
            auto &transfer = it->second;
            auto result = build_result(handle, msg->data.result, transfer);
            transfer.promise.set_value(std::move(result));
            curl_multi_remove_handle(this->multi, handle);
            if (transfer.headers != nullptr) curl_slist_free_all(transfer.headers);
            curl_easy_cleanup(handle);
            this->active.erase(it);
        }

        if (!this->active.empty())
            curl_multi_poll(this->multi, nullptr, 0, ACTIVE_POLL_TIMEOUT, nullptr);
    }

    for (auto &[handle, transfer]: this->active) {
        transfer.promise.set_value({
            Response{},
            x::errors::Error(http::errors::CRITICAL_ERROR, "processor shutting down"),
        });
        curl_multi_remove_handle(this->multi, handle);
        if (transfer.headers != nullptr) curl_slist_free_all(transfer.headers);
        curl_easy_cleanup(handle);
    }
    this->active.clear();

    std::lock_guard lock(this->queue_mutex);
    const auto err = x::errors::Error(
        http::errors::CRITICAL_ERROR,
        "processor shutting down"
    );
    while (!this->pending.empty()) {
        this->pending.front().promise.set_value({Response{}, err});
        this->pending.pop_front();
    }
}

std::vector<std::pair<Response, x::errors::Error>>
Processor::execute(const std::vector<Request> &requests) {
    std::vector<std::future<std::pair<Response, x::errors::Error>>> futures;
    futures.reserve(requests.size());

    {
        std::lock_guard lock(this->queue_mutex);
        for (const auto &req: requests) {
            PendingRequest p;
            p.request = &req;
            futures.push_back(p.promise.get_future());
            this->pending.push_back(std::move(p));
        }
    }
    curl_multi_wakeup(this->multi);

    std::vector<std::pair<Response, x::errors::Error>> results;
    results.reserve(requests.size());
    for (auto &f: futures)
        results.push_back(f.get());
    return results;
}

std::pair<Response, x::errors::Error> Processor::execute(const Request &request) {
    std::future<std::pair<Response, x::errors::Error>> fut;
    {
        std::lock_guard lock(this->queue_mutex);
        PendingRequest p;
        p.request = &request;
        fut = p.promise.get_future();
        this->pending.push_back(std::move(p));
    }
    curl_multi_wakeup(this->multi);
    return fut.get();
}

}
