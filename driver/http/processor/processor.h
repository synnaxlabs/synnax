// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

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
    struct Impl;
    std::unique_ptr<Impl> impl;
};
}
