// Copyright 2026 Synnax Labs, Inc.
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
    // 12 and 13 were TimeStamp and TimeSpan, now removed.
    // Timestamps and timespans are represented as i64 with time unit metadata.
    Chan = 14,
    Series = 15,
};

/// @brief Dimensions represents dimension exponents for dimensional analysis.
/// Uses SI base dimensions plus pragmatic extensions for hardware telemetry.
struct Dimensions {
    int8_t length = 0; // meters (m)
    int8_t mass = 0; // kilograms (kg)
    int8_t time = 0; // seconds (s)
    int8_t current = 0; // amperes (A)
    int8_t temperature = 0; // kelvin (K)
    int8_t angle = 0; // radians/degrees
    int8_t count = 0; // samples, items, cycles
    int8_t data = 0; // bits, bytes

    bool operator==(const Dimensions &other) const {
        return length == other.length && mass == other.mass && time == other.time &&
               current == other.current && temperature == other.temperature &&
               angle == other.angle && count == other.count && data == other.data;
    }

    [[nodiscard]] bool is_zero() const {
        return length == 0 && mass == 0 && time == 0 && current == 0 &&
               temperature == 0 && angle == 0 && count == 0 && data == 0;
    }

    explicit Dimensions(const arc::v1::types::PBDimensions &pb):
        length(static_cast<int8_t>(pb.length())),
        mass(static_cast<int8_t>(pb.mass())),
        time(static_cast<int8_t>(pb.time())),
        current(static_cast<int8_t>(pb.current())),
        temperature(static_cast<int8_t>(pb.temperature())),
        angle(static_cast<int8_t>(pb.angle())),
        count(static_cast<int8_t>(pb.count())),
        data(static_cast<int8_t>(pb.data())) {}

    Dimensions() = default;

    void to_proto(arc::v1::types::PBDimensions *pb) const {
        pb->set_length(length);
        pb->set_mass(mass);
        pb->set_time(time);
        pb->set_current(current);
        pb->set_temperature(temperature);
        pb->set_angle(angle);
        pb->set_count(count);
        pb->set_data(data);
    }
};

/// @brief Unit holds unit metadata for numeric types.
struct Unit {
    Dimensions dimensions;
    double scale = 0;
    std::string name;

    bool operator==(const Unit &other) const {
        return dimensions == other.dimensions && scale == other.scale &&
               name == other.name;
    }

    explicit Unit(const arc::v1::types::PBUnit &pb):
        dimensions(pb.dimensions()), scale(pb.scale()), name(pb.name()) {}

    Unit() = default;
    Unit(Dimensions dims, double s, std::string n):
        dimensions(dims), scale(s), name(std::move(n)) {}

    void to_proto(arc::v1::types::PBUnit *pb) const {
        dimensions.to_proto(pb->mutable_dimensions());
        pb->set_scale(scale);
        pb->set_name(name);
    }

    /// @brief Returns true if this is a time unit with nanosecond scale.
    [[nodiscard]] bool is_timestamp() const {
        // Check that time dimension is 1 and all other dimensions are 0
        return dimensions.time == 1 && dimensions.length == 0 && dimensions.mass == 0 &&
               dimensions.current == 0 && dimensions.temperature == 0 &&
               dimensions.angle == 0 && dimensions.count == 0 && dimensions.data == 0 &&
               name == "ns" && scale == 1.0;
    }
};

struct Type {
    Kind kind = Kind::Invalid;
    std::unique_ptr<Type> elem;
    std::unique_ptr<Unit> unit;

    explicit Type(const arc::v1::types::PBType &pb) {
        this->kind = static_cast<Kind>(pb.kind());
        if (pb.has_elem()) this->elem = std::make_unique<Type>(pb.elem());
        if (pb.has_unit()) this->unit = std::make_unique<Unit>(pb.unit());
    }

    void to_proto(arc::v1::types::PBType *pb) const {
        pb->set_kind(static_cast<arc::v1::types::PBKind>(kind));
        if (elem) elem->to_proto(pb->mutable_elem());
        if (unit) unit->to_proto(pb->mutable_unit());
    }

    Type() = default;
    explicit Type(const Kind k): kind(k) {}
    Type(const Kind k, Type elem_type):
        kind(k), elem(std::make_unique<Type>(std::move(elem_type))) {}
    Type(const Kind k, Unit u): kind(k), unit(std::make_unique<Unit>(std::move(u))) {}

    Type(const Type &other): kind(other.kind) {
        if (other.elem) { elem = std::make_unique<Type>(*other.elem); }
        if (other.unit) { unit = std::make_unique<Unit>(*other.unit); }
    }

    Type &operator=(const Type &other) {
        if (this != &other) {
            kind = other.kind;
            if (other.elem)
                elem = std::make_unique<Type>(*other.elem);
            else
                elem.reset();
            if (other.unit)
                unit = std::make_unique<Unit>(*other.unit);
            else
                unit.reset();
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
                return 8;
            case Kind::Invalid:
            case Kind::String:
            case Kind::Chan:
            case Kind::Series:
                return 0;
        }
        return 0;
    }

    [[nodiscard]] bool is_valid() const { return kind != Kind::Invalid; }

    /// @brief Returns true if this type represents a timestamp (i64 with ns unit).
    [[nodiscard]] bool is_timestamp() const {
        return kind == Kind::I64 && unit && unit->is_timestamp();
    }

    [[nodiscard]] telem::DataType telem() const;

    /// @brief Returns the string representation of the type.
    [[nodiscard]] std::string to_string() const {
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
                if (elem) return "chan " + elem->to_string();
                return "chan <invalid>";
            case Kind::Series:
                if (elem) return "series " + elem->to_string();
                return "series <invalid>";
            case Kind::Invalid:
                return "invalid";
        }
        // For numeric types, append unit name if present
        if (unit && !unit->name.empty()) { return base + " " + unit->name; }
        return base;
    }

    friend std::ostream &operator<<(std::ostream &os, const Type &t) {
        return os << t.to_string();
    }
};
}
