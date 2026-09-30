// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"

namespace synnax::errors {
/// @brief the base of every license error the Core returns.
const x::errors::Error LICENSE = x::errors::SY.sub("license");
/// @brief the Core refuses requests because no license is activated on it.
const x::errors::Error LICENSE_MISSING = LICENSE.sub("missing");
/// @brief the Core refuses requests because its license has expired.
const x::errors::Error LICENSE_EXPIRED = LICENSE.sub("expired");

/// @brief whether the Core is unreachable or refuses requests until a license is
/// activated. Both clear without a restart, so the caller can wait and retry.
/// @param err the error a request to the Core returned.
/// @returns true if the caller should wait and retry, false if the error is permanent.
inline bool is_temporarily_unavailable(const x::errors::Error &err) {
    return err.matches(freighter::UNREACHABLE) || err.matches(LICENSE_MISSING) ||
           err.matches(LICENSE_EXPIRED);
}

inline x::errors::Error unexpected_missing_error(const std::string &name) {
    return x::errors::Error(
        x::errors::UNEXPECTED,
        "No " + name +
            " returned from server on create. Please report this error to the Synnax team."
    );
}

inline x::errors::Error
not_found_error(const std::string &resource_name, const std::string &query) {
    return x::errors::Error(
        x::errors::NOT_FOUND,
        resource_name + " matching " + query + " not found."
    );
}

inline x::errors::Error
multiple_found_error(const std::string &resource_name, const std::string &query) {
    return x::errors::Error(
        x::errors::MULTIPLE_RESULTS,
        "Multiple " + resource_name + " matching " + query + " not found."
    );
}
}
