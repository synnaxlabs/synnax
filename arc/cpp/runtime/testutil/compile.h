// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <stdexcept>
#include <string>

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"

#include "arc/cpp/program/program.h"

namespace arc::runtime::testutil {
/// @brief Compiles an Arc program via the Synnax client.
inline arc::program::Program
compile_text(const synnax::Synnax &client, const std::string &source) {
    auto arc = synnax::arc::Arc{
        .name = make_unique_channel_name("test_arc"),
        .mode = synnax::arc::MODE_TEXT
    };
    arc.text.raw = source;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::arc::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    if (!compiled.program.has_value())
        throw std::runtime_error("Compiled arc has no program");
    return *compiled.program;
}
}
