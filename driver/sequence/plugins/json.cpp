// Copyright 2025 Synnax Labs, Inc.
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
#include "x/cpp/xlua/xlua.h"

#include "driver/sequence/plugins/plugins.h"

plugins::JSON::JSON(json source_data): data(std::move(source_data)) {}

xerrors::Error plugins::JSON::before_all(lua_State *L) {
    return xlua::set_globals_from_json_object(L, this->data);
}
