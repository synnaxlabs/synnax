// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "x/cpp/xerrors/errors.h"

inline xerrors::Error unexpected_missing(const std::string &name) {
    return xerrors::Error(
        xerrors::UNEXPECTED_ERROR,
        "No " + name +
        " returned from server on create. Please report this error to the Synnax team."
    );
}
