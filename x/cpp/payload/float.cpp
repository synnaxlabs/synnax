// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <limits>
#include <string>

#include "x/cpp/payload/float.h"

namespace x::payload {

Float Float::parse(x::json::Parser parser) {
    const auto j = parser.get_json();
    if (j.is_number() && !j.is_boolean()) return Float(j.get<double>());
    if (j.is_string()) {
        const auto s = j.get<std::string>();
        if (s == "NaN") return Float(std::numeric_limits<double>::quiet_NaN());
        if (s == "Infinity") return Float(std::numeric_limits<double>::infinity());
        if (s == "-Infinity") return Float(-std::numeric_limits<double>::infinity());
    }
    parser.field_err("", "cannot parse a float from " + j.dump());
    return Float();
}

x::json::json Float::to_json() const {
    if (std::isnan(this->value)) return "NaN";
    if (std::isinf(this->value)) return this->value > 0 ? "Infinity" : "-Infinity";
    return this->value;
}

}
