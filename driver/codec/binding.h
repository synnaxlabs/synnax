// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/telem/telem.h"

namespace driver::codec {
/// @brief Binding names a field of a message and the data type of the channel it feeds.
struct Binding {
    /// @brief field is the key of the message field to read.
    synnax::icd::FieldKey field;
    /// @brief data_type is the data type of the channel that the field writes to.
    x::telem::DataType data_type;
};
}
