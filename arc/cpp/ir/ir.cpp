// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/ir/ir.h"

#include "x/cpp/telem/telem.h"

namespace arc::ir {

telem::DataType Type::telem() const {
    switch (kind) {
        case TypeKind::U8:
            return ::telem::UINT8_T;
        case TypeKind::U16:
            return ::telem::UINT16_T;
        case TypeKind::U32:
            return ::telem::UINT32_T;
        case TypeKind::U64:
            return ::telem::UINT64_T;
        case TypeKind::I8:
            return ::telem::INT8_T;
        case TypeKind::I16:
            return ::telem::INT16_T;
        case TypeKind::I32:
            return ::telem::INT32_T;
        case TypeKind::I64:
            return ::telem::INT64_T;
        case TypeKind::F32:
            return ::telem::FLOAT32_T;
        case TypeKind::F64:
            return ::telem::FLOAT64_T;
        case TypeKind::String:
            return ::telem::STRING_T;
        case TypeKind::TimeStamp:
            return ::telem::TIMESTAMP_T;
        case TypeKind::Series:
            // For series, get the element type
            if (elem) {
                return elem->telem();
            }
            return ::telem::UNKNOWN_T;
        default:
            return ::telem::UNKNOWN_T;
    }
}

} // namespace arc::ir