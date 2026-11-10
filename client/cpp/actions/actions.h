// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <type_traits>
#include <vector>

#include "x/cpp/json/json.h"

namespace synnax::actions {
/// @brief returns a dispatch request that applies the actions to the record with the
/// given key, in order, in one transaction.
/// @tparam Request the gRPC dispatch request of the record type.
/// @param key the key of the record, in the form that the request holds.
/// @param actions the actions to apply.
/// @param encode returns the JSON form of an action.
template<typename Request, typename Key, typename Action>
Request request(
    const Key &key,
    const std::vector<Action> &actions,
    std::type_identity_t<x::json::json (*)(const Action &)> encode
) {
    Request req;
    req.set_key(key);
    for (const auto &action: actions)
        req.add_actions(encode(action).dump());
    return req;
}
}
