// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "arc/cpp/types/types.gen.h"

namespace arc::types {

bool Dimensions::operator==(const Dimensions& other) const {
    return length == other.length && mass == other.mass && time == other.time &&
           current == other.current && temperature == other.temperature &&
           angle == other.angle && count == other.count && data == other.data;
}

bool Dimensions::is_zero() const {
    return length == 0 && mass == 0 && time == 0 && current == 0 &&
           temperature == 0 && angle == 0 && count == 0 && data == 0;
}

bool Unit::operator==(const Unit& other) const {
    return dimensions == other.dimensions && scale == other.scale && name == other.name;
}

bool Unit::is_timestamp() const {
    return dimensions.time == 1 && dimensions.length == 0 && dimensions.mass == 0 &&
           dimensions.current == 0 && dimensions.temperature == 0 &&
           dimensions.angle == 0 && dimensions.count == 0 && dimensions.data == 0 &&
           name == "ns" && scale == 1.0;
}

size_t Type::density() const {
    switch (kind) {
        case Kind::U8:
        case Kind::I8:
            return 1;
        case Kind::U16:
        case Kind::I16:
            return 2;
        case Kind::U32:
        case Kind::I32:
        case Kind::F32:
            return 4;
        case Kind::U64:
        case Kind::I64:
        case Kind::F64:
            return 8;
        default:
            return 0;
    }
}

bool Type::is_valid() const { return kind != Kind::Invalid; }

bool Type::is_timestamp() const {
    return kind == Kind::I64 && unit && unit->is_timestamp();
}

std::string Type::to_string() const {
    std::string base;
    switch (kind) {
        case Kind::U8:
            base = "u8";
            break;
        case Kind::U16:
            base = "u16";
            break;
        case Kind::U32:
            base = "u32";
            break;
        case Kind::U64:
            base = "u64";
            break;
        case Kind::I8:
            base = "i8";
            break;
        case Kind::I16:
            base = "i16";
            break;
        case Kind::I32:
            base = "i32";
            break;
        case Kind::I64:
            base = "i64";
            break;
        case Kind::F32:
            base = "f32";
            break;
        case Kind::F64:
            base = "f64";
            break;
        case Kind::String:
            return "str";
        case Kind::Chan:
            if (elem.has_value()) return "chan " + elem->to_string();
            return "chan <invalid>";
        case Kind::Series:
            if (elem.has_value()) return "series " + elem->to_string();
            return "series <invalid>";
        default:
            return "invalid";
    }
    if (unit && !unit->name.empty()) { return base + " " + unit->name; }
    return base;
}

x::telem::DataType Type::telem() const {
    switch (kind) {
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
            if (is_timestamp()) return x::telem::TIMESTAMP_T;
            return x::telem::INT64_T;
        case Kind::F32:
            return x::telem::FLOAT32_T;
        case Kind::F64:
            return x::telem::FLOAT64_T;
        case Kind::String:
            return x::telem::STRING_T;
        default:
            return x::telem::UNKNOWN_T;
    }
}

std::ostream& operator<<(std::ostream& os, const Type& t) { return os << t.to_string(); }

}
