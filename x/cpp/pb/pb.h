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
#include <utility>

#include "x/cpp/errors/errors.h"

namespace x::pb {

/// @brief converts a protobuf integer to the narrower integer type T.
/// @param v the value to convert.
/// @returns the value as T, and a VALIDATION error when v does not fit in T.
template<typename T, typename S>
std::pair<T, x::errors::Error> narrow(const S v) {
    if (!std::in_range<T>(v))
        return {
            T{},
            x::errors::Error(
                x::errors::VALIDATION,
                std::to_string(v) + " is out of range"
            )
        };
    return {static_cast<T>(v), x::errors::NIL};
}

/// @brief Converts a protobuf repeated field to a container of C++ objects.
/// @tparam CppElem The C++ element type (must have static from_proto method).
/// @tparam Container The destination container type (must have push_back method).
/// @tparam PbContainer The protobuf repeated field type.
/// @param dst The destination container to populate.
/// @param src The source protobuf repeated field.
/// @return x::errors::NIL on success, or the first error encountered.
template<typename CppElem, typename Container, typename PbContainer>
x::errors::Error from_proto_repeated(Container &dst, const PbContainer &src) {
    for (const auto &item: src) {
        auto [v, err] = CppElem::from_proto(item);
        if (err) return err;
        dst.push_back(v);
    }
    return x::errors::NIL;
}

}
