//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

/// std. lib.
#include <string>

/// external.
#include "client/cpp/synnax.h"

extern "C" {
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
}

/// internal.
#include "driver/sequence/operator.h"
#include "driver/sequence/source.h"


namespace sequence {
const freighter::Error BASE_ERROR = synnax::BASE_ERROR.sub("sequence");
const freighter::Error COMPILATION_ERROR = BASE_ERROR.sub("compilation");
const freighter::Error RUNTIME_ERROR = BASE_ERROR.sub("runtime");


struct LuaStateDeleter {
    void operator()(lua_State *L) const { if (L) lua_close(L); }
};

class Sequence {
public:
    static std::pair<std::unique_ptr<Sequence>, freighter::Error> create(
        const std::shared_ptr<Operator> &ops,
        const std::shared_ptr<Source> &source,
        std::string script
    ) {
        auto sequence = std::make_unique<Sequence>(ops, source, std::move(script));
        if (auto err = sequence->compile(); err) return {nullptr, err};
        sequence->bind_ops();
        return {std::move(sequence), freighter::NIL};
    }

    ~Sequence() {
        if (script_ref != LUA_NOREF)
            luaL_unref(L.get(), LUA_REGISTRYINDEX, script_ref);
    }

    freighter::Error next() {
        if (auto err = this->source->bind(this->L.get())) return err;
        this->ops->next();
        lua_rawgeti(L.get(), LUA_REGISTRYINDEX, script_ref);
        if (lua_pcall(L.get(), 0, 0, 0) != LUA_OK) {
            const char *error_msg = lua_tostring(L.get(), -1);
            lua_pop(L.get(), 1);
            return freighter::Error(RUNTIME_ERROR, error_msg);
        }
        this->ops->flush();
        return freighter::NIL;
    }

    Sequence(
        const std::shared_ptr<Operator> &ops,
        const std::shared_ptr<Source> &source,
        std::string script
    ) : source(source), ops(ops), script(std::move(script)) {
        L.reset(luaL_newstate());
        luaL_openlibs(L.get());
    }

private:
    /// @brief sink is used to process output operations of the sequence.
    std::shared_ptr<Source> source;
    /// @brief source is used to bind relevant variables to the lua state.
    std::shared_ptr<Operator> ops;
    /// @brief L is the lua program state.
    std::unique_ptr<lua_State, LuaStateDeleter> L;
    /// @brief script is the raw lua script.
    std::string script;
    /// @brief script_ref is the reference to the cache, compiled lua script.
    int script_ref = LUA_NOREF;

    freighter::Error compile() {
        if (luaL_loadstring(L.get(), this->script.c_str()) != LUA_OK) {
            const char *error_msg = lua_tostring(L.get(), -1);
            lua_pop(L.get(), 1);
            return freighter::Error(COMPILATION_ERROR, error_msg);
        }
        script_ref = luaL_ref(this->L.get(), LUA_REGISTRYINDEX);
        return freighter::NIL;
    }

    void bind_ops() const {
        this->ops->bind(this->L.get());
    }


};
}
