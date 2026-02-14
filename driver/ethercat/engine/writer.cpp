// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
Engine::Writer::Writer(
    Engine &eng,
    const size_t id,
    std::shared_ptr<Registration> registration
):
    engine(eng), id(id), registration(std::move(registration)) {
    std::lock_guard lock(this->engine.write_mu);
    this->refresh_pdos_locked();
}

void Engine::Writer::refresh_pdos_locked() const {
    this->pdos.clear();
    this->pdos.reserve(this->registration->entries.size());
    for (size_t i = 0; i < this->registration->entries.size(); ++i)
        this->pdos.push_back(
            {this->registration->offsets[i],
             this->registration->entries[i].data_type,
             this->registration->entries[i].bit_length}
        );
    this->my_config_gen = this->engine.config_gen.load(std::memory_order_acquire);
}

Engine::Writer::~Writer() {
    this->engine.unregister_writer(this->id);
}

Engine::Writer::Transaction::Transaction(const Writer &writer):
    engine(writer.engine), lock(writer.engine.write_mu), pdos(writer.pdos) {
    if (writer.engine.config_gen.load(std::memory_order_acquire) !=
        writer.my_config_gen)
        writer.refresh_pdos_locked();
}

void Engine::Writer::Transaction::write(
    const size_t pdo_index,
    const x::telem::SampleValue &value
) const {
    if (pdo_index >= this->pdos.size()) return;
    const auto &pdo = this->pdos[pdo_index];
    const size_t required = telem::pdo_required_bytes(pdo.offset.bit, pdo.bit_length);
    if (pdo.offset.byte + required > this->engine.write_staging.size()) return;
    uint8_t *dest = this->engine.write_staging.data() + pdo.offset.byte;
    telem::write_pdo_from_value(
        dest,
        pdo.offset.bit,
        pdo.bit_length,
        pdo.data_type,
        value
    );
}

Engine::Writer::Transaction Engine::Writer::open_tx() const {
    return Transaction(*this);
}

void Engine::Writer::write(
    const size_t pdo_index,
    const x::telem::SampleValue &value
) const {
    this->open_tx().write(pdo_index, value);
}
}
