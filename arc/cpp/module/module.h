//
// Created by Emiliano Bonilla on 11/11/25.
//

#pragma once
#include <string>
#include <vector>

#include "arc/cpp/ir/ir.h"
#include "arc/go/module/arc/go/module/module.pb.h"
#include "x/cpp/xjson/xjson.h"

namespace arc::module {

/// @brief Decodes a base64-encoded string into a vector of bytes
/// @param encoded The base64-encoded string
/// @return A vector of decoded bytes
static std::vector<uint8_t> decode_base64(const std::string &encoded) {
    static const std::string base64_chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
                                            "abcdefghijklmnopqrstuvwxyz"
                                            "0123456789+/";

    std::vector<uint8_t> decoded;
    std::vector<int> temp(4);
    int i = 0;
    int in_len = encoded.size();
    int in_idx = 0;

    while (in_len-- && encoded[in_idx] != '=' &&
           (isalnum(encoded[in_idx]) || encoded[in_idx] == '+' ||
            encoded[in_idx] == '/')) {
        temp[i++] = encoded[in_idx++];
        if (i == 4) {
            for (i = 0; i < 4; i++) {
                temp[i] = base64_chars.find(temp[i]);
            }

            decoded.push_back((temp[0] << 2) + ((temp[1] & 0x30) >> 4));
            decoded.push_back(((temp[1] & 0xf) << 4) + ((temp[2] & 0x3c) >> 2));
            decoded.push_back(((temp[2] & 0x3) << 6) + temp[3]);

            i = 0;
        }
    }

    if (i) {
        for (int j = i; j < 4; j++) {
            temp[j] = 0;
        }

        for (int j = 0; j < 4; j++) {
            temp[j] = base64_chars.find(temp[j]);
        }

        decoded.push_back((temp[0] << 2) + ((temp[1] & 0x30) >> 4));

        if (i > 2) {
            decoded.push_back(((temp[1] & 0xf) << 4) + ((temp[2] & 0x3c) >> 2));
        }
    }

    return decoded;
}

struct Module : ir::IR {
    std::vector<uint8_t> wasm;
    std::map<std::string, uint32_t> output_memory_bases;

    explicit Module(xjson::Parser p): ir::IR(p) {
        auto wasm_str = p.field<std::string>("wasm");
        this->wasm = decode_base64(wasm_str);
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

    explicit Module(const arc::v1::module::PBModule &pb): ir::IR(pb.ir()) {
        this->wasm.assign(pb.wasm().begin(), pb.wasm().end());
        for (const auto &[key, value] : pb.output_memory_bases())
            this->output_memory_bases[key] = value;
    }

    void to_proto(arc::v1::module::PBModule *pb) const {
        ir::IR::to_proto(pb->mutable_ir());
        pb->set_wasm(wasm.data(), wasm.size());
        auto *bases_map = pb->mutable_output_memory_bases();
        for (const auto &[key, value] : output_memory_bases)
            (*bases_map)[key] = value;
    }

    Module() = default;
};
}
