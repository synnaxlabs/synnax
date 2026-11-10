// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstddef>
#include <cstdint>
#include <functional>

#include "x/cpp/errors/errors.h"

#include "driver/modbus/testutil/pseudo_terminal.h"

namespace driver::modbus::testutil {
x::errors::Error PseudoTerminal::open() {
    return x::errors::Error("a serial slave needs a POSIX pseudo-terminal");
}

void PseudoTerminal::close() {}

std::size_t PseudoTerminal::read_frame(std::uint8_t *, const std::size_t) const {
    return 0;
}

void PseudoTerminal::write_damaged(const std::function<void(int)> &) const {}
}
