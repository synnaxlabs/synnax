// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <functional>
#include <string>

#include "x/cpp/errors/errors.h"

namespace driver::modbus::testutil {
/// @brief a pseudo-terminal pair that a test slave serves Modbus RTU on. POSIX only: on
/// Windows, open fails.
class PseudoTerminal {
    int controller_ = -1;
    int terminal_ = -1;
    std::string path_;

public:
    PseudoTerminal() = default;
    ~PseudoTerminal() { this->close(); }

    PseudoTerminal(const PseudoTerminal &) = delete;
    PseudoTerminal &operator=(const PseudoTerminal &) = delete;

    /// @brief opens the pair with both sides in raw mode.
    x::errors::Error open();

    /// @brief closes both sides. Does nothing if the pair is not open.
    void close();

    /// @brief waits up to 100 ms for a frame and reads it. A frame ends at 5 ms of
    /// silence.
    /// @param frame the destination buffer.
    /// @param size the length of the buffer.
    /// @returns the length of the frame, or 0 if no frame arrived.
    std::size_t read_frame(std::uint8_t *frame, std::size_t size) const;

    /// @brief runs write on a descriptor of its own, then sends what it wrote to the
    /// client with the last byte flipped, as line noise would.
    void write_damaged(const std::function<void(int)> &write) const;

    /// @returns the descriptor the slave reads requests from and writes replies to.
    [[nodiscard]] int fd() const { return this->controller_; }

    /// @returns the path a client opens to reach the slave, valid after open.
    [[nodiscard]] std::string path() const { return this->path_; }
};
}
