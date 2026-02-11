// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/json/json.h"

#include "x/go/spatial/x/go/spatial/spatial.pb.h"

namespace x::spatial {
struct XY {
    float x = 0.0f;
    float y = 0.0f;

    XY() = default;
    XY(const float x, const float y): x(x), y(y) {}

    explicit XY(json::Parser p): x(p.field<float>("x")), y(p.field<float>("y")) {}

    explicit XY(const v1::spatial::PBXY &pb): x(pb.x()), y(pb.y()) {}

    [[nodiscard]] nlohmann::json to_json() const {
        return {{"x", this->x}, {"y", this->y}};
    }

    void to_proto(v1::spatial::PBXY *pb) const {
        pb->set_x(this->x);
        pb->set_y(this->y);
    }
};
}
