// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <include/gtest/gtest.h>

#include "client/cpp/actions/actions.h"
#include "client/cpp/rack/rack.h"

namespace synnax::actions {
/// @brief it should hold the key and the JSON form of each action, in order.
TEST(ActionsTests, testRequest) {
    const std::vector<rack::Action> acts{
        rack::update({.value = {.name = "Rack 2"}}),
        rack::set({.path = {"name"}, .value = "Rack 3"}),
    };
    const auto req = request<grpc::rack::DispatchRequest>(
        rack::Key{12},
        acts,
        rack::to_json
    );
    ASSERT_EQ(req.key(), 12);
    ASSERT_EQ(req.actions_size(), 2);
    ASSERT_EQ(x::json::json::parse(req.actions(0)), rack::to_json(acts[0]));
    ASSERT_EQ(x::json::json::parse(req.actions(1)), rack::to_json(acts[1]));
}
}
