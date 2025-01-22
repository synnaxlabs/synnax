//
// Created by Emiliano Bonilla on 1/21/25.
//

#pragma once

extern "C" {
#include <lua.h>
}

#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace sequence {
class Source {
public:
    virtual ~Source() = default;
    virtual freighter::Error bind(lua_State *L) = 0;
};

class MultiSource final: public Source {
    std::vector<std::shared_ptr<Source>> sources;

    explicit MultiSource(std::vector<std::shared_ptr<Source>> sources): sources(std::move(sources)) {}

    freighter::Error bind(lua_State *L) override {
        for (auto &s: this->sources) s->bind(L);
        return freighter::NIL;
    }
};
}