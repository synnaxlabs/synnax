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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "freighter/cpp/freighter.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/filter/describe.h"

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

namespace details {
inline std::string where_clause(const std::string &where) {
    return where.empty() ? "" : " where " + where;
}
}

/// @returns a NOT_FOUND error for a retrieve of one record that found none.
/// @param name the singular kind of record, such as "range".
/// @param where the description of the filter from x::filter::describe, or empty.
inline x::errors::Error
not_found_error(const std::string &name, const std::string &where) {
    return x::errors::Error(
        x::errors::NOT_FOUND,
        "Found no " + name + details::where_clause(where)
    );
}

/// @returns a MULTIPLE_RESULTS error for a retrieve of one record that found count.
/// @param name the singular kind of record, such as "range".
/// @param where the description of the filter from x::filter::describe, or empty.
inline x::errors::Error multiple_found_error(
    const std::string &name,
    const std::string &where,
    const std::size_t count
) {
    return x::errors::Error(
        x::errors::MULTIPLE_RESULTS,
        "Expected one " + name + details::where_clause(where) + ", but found " +
            std::to_string(count)
    );
}

/// @returns the one record that a retrieve found, or the error of the retrieve.
/// @param name the singular kind of record, such as "range".
/// @param where the filter that selected the records, for the error message.
/// @param retrieved the records and error that the retrieve returned.
/// @returns a NOT_FOUND error if retrieved holds no record, and a MULTIPLE_RESULTS
/// error if it holds more than one.
template<typename T, typename W>
[[nodiscard]] std::pair<T, x::errors::Error> exactly_one(
    const std::string &name,
    const std::optional<W> &where,
    std::pair<std::vector<T>, x::errors::Error> retrieved
) {
    auto &[records, err] = retrieved;
    if (err) return {T{}, err};
    if (records.empty())
        return {T{}, not_found_error(name, x::filter::describe(where))};
    if (records.size() > 1)
        return {
            T{},
            multiple_found_error(name, x::filter::describe(where), records.size())
        };
    return {std::move(records.front()), x::errors::NIL};
}
}
