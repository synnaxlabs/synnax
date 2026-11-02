// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "driver/opcua/method/method.h"

namespace driver::opcua::method {
std::pair<std::vector<std::optional<std::vector<Argument>>>, x::errors::Error>
read_inputs(
    const connection::Pool::Connection &conn,
    const std::vector<UA_NodeId> &methods
) {
    using Inputs = std::vector<std::optional<std::vector<Argument>>>;
    Inputs out(methods.size());
    auto [browsed, err] = conn.browse(methods);
    if (err) return {Inputs(), err};
    types::ReadRequestBuilder req;
    std::vector<std::size_t> read;
    for (std::size_t i = 0; i < browsed.size(); ++i) {
        if (browsed[i].err) continue;
        const auto &refs = browsed[i].references;
        const auto it = std::ranges::find(
            refs,
            std::string("InputArguments"),
            &types::Reference::browse_name
        );
        if (it == refs.end()) {
            // A cut-short browse can leave out the property of a method that has one.
            if (!browsed[i].truncated) out[i] = std::vector<Argument>{};
            continue;
        }
        req.add_node(it->node.resolve({}).value());
        read.push_back(i);
    }
    if (req.empty()) return {std::move(out), x::errors::NIL};
    auto [res, read_err] = conn.read(req.build());
    if (read_err) return {Inputs(), read_err};
    for (std::size_t k = 0; k < read.size(); ++k) {
        const auto &value = res.get().results[k].value;
        if (!UA_Variant_hasArrayType(&value, &UA_TYPES[UA_TYPES_ARGUMENT])) continue;
        const auto *args = static_cast<const UA_Argument *>(value.data);
        auto &inputs = out[read[k]].emplace();
        for (std::size_t j = 0; j < value.arrayLength; ++j)
            inputs.push_back({
                .name = std::string(
                    reinterpret_cast<const char *>(args[j].name.data),
                    args[j].name.length
                ),
                .data_type = types::NodeId(args[j].dataType),
            });
    }
    return {std::move(out), x::errors::NIL};
}
}
