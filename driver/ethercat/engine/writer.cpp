// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <format>
#include <string>

#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
Engine::Writer::Writer(Engine &engine, std::shared_ptr<Output> output):
    engine(engine), output(std::move(output)) {}

Engine::Writer::~Writer() {
    this->engine.close_output(this->output);
}

Engine::Writer::Transaction::Transaction(const Writer &writer):
    engine(writer.engine), lock(writer.engine.write_mu), writer(writer) {}

x::errors::Error Engine::Writer::Transaction::write(
    const size_t pdo_index,
    const x::telem::SampleValue &value
) const {
    const auto &out = *this->writer.output;
    if (pdo_index >= out.entries.size())
        return x::errors::Error(
            errors::PDO_MAPPING_ERROR,
            "output index " + std::to_string(pdo_index) + " is out of range for " +
                std::to_string(out.entries.size()) + " registered entries"
        );
    const auto &entry = out.entries[pdo_index];
    const auto &offset = out.offsets[pdo_index];
    telem::write_pdo_from_value(
        this->engine.write_staging.data() + offset.byte,
        offset.bit,
        entry.bit_length,
        entry.data_type,
        value
    );
    return x::errors::NIL;
}

Engine::Writer::Transaction Engine::Writer::open_tx() const {
    for (auto &slave: this->output->slaves)
        slave.returned.store(false, std::memory_order_relaxed);
    return Transaction(*this);
}

std::string Engine::Writer::warning() const {
    std::string msg;
    for (const auto &slave: this->output->slaves) {
        std::string sentence;
        if (slave.out.load(std::memory_order_relaxed))
            sentence = out_of_op(
                slave.position,
                slave.name,
                slave.silent.load(std::memory_order_relaxed)
            );
        else if (slave.returned.load(std::memory_order_relaxed))
            sentence = std::format(
                "Slave {} ({}) returned and took the last commanded values.",
                slave.position,
                slave.name
            );
        else
            continue;
        msg += (msg.empty() ? "" : " ") + sentence;
    }
    return msg;
}
}
