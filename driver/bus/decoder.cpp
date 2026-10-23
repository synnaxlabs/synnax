// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <charconv>
#include <cmath>
#include <string>
#include <utility>

#include "x/cpp/strings/strings.h"

#include "driver/bus/decoder.h"
#include "driver/common/common.h"

namespace driver::bus {
namespace {
/// @brief the number of late messages named in a warning before the rest are counted.
constexpr std::size_t MAX_NAMED = 3;

/// @returns the shortest text that reads back as v.
std::string format(const double v) {
    if (std::isnan(v)) return "NaN";
    char buf[32];
    const auto [end, _] = std::to_chars(buf, buf + sizeof(buf), v);
    return {buf, end};
}
}

Decoder::Decoder(const ReadConfig &cfg, x::telem::NowFunc now):
    cfg(cfg), now(std::move(now)), pending(cfg.messages.size()), warnings(this->now) {
    for (std::size_t i = 0; i < cfg.messages.size(); i++)
        this->pending[i].groups.resize(cfg.messages[i].groups.size());
    this->reset();
}

synnax::framer::WriterConfig Decoder::writer_config() const {
    std::vector<synnax::channel::Key> keys;
    for (const auto &m: this->cfg.messages)
        for (const auto &g: m.groups) {
            if (g.index != 0) keys.push_back(g.index);
            for (const auto slot: g.slots)
                keys.push_back(m.channels[slot].key);
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

void Decoder::reset() {
    this->warnings.clear();
    this->restart();
}

void Decoder::restart() {
    const auto now = this->now();
    this->started = now;
    this->received.reset();
    this->matched = now;
    for (auto &p: this->pending)
        p.arrived = now;
}

void Decoder::raw(const std::span<const std::uint8_t> frame) {
    this->received = this->now();
    if (this->cfg.raw == 0) return;
    this->raws.emplace_back(reinterpret_cast<const char *>(frame.data()), frame.size());
}

void Decoder::decode(
    const std::size_t message,
    const std::span<const std::uint8_t> payload,
    const x::telem::TimeStamp time
) {
    const auto &m = this->cfg.messages[message];
    auto &p = this->pending[message];
    const auto now = this->now();
    this->matched = now;
    p.arrived = now;
    if (p.used == p.values.size()) p.values.push_back(m.plan.values());
    auto &values = p.values[p.used];
    if (const auto err = m.plan.decode(payload, values)) {
        this->warn(m.entry.name + ": " + err.data);
        return;
    }
    if (values.invalid() > 0) {
        this->warn(
            m.entry.name + ": " + std::to_string(values.invalid()) +
            " fields did not parse as numbers"
        );
        return;
    }
    for (const auto &group: m.groups)
        for (const auto slot: group.slots) {
            const auto &ch = m.channels[slot];
            if (values.fits(slot, ch.data_type)) continue;
            this->warn(
                m.entry.name + ": channel " + ch.name + " of data type " +
                ch.data_type.name() + " cannot hold " + format(values.get(slot))
            );
            return;
        }
    bool kept = false;
    for (std::size_t g = 0; g < m.groups.size(); g++) {
        const auto &slots = m.groups[g].slots;
        if (!std::ranges::all_of(slots, [&](const auto s) {
                return values.present(s);
            }))
            continue;
        auto &pg = p.groups[g];
        const auto stamp = std::max(time, pg.last + x::telem::TimeSpan(1));
        pg.last = stamp;
        pg.samples.push_back(p.used);
        pg.times.push_back(stamp);
        kept = true;
    }
    if (kept) p.used++;
}

void Decoder::late(const std::size_t message) {
    this->matched = this->now();
    this->warn(
        "late reply to " + this->cfg.messages[message].entry.name + " discarded"
    );
}

void Decoder::warn(const std::string &message) {
    this->warnings.add(message);
}

void Decoder::flush(x::telem::Frame &fr) {
    for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
        auto &p = this->pending[i];
        if (p.used == 0) continue;
        const auto &m = this->cfg.messages[i];
        for (std::size_t g = 0; g < m.groups.size(); g++) {
            auto &pg = p.groups[g];
            const auto n = pg.samples.size();
            if (n == 0) continue;
            for (const auto slot: m.groups[g].slots) {
                const auto &ch = m.channels[slot];
                x::telem::Series series(ch.data_type, n);
                for (const auto k: pg.samples)
                    p.values[k].write(slot, series);
                fr.emplace(ch.key, std::move(series));
            }
            if (m.groups[g].index != 0)
                fr.emplace(m.groups[g].index, x::telem::Series(pg.times));
            pg.samples.clear();
            pg.times.clear();
        }
        p.used = 0;
    }
    if (!this->raws.empty()) {
        fr.emplace(this->cfg.raw, x::telem::Series(this->raws, x::telem::BYTES_T));
        this->raws.clear();
    }
}

std::string Decoder::warning() {
    const auto now = this->now();
    std::vector<std::string> out;
    if (!this->received.has_value() && now - this->started >= SILENCE)
        out.emplace_back("no frames received since start");
    else if (
        this->received.has_value() && *this->received > this->matched &&
        now - this->matched >= SILENCE && now - *this->received < SILENCE
    )
        out.emplace_back("frames arrive, but none match a message in the ICD");
    else {
        std::size_t late = 0;
        for (std::size_t i = 0; i < this->cfg.messages.size(); i++) {
            const auto &m = this->cfg.messages[i];
            if (m.period <= x::telem::TimeSpan::ZERO() ||
                now - this->pending[i].arrived < m.period * MISSED_PERIODS)
                continue;
            if (++late > MAX_NAMED) continue;
            out.push_back(
                m.entry.name + " missed " + std::to_string(MISSED_PERIODS) +
                " periods (expected every " + m.period.to_string() + ")"
            );
        }
        if (late > MAX_NAMED)
            out.push_back(
                std::to_string(late - MAX_NAMED) + " more messages missed " +
                std::to_string(MISSED_PERIODS) + " periods"
            );
    }
    this->warnings.collect(out);
    return x::strings::join(out, "; ");
}
}
