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

    virtual void bind(lua_State *L) = 0;

    virtual void next() = 0;

    virtual freighter::Error flush() = 0;
};

class MultiOperator final : public Operator {
    std::vector<std::shared_ptr<Operator> > ops;

public:
    explicit MultiOperator(std::vector<std::shared_ptr<Operator> > ops): ops(std::move(ops)) {}

    void bind(lua_State *L) override {
        for (auto &op: this->ops) op->bind(L);
    }

    freighter::Error flush() override {
        for (auto &op: this->ops)
            if (auto err = op->flush()) return err;
        return freighter::NIL;
    }

    void next() override {
        for (auto &op: this->ops) op->next();
    }
};
};
