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
#include "driver/sequence/plugins/plugins.h"

xerrors::Error plugins::JSON::push_value(lua_State *L, const json &value) {
    if (value.is_null()) lua_pushnil(L);
    else if (value.is_boolean()) lua_pushboolean(L, value.get<bool>());
    else if (value.is_number_integer()) lua_pushinteger(L, value.get<int64_t>());
    else if (value.is_number_float()) lua_pushnumber(L, value.get<double>());
    else if (value.is_string()) lua_pushstring(L, value.get<std::string>().c_str());
    else if (value.is_array()) {
        lua_createtable(L, value.size(), 0);
        int index = 1; // Lua arrays are 1-based
        for (const auto &element: value) {
            auto err = push_value(L, element);
            if (!err.ok()) return err;
            lua_rawseti(L, -2, index++);
        }
    } else if (value.is_object()) {
        lua_createtable(L, 0, value.size());
        for (auto it = value.begin(); it != value.end(); ++it) {
            lua_pushstring(L, it.key().c_str());
            auto err = push_value(L, it.value());
            if (!err.ok()) return err;
            lua_rawset(L, -3);
        }
    } else
        return xerrors::Error("Unsupported JSON type");
    return xerrors::NIL;
}

plugins::JSON::JSON(json source_data): data(std::move(source_data)) {
}


xerrors::Error plugins::JSON::before_all(lua_State *L) {
    if (!data.is_object())
        return xerrors::Error("Root JSON must be an object");
    for (auto it = data.begin(); it != data.end(); ++it) {
        if (auto err = push_value(L, it.value()); err) return err;
        lua_setglobal(L, it.key().c_str());
    }
    return xerrors::NIL;
}
