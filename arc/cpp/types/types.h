// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <ostream>

#include "x/cpp/telem/telem.h"

#include "arc/go/types/arc/go/types/types.pb.h"

namespace arc::types {
using ChannelKey = std::uint32_t;

enum class Kind : uint8_t {
    Invalid = 0,
    U8 = 1,
    U16 = 2,
    U32 = 3,
    U64 = 4,
    I8 = 5,
    I16 = 6,
    I32 = 7,
    I64 = 8,
    F32 = 9,
    F64 = 10,
    String = 11,
    TimeStamp = 12,
    TimeSpan = 13,
    Chan = 14,
    Series = 15,
};

struct Type {
    Kind kind = Kind::Invalid;
    std::unique_ptr<Type> elem;

    explicit Type(const arc::v1::types::PBType &pb) {
        this->kind = static_cast<Kind>(pb.kind());
        if (pb.has_elem()) this->elem = std::make_unique<Type>(pb.elem());
    }

    void to_proto(arc::v1::types::PBType *pb) const {
        pb->set_kind(static_cast<arc::v1::types::PBKind>(kind));
        if (elem) elem->to_proto(pb->mutable_elem());
    }

    Type() = default;
    explicit Type(const Kind k): kind(k) {}
    Type(const Kind k, Type elem_type):
        kind(k), elem(std::make_unique<Type>(std::move(elem_type))) {}

    Type(const Type &other): kind(other.kind) {
        if (other.elem) { elem = std::make_unique<Type>(*other.elem); }
    }

    Type &operator=(const Type &other) {
        if (this != &other) {
            kind = other.kind;
            if (other.elem)
                elem = std::make_unique<Type>(*other.elem);
            else
                elem.reset();
        }
        return *this;
    }

    Type(Type &&) = default;
    Type &operator=(Type &&) = default;

    [[nodiscard]] size_t density() const {
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
            case Kind::TimeStamp:
            case Kind::TimeSpan:
                return 8;
            default:
                return 0;
        }
    }

    [[nodiscard]] bool is_valid() const { return kind != Kind::Invalid; }

    [[nodiscard]] telem::DataType telem() const;

    /// @brief Returns the string representation of the type.
    [[nodiscard]] std::string to_string() const {
        switch (kind) {
            case Kind::U8:
                return "u8";
            case Kind::U16:
                return "u16";
            case Kind::U32:
                return "u32";
            case Kind::U64:
                return "u64";
            case Kind::I8:
                return "i8";
            case Kind::I16:
                return "i16";
            case Kind::I32:
                return "i32";
            case Kind::I64:
                return "i64";
            case Kind::F32:
                return "f32";
            case Kind::F64:
                return "f64";
            case Kind::String:
                return "str";
            case Kind::TimeStamp:
                return "timestamp";
            case Kind::TimeSpan:
                return "timespan";
            case Kind::Chan:
                if (elem) return "chan " + elem->to_string();
                return "chan <invalid>";
            case Kind::Series:
                if (elem) return "series " + elem->to_string();
                return "series <invalid>";
            default:
                return "invalid";
        }
    }

    friend std::ostream &operator<<(std::ostream &os, const Type &t) {
        return os << t.to_string();
    }
};
}
