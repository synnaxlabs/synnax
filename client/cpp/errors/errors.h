// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/auth/auth.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/describe.h"
#include "x/cpp/json/json.h"

#include "arc/cpp/errors/errors.h"

namespace synnax::errors {
/// @brief the base of every license error the Core returns.
const x::errors::Error LICENSE = x::errors::SY.sub("license");
/// @brief the Core refuses requests because no license is activated on it.
const x::errors::Error LICENSE_MISSING = LICENSE.sub("missing");
/// @brief the Core refuses requests because its license has expired.
const x::errors::Error LICENSE_EXPIRED = LICENSE.sub("expired");

/// @brief whether the Core is unreachable or did not answer in time.
/// @param err the error a request to the Core returned.
inline bool is_unreachable(const x::errors::Error &err) {
    return err.matches(freighter::UNREACHABLE) || err.matches(freighter::TIMEOUT);
}

/// @brief whether the Core is unreachable, did not answer in time, or refuses requests
/// until a license is activated. These clear without a restart, so the caller can wait
/// and retry.
/// @param err the error a request to the Core returned.
/// @returns true if the caller should wait and retry, false if the error is permanent.
inline bool is_temporarily_unavailable(const x::errors::Error &err) {
    return is_unreachable(err) || err.matches(LICENSE_MISSING) ||
           err.matches(LICENSE_EXPIRED);
}

/// @brief how a caller handles an error from the Core.
enum class Kind {
    /// @brief the Core or the path to it fails for now. Wait, then retry. A timeout or
    /// a broken stream can follow a request that took effect.
    TRANSIENT,
    /// @brief the Core reports that it stopped while it handled the request, so the
    /// request may have taken effect. Wait, then resend only a request that is safe to
    /// send twice.
    INTERRUPTED,
    /// @brief the Core refused the request. The same request fails again.
    REJECTED,
    /// @brief a defect in the Driver or the Core. Fail loud.
    BUG,
};

/// @brief the Core stopped while it handled the request.
// TODO(SY-6843): use freighter::INTERRUPTED once it is on main.
const x::errors::Error INTERRUPTED("freighter.interrupted", "Interrupted");

/// @brief the kind of each error type. The first match wins, so a sub-type comes
/// before its parent. A type that matches no row is a BUG.
const std::vector<std::pair<x::errors::Error, Kind>> KINDS = {
    {INTERRUPTED, Kind::INTERRUPTED},
    {freighter::UNREACHABLE, Kind::TRANSIENT},
    {freighter::TIMEOUT, Kind::TRANSIENT},
    {freighter::TRANSPORT, Kind::TRANSIENT},
    {freighter::CANCELLED, Kind::TRANSIENT},
    {freighter::STREAM_CLOSED, Kind::TRANSIENT},
    {freighter::ERR_EOF, Kind::TRANSIENT},
    {LICENSE_MISSING, Kind::TRANSIENT},
    {LICENSE_EXPIRED, Kind::TRANSIENT},
    {auth::ERR_INVALID_TOKEN, Kind::TRANSIENT},
    {auth::ERR_EXPIRED_TOKEN, Kind::TRANSIENT},
    {auth::ERR_SESSION_ENDED, Kind::TRANSIENT},
    // gRPC sends it for a message over the size limit, which fails again on a retry.
    {freighter::RESOURCE_EXHAUSTED, Kind::REJECTED},
    {LICENSE, Kind::REJECTED},
    {auth::ERR, Kind::REJECTED},
    {x::errors::VALIDATION, Kind::REJECTED},
    {x::errors::QUERY, Kind::REJECTED},
    {x::errors::CONTROL, Kind::REJECTED},
    {x::errors::NOT_SUPPORTED, Kind::REJECTED},
    {::arc::errors::BASE, Kind::REJECTED},
};

/// @brief returns how a caller handles err.
/// @param err a non-nil error that a request to the Core returned.
inline Kind classify(const x::errors::Error &err) {
    for (const auto &[type, kind]: KINDS)
        if (err.matches(type)) return kind;
    return Kind::BUG;
}

inline x::errors::Error unexpected_missing_error(const std::string &name) {
    return x::errors::Error(
        x::errors::UNEXPECTED,
        "No " + name +
            " returned from server on create. Please report this error to the Synnax team."
    );
}

/// @brief returns the error for a query for one record that sets a limit or a cursor,
/// since a page could hide a second match.
inline x::errors::Error paged_one_error() {
    return x::errors::Error(
        x::errors::VALIDATION,
        "a query for one record takes no limit or cursor, since a page could hide a "
        "second match"
    );
}

inline x::errors::Error
not_found_error(const std::string &resource_name, const std::string &query) {
    return x::errors::Error(
        x::errors::NOT_FOUND,
        resource_name + " matching " + query + " not found."
    );
}

/// @brief checks that a retrieve of one record found exactly one.
/// @param name the name of the record type, such as "range".
/// @param where the JSON form of the filter that the retrieve sent. The errors
/// describe it.
/// @param search_term the search term that the retrieve sent, or an empty string. The
/// errors name it.
/// @param count the number of records that the retrieve found.
/// @returns NIL when count is 1, NOT_FOUND when it is 0, or MULTIPLE_RESULTS.
inline x::errors::Error check_one(
    const std::string &name,
    const x::json::json &where,
    const std::string &search_term,
    const std::size_t count
) {
    if (count == 1) return x::errors::NIL;
    const auto description = x::filter::describe(where);
    std::string clause = description.empty() ? "" : " where " + description;
    if (!search_term.empty())
        clause = " matching " + x::json::json(search_term).dump() + clause;
    if (count == 0)
        return x::errors::Error(x::errors::NOT_FOUND, "Found no " + name + clause);
    return x::errors::Error(
        x::errors::MULTIPLE_RESULTS,
        "Expected one " + name + clause + ", but found " + std::to_string(count)
    );
}
}
