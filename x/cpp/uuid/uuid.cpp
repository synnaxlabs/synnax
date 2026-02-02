// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>

#include "x/cpp/uuid/uuid.h"

#include "boost/uuid/nil_generator.hpp"
#include "boost/uuid/random_generator.hpp"
#include "boost/uuid/string_generator.hpp"

namespace x::uuid {

UUID::UUID(): value(boost::uuids::nil_uuid()) {}

UUID::UUID(const boost::uuids::uuid &u): value(u) {}

UUID::UUID(const std::array<std::uint8_t, 16> &bytes) {
    std::copy(bytes.begin(), bytes.end(), value.begin());
}

std::pair<UUID, errors::Error> UUID::parse(const std::string &str) {
    if (str.empty()) return {UUID{}, errors::Error{INVALID, "empty string"}};
    try {
        boost::uuids::string_generator gen;
        return {UUID{gen(str)}, errors::NIL};
    } catch (const std::exception &e) {
        return {UUID{}, errors::Error{INVALID, e.what()}};
    }
}

UUID UUID::parse(json::Parser parser) {
    const auto str = parser.field<std::string>();
    if (str.empty()) return UUID{};
    auto [uuid, err] = parse(str);
    if (err) {
        parser.field_err("", err);
        return UUID{};
    }
    return uuid;
}

bool UUID::is_nil() const {
    return value.is_nil();
}

std::string UUID::to_string() const {
    return boost::uuids::to_string(value);
}

json::json UUID::to_json() const {
    return to_string();
}

const boost::uuids::uuid &UUID::underlying() const {
    return value;
}

const std::uint8_t *UUID::data() const {
    return value.data;
}

bool UUID::operator==(const UUID &other) const {
    return value == other.value;
}

bool UUID::operator!=(const UUID &other) const {
    return value != other.value;
}

bool UUID::operator<(const UUID &other) const {
    return value < other.value;
}

bool UUID::operator>(const UUID &other) const {
    return value > other.value;
}

bool UUID::operator<=(const UUID &other) const {
    return value <= other.value;
}

bool UUID::operator>=(const UUID &other) const {
    return value >= other.value;
}

std::ostream &operator<<(std::ostream &os, const UUID &uuid) {
    os << uuid.to_string();
    return os;
}

}

size_t std::hash<x::uuid::UUID>::operator()(const x::uuid::UUID &uuid) const noexcept {
    return boost::uuids::hash_value(uuid.underlying());
}
