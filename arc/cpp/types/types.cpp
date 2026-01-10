// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "arc/cpp/types/types.h"

namespace arc::types {
x::telem::DataType Type::telem() const {
    switch (this->kind) {
        case Kind::U8:
            return x::telem::UINT8_T;
        case Kind::U16:
            return x::telem::UINT16_T;
        case Kind::U32:
            return x::telem::UINT32_T;
        case Kind::U64:
            return x::telem::UINT64_T;
        case Kind::I8:
            return x::telem::INT8_T;
        case Kind::I16:
            return x::telem::INT16_T;
        case Kind::I32:
            return x::telem::INT32_T;
        case Kind::I64:
            return x::telem::INT64_T;
        case Kind::F32:
            return x::telem::FLOAT32_T;
        case Kind::F64:
            return x::telem::FLOAT64_T;
        case Kind::String:
            return x::telem::STRING_T;
        case Kind::Series:
        case Kind::Chan:
            if (this->elem) return elem->telem();
            [[fallthrough]];
        default:
            return x::telem::UNKNOWN_T;
    }
}
}
