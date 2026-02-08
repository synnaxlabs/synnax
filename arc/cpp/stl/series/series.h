// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstring>
#include <memory>
#include <stdexcept>

#include "x/cpp/telem/series.h"

#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::series {

class Module : public stl::Module {
    std::shared_ptr<State> series_state;

public:
    explicit Module(std::shared_ptr<State> series_state):
        series_state(std::move(series_state)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override;

    void clear_cycle() override { this->series_state->clear(); }

    void reset() override { this->series_state->clear(); }
};

}
