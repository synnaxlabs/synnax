// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <deque>
#include <future>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <curl/curl.h>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/http/types/types.h"

namespace driver::http {
/// @brief background event loop that drives all HTTP I/O through a single persistent
/// curl multi handle. Task threads submit Request objects and block on futures; the
/// event loop thread owns all curl handles internally.
class Processor {
public:
    Processor();
    ~Processor();

    Processor(const Processor &) = delete;
    Processor &operator=(const Processor &) = delete;

    /// @brief executes requests in parallel and blocks until all complete.
    /// @param requests the requests to execute concurrently.
    /// @returns per-request response/error pairs.
    [[nodiscard]] std::vector<std::pair<Response, x::errors::Error>>
    execute(const std::vector<Request> &requests);

    /// @brief executes a single request and blocks until complete. Zero-allocation fast
    /// path — does not create a vector or delegate to the batch overload.
    /// @param request the request to execute.
    /// @returns the response paired with an error.
    [[nodiscard]] std::pair<Response, x::errors::Error> execute(const Request &request);

private:
    /// @brief internal state for a single in-flight curl transfer.
    struct ActiveTransfer {
        std::promise<std::pair<Response, x::errors::Error>> promise;
        std::string response_body;
        struct curl_slist *headers = nullptr;
        Method method;
        x::telem::TimeStamp start;
    };

    /// @brief a pending request waiting to be picked up by the event loop.
    struct PendingRequest {
        const Request *request;
        std::promise<std::pair<Response, x::errors::Error>> promise;
    };

    /// @brief event loop that processes pending requests and drives curl transfers.
    void run();

    /// @brief builds a Response + Error pair from a completed curl handle.
    static std::pair<Response, x::errors::Error>
    build_result(CURL *handle, CURLcode result_code, ActiveTransfer &t);

    /// @brief creates a curl easy handle from a Request and ActiveTransfer.
    static CURL *create_handle(const Request &req, ActiveTransfer &t);

    CURLM *multi = nullptr;
    std::thread io_thread;
    std::atomic<bool> running{true};
    std::mutex queue_mutex;
    std::deque<PendingRequest> pending;
    std::unordered_map<CURL *, ActiveTransfer> active;
};
}
