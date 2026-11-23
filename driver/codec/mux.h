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
#include <map>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/codec/bits.h"

namespace driver::codec {
/// @brief Multiplexors holds the multiplexors that select the fields of one binary
/// message, and reports which fields their values select.
class Multiplexors {
public:
    /// @brief Condition selects a field when its multiplexor holds one of values.
    struct Condition {
        /// @brief mux indexes all(), or is -1 for a field that every frame holds.
        int mux = -1;
        std::vector<std::int64_t> values;
    };

    /// @brief Entry is a field that selects which other fields a frame holds.
    struct Entry {
        synnax::icd::FieldKey key;
        std::string name;
        BitRange bits;
        bool is_signed = false;
        Condition condition;
    };

    /// @brief Layout maps the key of each field of a message to the field.
    using Layout = std::map<synnax::icd::FieldKey, const synnax::icd::BinaryField *>;

    /// @brief returns the condition that selects f, and adds the multiplexors that it
    /// reads.
    /// @param fields holds every field of the message of f.
    /// @returns CONFIG_ERROR when a multiplexor does not exist, is a float field,
    /// selects on no values, or is in a cycle of multiplexors.
    std::pair<Condition, x::errors::Error>
    resolve(const synnax::icd::BinaryBaseField &f, const Layout &fields);

    /// @brief reports whether raw, the value of each multiplexor in the order of all(),
    /// selects c.
    [[nodiscard]] bool
    selected(const Condition &c, std::span<const std::int64_t> raw) const;

    /// @brief returns the multiplexors, each after the multiplexor that selects it.
    [[nodiscard]] const std::vector<Entry> &all() const { return this->muxes; }

    /// @brief returns the number of payload bytes that the multiplexors need.
    [[nodiscard]] std::size_t end() const { return this->end_byte; }

private:
    std::vector<Entry> muxes;
    std::size_t end_byte = 0;

    /// @param depth is the number of multiplexors below f in its chain.
    std::pair<Condition, x::errors::Error> resolve(
        const synnax::icd::BinaryBaseField &f,
        const Layout &fields,
        std::size_t depth
    );
};
}
