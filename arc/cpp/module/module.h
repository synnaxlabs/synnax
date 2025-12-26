// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <iomanip>
#include <ostream>
#include <sstream>
#include <string>
#include <vector>

#include "arc/cpp/ir/ir.h"
#include "arc/go/module/arc/go/module/module.pb.h"

namespace arc::module {
struct Module : ir::IR {
    std::vector<uint8_t> wasm;
    std::map<std::string, uint32_t> output_memory_bases;

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

    /// @brief Returns a human-readable string representation of the module.
    [[nodiscard]] std::string to_string() const {
        std::ostringstream ss;
        ss << "Arc Module\n";

        const bool has_content = !functions.empty() || !nodes.empty() ||
                                 !edges.empty() || !strata.empty() ||
                                 !sequences.empty();

        ss << ir::tree_prefix(!has_content) << wasm_summary() << "\n";
        if (has_content) ss << IR::to_string_with_prefix("");
        return ss.str();
    }

    friend std::ostream &operator<<(std::ostream &os, const Module &m) {
        return os << m.to_string();
    }

private:
    /// @brief Returns a summary of the WASM bytecode.
    [[nodiscard]] std::string wasm_summary() const {
        if (wasm.empty()) return "WASM: (none)";
        std::ostringstream ss;
        ss << "WASM: " << wasm.size() << " bytes (sha256: ";
        ss << std::hex << std::setfill('0');
        for (size_t i = 0; i < std::min(static_cast<size_t>(4), wasm.size()); ++i)
            ss << std::setw(2) << static_cast<int>(wasm[i]);
        ss << "...)";
        return ss.str();
    }
};
}
