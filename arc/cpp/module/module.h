#pragma once

#include <string>
#include <vector>

#include "x/cpp/binary/base64.h"
#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/ir/ir.h"
#include "arc/go/module/arc/go/module/module.pb.h"

namespace arc::module {
struct Module : ir::IR {
    std::vector<uint8_t> wasm;
    std::map<std::string, uint32_t> output_memory_bases;

    explicit Module(xjson::Parser p): IR(p) {
        auto wasm_str = p.field<std::string>("wasm");
        this->wasm = binary::decode_base64(wasm_str);
        this->output_memory_bases = p.field<std::map<std::string, uint32_t>>(
            "output_memory_bases"
        );
    }

    [[nodiscard]] nlohmann::json to_json() const {
        auto j = IR::to_json();
        j["wasm"] = wasm;
        j["output_memory_bases"] = output_memory_bases;
        return j;
    }

    explicit Module(const v1::module::PBModule &pb): IR(pb.ir()) {
        this->wasm.assign(pb.wasm().begin(), pb.wasm().end());
        for (const auto &[key, value]: pb.output_memory_bases())
            this->output_memory_bases[key] = value;
    }

    void to_proto(v1::module::PBModule *pb) const {
        IR::to_proto(pb->mutable_ir());
        pb->set_wasm(wasm.data(), wasm.size());
        auto *bases_map = pb->mutable_output_memory_bases();
        for (const auto &[key, value]: output_memory_bases)
            (*bases_map)[key] = value;
    }

    Module() = default;
};
}
