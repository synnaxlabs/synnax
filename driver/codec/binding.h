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

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace driver::codec {
/// @brief Binding names a field of a message and the data type of the channel it feeds.
struct Binding {
    /// @brief field is the key of the message field to read.
    synnax::icd::FieldKey field;
    /// @brief data_type is the data type of the channel that the field writes to.
    x::telem::DataType data_type;
};

/// @brief Failure is a field that the last decode of its message could not decode.
struct Failure {
    /// @brief binding is the position of the field in the bindings of its message.
    std::size_t binding = 0;
    /// @brief missing is true when the payload has no value for the field, and the
    /// field has no default.
    bool missing = false;
    /// @brief error is why the field did not decode. It names the field, and also the
    /// message when the decoder holds more than one.
    x::errors::Error error;
};
}
