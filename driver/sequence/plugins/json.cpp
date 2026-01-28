// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std.
#include <cstdint>

/// internal.
#include "x/cpp/lua/lua.h"

#include "driver/sequence/plugins/plugins.h"

namespace driver::sequence::plugins {
JSON::JSON(json source_data): data(std::move(source_data)) {}

x::errors::Error JSON::before_all(lua_State *L) {
    return x::lua::set_globals_from_json_object(L, this->data);
}
}