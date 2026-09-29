// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <utility>

#include "x/cpp/strings/strings.h"

#include "driver/bus/decoder.h"
#include "driver/common/common.h"

namespace driver::bus {
Decoder::Decoder(const ReadConfig &cfg): cfg(cfg), pending(cfg.messages.size()) {}

synnax::framer::WriterConfig Decoder::writer_config() const {
    std::vector<synnax::channel::Key> keys;
    for (const auto &m: this->cfg.messages) {
        if (m.index != 0) keys.push_back(m.index);
        for (const auto &ch: m.channels)
            keys.push_back(ch.key);
    }
    if (this->cfg.raw != 0) keys.push_back(this->cfg.raw);
    return {
        .channels = keys,
        .mode = common::data_saving_writer_mode(this->cfg.data_saving_disabled),
    };
}

std::vector<synnax::channel::Channel> Decoder::channels() const {
    std::vector<synnax::channel::Channel> out;
    for (const auto &m: this->cfg.messages)
        out.insert(out.end(), m.channels.begin(), m.channels.end());
    return out;
}

void Decoder::decode(
    const std::size_t message,
    const std::span<const std::uint8_t> payload,
    const x::telem::TimeStamp time
) {
    const auto &m = this->cfg.messages[message];
    auto &p = this->pending[message];
    const auto n = p.times.size();
    if (n == p.values.size()) p.values.push_back(m.plan.values());
    auto &values = p.values[n];
    if (const auto err = m.plan.decode(payload, values)) {
        this->warnings.push_back(m.entry.name + ": " + err.data);
        return;
    }
    if (values.invalid() > 0) {
        this->warnings.push_back(
            m.entry.name + ": " + std::to_string(values.invalid()) +
            " fields did not parse as numbers"
        );
        return;
    }
    for (std::size_t s = 0; s < values.size(); s++)
        if (!values.present(s)) return;
    const auto stamp = std::max(time, p.last + x::telem::TimeSpan(1));
    p.last = stamp;
    p.times.push_back(stamp);
}

void Decoder::raw(const std::span<const std::uint8_t> frame) {
    if (this->cfg.raw == 0) return;
    this->raws.emplace_back(reinterpret_cast<const char *>(frame.data()), frame.size());
}

void Decoder::warn(std::string warning) {
    this->warnings.push_back(std::move(warning));
}

void Decoder::flush(x::telem::Frame &fr) {
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        auto &p = this->pending[i];
        const auto n = p.times.size();
        if (n == 0) continue;
        const auto &m = this->cfg.messages[i];
        for (std::size_t s = 0; s < m.channels.size(); s++) {
            x::telem::Series series(m.channels[s].data_type, n);
            for (std::size_t k = 0; k < n; k++)
                p.values[k].write(s, series);
            fr.emplace(m.channels[s].key, std::move(series));
        }
        if (m.index != 0) fr.emplace(m.index, x::telem::Series(p.times));
        p.times.clear();
    }
    if (!this->raws.empty()) {
        fr.emplace(this->cfg.raw, x::telem::Series(this->raws, x::telem::BYTES_T));
        this->raws.clear();
    }
}

std::string Decoder::warning(const x::telem::TimeStamp now) {
    if (!this->warnings.empty()) {
        this->held = x::strings::join(this->warnings, "; ");
        this->held_at = now;
        this->warnings.clear();
    } else if (now - this->held_at >= WARNING_HOLD)
        this->held.clear();
    return this->held;
}

void Decoder::reset() {
    this->held.clear();
    this->warnings.clear();
}
}
