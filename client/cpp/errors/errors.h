// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"

namespace synnax {
inline xerrors::Error unexpected_missing_error(const std::string &name) {
    return xerrors::Error(
        xerrors::UNEXPECTED,
        "No " + name +
            " returned from server on create. Please report this error to the Synnax team."
    );
}

inline xerrors::Error
not_found_error(const std::string &resource_name, const std::string &query) {
    return xerrors::Error(
        xerrors::NOT_FOUND,
        resource_name + " matching " + query + " not found."
    );
}

inline xerrors::Error
multiple_found_error(const std::string &resource_name, const std::string &query) {
    return xerrors::Error(
        xerrors::MULTIPLE_RESULTS,
        "Multiple " + resource_name + " matching " + query + " not found."
    );
}
}
