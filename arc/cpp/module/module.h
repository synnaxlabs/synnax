//
// Created by Emiliano Bonilla on 11/11/25.
//

#pragma once
#include <vector>

#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/ir/types.h"

namespace arc::module {
struct Module : ir::IR {
    std::vector<uint8_t> wasm;
    std::map<std::string, uint32_t> output_memory_bases;

    explicit Module(xjson::Parser p): ir::IR(p) {
        this->wasm = p.field<std::vector<uint8_t>>("wasm");
        this->output_memory_bases = p.field<std::map<std::string, uint32_t>>(
            "output_memory_bases"
        );
    }

    [[nodiscard]] nlohmann::json to_json() const {
        auto j = ir::IR::to_json();
        j["wasm"] = wasm;
        j["output_memory_bases"] = output_memory_bases;
        return j;
    }

    Module() = default;
};
}
