// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <span>
#include <utility>
#include <variant>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"
#include "driver/codec/field.h"
#include "driver/codec/mux.h"

namespace driver::codec {
namespace icd = synnax::icd;

std::pair<Multiplexors::Condition, x::errors::Error>
Multiplexors::resolve(const icd::BinaryBaseField &f, const Layout &fields) {
    return this->resolve(f, fields, 0);
}

std::pair<Multiplexors::Condition, x::errors::Error> Multiplexors::resolve(
    const icd::BinaryBaseField &f,
    const Layout &fields,
    const std::size_t depth
) {
    if (!f.multiplexor.has_value()) return {Condition{}, x::errors::NIL};
    if (f.multiplex_values.empty())
        return {
            Condition{},
            x::errors::Error(CONFIG_ERROR, "multiplex values are empty")
        };
    Condition c{.values = {f.multiplex_values.begin(), f.multiplex_values.end()}};
    const auto &key = *f.multiplexor;
    if (const auto it = std::ranges::find(this->muxes, key, &Entry::key);
        it != this->muxes.end()) {
        c.mux = static_cast<int>(it - this->muxes.begin());
        return {std::move(c), x::errors::NIL};
    }
    const auto found = fields.find(key);
    if (found == fields.end())
        return {
            Condition{},
            x::errors::Error(
                CONFIG_ERROR,
                "multiplexor " + key.to_string() + " does not exist"
            ),
        };
    const auto &mux = *found->second;
    const auto &mb = icd::base(mux);
    if (std::holds_alternative<icd::FloatBinaryField>(mux))
        return {
            Condition{},
            x::errors::Error(
                CONFIG_ERROR,
                "multiplexor " + mb.name + " is a float field"
            ),
        };
    // A multiplexor joins muxes only after the ones above it, so a cycle never ends in
    // muxes. A chain longer than the fields is a cycle.
    if (depth >= fields.size())
        return {
            Condition{},
            x::errors::Error(CONFIG_ERROR, "multiplexors form a cycle")
        };
    auto [bits, bits_err] = BitRange::create(mb);
    if (bits_err) return {Condition{}, as_config(bits_err, "multiplexor " + mb.name)};
    auto [above, err] = this->resolve(mb, fields, depth + 1);
    if (err) return {Condition{}, err};
    c.mux = static_cast<int>(this->muxes.size());
    this->muxes.push_back({
        .key = key,
        .name = mb.name,
        .bits = bits,
        .is_signed = std::holds_alternative<icd::IntegerBinaryField>(mux),
        .condition = std::move(above),
    });
    this->end_byte = std::max(this->end_byte, bits.end());
    return {std::move(c), x::errors::NIL};
}

bool Multiplexors::selected(
    const Condition &c,
    const std::span<const std::int64_t> raw
) const {
    for (const Condition *at = &c; at->mux >= 0; at = &this->muxes[at->mux].condition)
        if (std::ranges::find(at->values, raw[at->mux]) == at->values.end())
            return false;
    return true;
}
}
