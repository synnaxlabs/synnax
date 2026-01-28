// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/sequence/plugins/plugins.h"

namespace driver::sequence::plugins {
x::errors::Error Time::before_all(lua_State *L) {
    this->start_time = x::telem::TimeStamp(this->now());
    this->elapsed = x::telem::TimeSpan::ZERO();
    this->iteration = 0;
    lua_pushlightuserdata(L, this);
    lua_pushcclosure(
        L,
        [](lua_State *cL) -> int {
            const auto *plug = static_cast<Time *>(
                lua_touserdata(cL, lua_upvalueindex(1))
            );
            const auto start = x::telem::SECOND * luaL_checknumber(cL, 1);
            const auto end = x::telem::SECOND * luaL_checknumber(cL, 2);
            lua_pushboolean(cL, plug->elapsed >= start && plug->elapsed <= end);
            return 1;
        },
        1
    );
    lua_setglobal(L, "elapsed_time_within");
    return x::errors::NIL;
}

x::errors::Error Time::before_next(lua_State *L) {
    this->elapsed = this->now() - this->start_time;
    this->iteration++;
    lua_pushnumber(L, this->elapsed.seconds());
    lua_setglobal(L, "elapsed_time");
    lua_pushinteger(L, this->iteration);
    lua_setglobal(L, "iteration");
    return x::errors::NIL;
}
}