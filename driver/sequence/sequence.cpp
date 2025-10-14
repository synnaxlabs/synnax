// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

extern "C" {
#include <lualib.h>
}

/// internal.
#include "driver/sequence/sequence.h"

sequence::Sequence::Sequence(
    const std::shared_ptr<plugins::Plugin> &plugins,
    std::string script
):
    plugins(plugins), L(luaL_newstate()), script(std::move(script)) {
    luaL_openlibs(L.get());
}

sequence::Sequence::~Sequence() {
    if (script_ref != LUA_NOREF) luaL_unref(L.get(), LUA_REGISTRYINDEX, script_ref);
}

[[nodiscard]] xerrors::Error sequence::Sequence::begin() {
    L.reset(luaL_newstate());
    luaL_openlibs(L.get());
    if (auto err = this->compile(); err) return err;
    return this->plugins->before_all(this->L.get());
}

[[nodiscard]] xerrors::Error sequence::Sequence::next() const {
    lua_State *raw_L = L.get();
    if (const auto err = this->plugins->before_next(raw_L)) return err;
    lua_rawgeti(raw_L, LUA_REGISTRYINDEX, script_ref);
    if (lua_pcall(raw_L, 0, 0, 0) != LUA_OK) {
        const char *error_msg = lua_tostring(raw_L, -1);
        lua_pop(raw_L, 1);
        return xerrors::Error(RUNTIME_ERROR, error_msg);
    }
    if (const auto err = this->plugins->after_next(raw_L)) return err;
    return xerrors::NIL;
}

[[nodiscard]] xerrors::Error sequence::Sequence::end() const {
    return this->plugins->after_all(this->L.get());
}

xerrors::Error sequence::Sequence::compile() {
    if (luaL_loadstring(L.get(), this->script.c_str()) != LUA_OK) {
        const char *error_msg = lua_tostring(L.get(), -1);
        lua_pop(L.get(), 1);
        return xerrors::Error(COMPILATION_ERROR, error_msg);
    }
    script_ref = luaL_ref(this->L.get(), LUA_REGISTRYINDEX);
    return xerrors::NIL;
}
