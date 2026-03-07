// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::http::errors {
/// @brief base critical error for the HTTP integration.
const x::errors::Error CRITICAL_ERROR = driver::errors::CRITICAL_HARDWARE_ERROR.sub(
    "http"
);
/// @brief base temporary error for the HTTP integration.
const x::errors::Error TEMPORARY_ERROR = driver::errors::TEMPORARY_HARDWARE_ERROR.sub(
    "http"
);
/// @brief HTTP server unreachable (connection refused, DNS failure, or timeout).
const x::errors::Error UNREACHABLE_ERROR = TEMPORARY_ERROR.sub("unreachable");

/// @brief classifies an HTTP status code into an error.
/// @param status_code the HTTP response status code.
/// @returns nil for 2xx, TEMPORARY_ERROR for retryable codes (404, 408, 429, 5xx),
/// CRITICAL_ERROR for permanent failures (other 4xx, 3xx, etc.).
[[nodiscard]] x::errors::Error classify_status(int status_code);
}
