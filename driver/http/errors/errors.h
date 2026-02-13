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
/// @brief HTTP client error (4xx responses or configuration issues).
const x::errors::Error CLIENT_ERROR =
    driver::errors::CRITICAL_HARDWARE_ERROR.sub("http.client");
/// @brief HTTP server error (5xx responses).
const x::errors::Error SERVER_ERROR =
    driver::errors::TEMPORARY_HARDWARE_ERROR.sub("http.server");
/// @brief HTTP server unreachable (connection refused, DNS failure, or timeout).
const x::errors::Error UNREACHABLE_ERROR =
    driver::errors::TEMPORARY_HARDWARE_ERROR.sub("http.unreachable");
/// @brief HTTP response parse error.
const x::errors::Error PARSE_ERROR =
    driver::errors::CRITICAL_HARDWARE_ERROR.sub("http.parse");
}
