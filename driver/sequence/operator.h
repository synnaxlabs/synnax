//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

#include <memory>

extern "C" {
#include <lua.h>
}

namespace sequence {
class Operator {
public:
    virtual ~Operator() = default;

    virtual freighter::Error before_start(lua_State *L) { return freighter::NIL; }

    virtual freighter::Error after_end(lua_State *L) { return freighter::NIL; }

    virtual freighter::Error before_next(lua_State *L) { return freighter::NIL; }

    virtual freighter::Error after_next(lua_State *L) { return freighter::NIL; }
};

class MultiOperator final : public Operator {
    std::vector<std::shared_ptr<Operator>> ops;

public:
    explicit MultiOperator(std::vector<std::shared_ptr<Operator>> ops): ops(std::move(ops)) {}

    freighter::Error before_start(lua_State *L) override {
        for (auto &op: ops)
            if (auto err = op->before_start(L)) return err;
        return freighter::NIL;
    }

    freighter::Error after_end(lua_State *L) override {
        for (auto &op: ops)
            if (auto err = op->after_end(L)) return err;
        return freighter::NIL;
    }

    freighter::Error before_next(lua_State *L) override {
        for (auto &op: ops)
            if (auto err = op->before_next(L)) return err;
        return freighter::NIL;
    }

    freighter::Error after_next(lua_State *L) override {
        for (auto &op: ops)
            if (auto err = op->after_next(L)) return err;
        return freighter::NIL;
    }
};
};
