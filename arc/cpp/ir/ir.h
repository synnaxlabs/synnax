// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "arc/cpp/ir/types.gen.h"
#include "arc/cpp/ir/json.gen.h"
#include "arc/cpp/ir/proto.gen.h"
#include "arc/cpp/ir/format.h"

namespace arc::ir {
constexpr std::string default_output_param = "output";
constexpr std::string default_input_param = "input";
constexpr std::string lhs_input_param = "lhs_input";
constexpr std::string rhs_input_param = "rhs_input";

inline bool operator==(const Handle& lhs, const Handle& rhs) {
    return lhs.node == rhs.node && lhs.param == rhs.param;
}

inline bool operator==(const Edge& lhs, const Edge& rhs) {
    return lhs.source == rhs.source && lhs.target == rhs.target && lhs.kind == rhs.kind;
}
}

template<>
struct std::hash<arc::ir::Handle> {
    size_t operator()(const arc::ir::Handle& h) const noexcept {
        return std::hash<std::string>{}(h.node + h.param);
    }
};

template<>
struct std::hash<arc::ir::Edge> {
    size_t operator()(const arc::ir::Edge& e) const noexcept {
        return std::hash<arc::ir::Handle>{}(e.source) ^
               std::hash<arc::ir::Handle>{}(e.target) << 1 ^
               std::hash<int>{}(static_cast<int>(e.kind)) << 2;
    }
};
