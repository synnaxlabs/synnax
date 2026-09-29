// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/check.h"

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
Engine::Writer::Writer(
    Engine &eng,
    const size_t id,
    std::shared_ptr<Registration> registration
):
    engine(eng), id(id), registration(std::move(registration)) {}

void Engine::Writer::refresh_pdos_locked() const {
    this->pdos.clear();
    this->pdos.reserve(this->registration->entries.size());
    for (size_t i = 0; i < this->registration->entries.size(); ++i) {
        const auto &entry = this->registration->entries[i];
        const auto &offset = this->registration->offsets[i];
        auto [plan, err] = telem::plan(entry, offset.bit);
        CHECK(!err) << "open_writer validates every entry: " << err;
        this->pdos.push_back({offset.byte, entry.data_type, std::move(plan)});
    }
    if (!this->pdos.empty()) this->value = this->pdos.front().plan.values();
    this->my_config_gen = this->engine.config_gen.load(std::memory_order_acquire);
}

Engine::Writer::~Writer() {
    this->engine.unregister_writer(this->id);
}

Engine::Writer::Transaction::Transaction(const Writer &writer):
    engine(writer.engine),
    lock(writer.engine.write_mu),
    pdos(writer.pdos),
    value(writer.value) {
    if (writer.engine.config_gen.load(std::memory_order_acquire) !=
        writer.my_config_gen)
        writer.refresh_pdos_locked();
}

x::errors::Error Engine::Writer::Transaction::write(
    const size_t pdo_index,
    const x::telem::SampleValue &value
) const {
    if (pdo_index >= this->pdos.size()) return x::errors::NIL;
    const auto &pdo = this->pdos[pdo_index];
    auto &staging = this->engine.write_staging;
    if (pdo.byte + pdo.plan.length() > staging.size()) return x::errors::NIL;
    const auto casted = pdo.data_type == x::telem::UNKNOWN_T
                          ? value
                          : pdo.data_type.cast(value);
    if (const auto err = this->value.set(0, casted)) return err;
    return pdo.plan.encode(this->value, std::span(staging).subspan(pdo.byte));
}

Engine::Writer::Transaction Engine::Writer::open_tx() const {
    return Transaction(*this);
}

x::errors::Error Engine::Writer::write(
    const size_t pdo_index,
    const x::telem::SampleValue &value
) const {
    return this->open_tx().write(pdo_index, value);
}
}
