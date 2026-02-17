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
/// @brief HTTP client error (configuration issues or unrecognized curl errors).
const x::errors::Error CLIENT_ERROR = CRITICAL_ERROR.sub("client");
/// @brief HTTP response parse error.
const x::errors::Error PARSE_ERROR = CRITICAL_ERROR.sub("parse");
/// @brief HTTP server unreachable (connection refused, DNS failure, or timeout).
const x::errors::Error UNREACHABLE_ERROR = TEMPORARY_ERROR.sub("unreachable");
/// @brief HTTP server error (5xx responses).
const x::errors::Error SERVER_ERROR = TEMPORARY_ERROR.sub("server");
}
