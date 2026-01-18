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

namespace x::pb {

/// @brief Converts a protobuf repeated field to a container of C++ objects.
/// @tparam CppElem The C++ element type (must have static from_proto method).
/// @tparam Container The destination container type (must have push_back method).
/// @tparam PbContainer The protobuf repeated field type.
/// @param dst The destination container to populate.
/// @param src The source protobuf repeated field.
/// @return x::errors::NIL on success, or the first error encountered.
template <typename CppElem, typename Container, typename PbContainer>
x::errors::Error from_proto_repeated(
    Container& dst,
    const PbContainer& src
) {
    for (const auto& item : src) {
        auto [v, err] = CppElem::from_proto(item);
        if (err) return err;
        dst.push_back(v);
    }
    return x::errors::NIL;
}

}
