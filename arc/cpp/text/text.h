// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "arc/go/text/arc/go/text/text.pb.h"

namespace arc::text {

/// @brief Represents the text-based source code representation of an Arc program.
struct Text {
    /// @brief Raw text source code
    std::string raw;

    /// @brief Constructs an empty Text
    Text() = default;

    /// @brief Constructs a Text from a raw string
    /// @param raw_text The raw source code text
    explicit Text(std::string raw_text): raw(std::move(raw_text)) {}

    /// @brief Constructs a Text from its protobuf representation
    /// @param pb The protobuf message
    explicit Text(const v1::text::PBText &pb): raw(pb.raw()) {}

    /// @brief Converts the Text to its protobuf representation
    /// @param pb Pointer to protobuf message to populate
    void to_proto(v1::text::PBText *pb) const { pb->set_raw(raw); }
};
}
