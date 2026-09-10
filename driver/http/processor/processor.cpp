// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <utility>

#include "absl/log/log.h"

#include "driver/http/errors/errors.h"
#include "driver/http/processor/processor.h"

namespace driver::http {
namespace {
/// @brief maximum time the event loop blocks when no transfers are active. Only affects
/// shutdown latency — new submissions interrupt the wait immediately via
/// curl_multi_wakeup.
const auto IDLE_POLL_TIMEOUT = static_cast<long>(x::telem::SECOND.milliseconds());

const auto SKIPPED = x::errors::Error(
    http::errors::SKIPPED_ERROR,
    "not sent, an earlier request to the device was unreachable"
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

Processor::Processor(const x::telem::TimeSpan active_poll_timeout):
    active_poll_timeout_ms(static_cast<int>(active_poll_timeout.milliseconds())) {
    ensure_curl_initialized();
    this->multi = curl_multi_init();
    this->io_thread = std::thread([this] { run(); });
}

Processor::~Processor() {
    this->running.store(false);
    curl_multi_wakeup(this->multi);
    if (this->io_thread.joinable()) this->io_thread.join();
    for (auto &[handle, transfer]: this->active)
        this->finish(handle, transfer);
    curl_multi_cleanup(this->multi);
}

bool Processor::dispatch(PendingRequest &&p, Gate &gate) {
    ActiveTransfer t;
    t.start = x::telem::TimeStamp::now();
    t.promise = std::move(p.promise);
    t.base_url = p.request->base_url;
    CURL *handle = create_handle(*p.request, t);
    if (handle == nullptr) {
        t.promise.set_value({
            Response{},
            x::errors::Error(
                http::errors::CRITICAL_ERROR,
                "failed to create curl handle"
            ),
        });
        return false;
    }
    auto [it, _] = this->active.emplace(handle, std::move(t));
    // Set WRITEDATA after emplace so it points to the response_body at its final
    // address in the map.
    curl_easy_setopt(handle, CURLOPT_WRITEDATA, &it->second.response_body);
    curl_multi_add_handle(this->multi, handle);
    gate.in_flight++;
    return true;
}

void Processor::finish(CURL *handle, const ActiveTransfer &t) {
    curl_multi_remove_handle(this->multi, handle);
    if (t.headers != nullptr) curl_slist_free_all(t.headers);
    curl_easy_cleanup(handle);
}

void Processor::fail_all(const x::errors::Error &err) {
    for (auto &[handle, transfer]: this->active) {
        transfer.promise.set_value({Response{}, err});
        this->finish(handle, transfer);
    }
    this->active.clear();
    for (auto &[_, gate]: this->gates)
        for (auto &p: gate.waiting)
            p.promise.set_value({Response{}, err});
    this->gates.clear();
    this->touched.clear();
}

void Processor::run() {
    while (this->running.load()) {
        // Dequeue pending requests under the lock.
        {
            std::lock_guard lock(this->queue_mutex);
            while (!this->pending.empty()) {
                auto &p = this->pending.front();
                auto [gate_it, opened] = this->gates.try_emplace(p.request->base_url);
                auto &gate = gate_it->second;
                if (opened) gate.cap = p.request->max_concurrent_requests;
                if (gate.in_flight >= gate.cap)
                    gate.waiting.push_back(std::move(p));
                else if (
                    !this->dispatch(std::move(p), gate) && gate.in_flight == 0 &&
                    gate.waiting.empty()
                )
                    this->gates.erase(gate_it);
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
            this->fail_all(
                x::errors::Error(http::errors::CRITICAL_ERROR, curl_multi_strerror(mc))
            );
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
            const bool unreachable = result.second.matches(
                http::errors::UNREACHABLE_ERROR
            );
            transfer.promise.set_value(std::move(result));
            const auto start = transfer.start;
            auto gate_it = this->gates.find(transfer.base_url);
            this->finish(handle, transfer);
            this->active.erase(it);
            if (gate_it == this->gates.end()) continue;

            auto &gate = gate_it->second;
            gate.in_flight--;
            if (unreachable)
                gate.unreachable_start = std::max(gate.unreachable_start, start);
            else
                gate.last_reached = x::telem::TimeStamp::now();
            if (!gate.touched) this->touched.push_back(gate_it);
            gate.touched = true;
        }

        // Skips wait for the whole batch so a success read after a timeout counts.
        for (const auto gate_it: this->touched) {
            auto &gate = gate_it->second;
            gate.touched = false;
            // A dead device fails its queue now, not one timeout at a time.
            if (gate.last_reached < gate.unreachable_start) {
                for (auto &p: gate.waiting)
                    p.promise.set_value({Response{}, SKIPPED});
                gate.waiting.clear();
            }
            gate.unreachable_start = x::telem::TimeStamp(0);
            while (gate.in_flight < gate.cap && !gate.waiting.empty()) {
                auto next = std::move(gate.waiting.front());
                gate.waiting.pop_front();
                this->dispatch(std::move(next), gate);
            }
            if (gate.in_flight == 0 && gate.waiting.empty()) this->gates.erase(gate_it);
        }
        this->touched.clear();

        if (!this->active.empty())
            curl_multi_poll(
                this->multi,
                nullptr,
                0,
                this->active_poll_timeout_ms,
                nullptr
            );
    }

    const auto err = x::errors::Error(
        http::errors::CRITICAL_ERROR,
        "processor shutting down"
    );
    this->fail_all(err);

    std::lock_guard lock(this->queue_mutex);
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
