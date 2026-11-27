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
#include <map>
#include <span>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/codec/framing/framing.h"
#include "driver/common/common.h"
#include "driver/errors/errors.h"
#include "driver/icd/sender.h"
#include "driver/icd/writer.h"
#include "driver/transport/transport.h"

namespace driver::icd {
std::pair<Sender, x::errors::Error> Sender::create(
    const std::vector<synnax::icd::WriteTarget> &targets,
    const std::vector<synnax::icd::ICD> &icds,
    const std::map<synnax::channel::Key, synnax::channel::Channel> &channels,
    const bool datagram,
    transport::Clock now
) {
    Sender s;
    s.now = std::move(now);
    for (std::size_t i = 0; i < targets.size(); i++) {
        const auto &target = targets[i];
        const auto context = "target " + std::to_string(i) + ": ";
        const auto refuse = [&context](const x::errors::Error &err) {
            return std::pair{Sender{}, x::errors::Error(err, context + err.data)};
        };
        const auto ic = std::ranges::find(icds, target.icd, &synnax::icd::ICD::key);
        if (ic == icds.end())
            return refuse(
                {driver::errors::CONFIGURATION_ERROR,
                 "ICD " + target.icd.to_string() + " does not exist"}
            );
        if (!datagram && !ic->framing.has_value())
            return refuse(
                {driver::errors::CONFIGURATION_ERROR,
                 "ICD " + ic->name + " has no framing to split the stream with"}
            );
        auto [writer, err] = Writer::create(target.message, *ic, channels);
        if (err) return refuse(err);
        Delivery d{.writer = std::move(writer), .period = target.period};
        if (ic->framing.has_value()) {
            auto [framer, framer_err] = codec::framing::create(*ic->framing);
            if (framer_err) return refuse(framer_err);
            d.framer = std::move(framer);
        }
        s.deliveries.push_back(std::move(d));
    }
    return {std::move(s), x::errors::NIL};
}

std::pair<std::span<const std::uint8_t>, x::errors::Error>
Sender::frame(const Delivery &d, const std::span<const std::uint8_t> payload) {
    if (d.framer == nullptr) return {payload, x::errors::NIL};
    this->framed.clear();
    if (auto err = d.framer->encode(payload, this->framed)) return {{}, err};
    return {this->framed, x::errors::NIL};
}

std::pair<common::Warning, x::errors::Error>
Sender::write(transport::Transport &t, const x::telem::Frame &fr) {
    x::errors::Error error;
    x::errors::Error refused;
    for (auto &d: this->deliveries) {
        Writer::Result res;
        if (d.period.has_value() || error)
            res.refused = d.writer.keep(fr);
        else
            res = d.writer.write(fr, [&](const auto payload) {
                const auto [bytes, err] = this->frame(d, payload);
                if (err) return err;
                return t.write(bytes, transport::WRITE_TIMEOUT);
            });
        if (!refused) refused = res.refused;
        if (!error) error = res.error;
    }
    common::Warning warning;
    if (refused)
        warning = {.message = "Skipped invalid command", .description = refused.data};
    return {warning, error};
}

std::pair<x::telem::TimeSpan, x::errors::Error> Sender::send(transport::Transport &t) {
    auto wait = x::telem::TimeSpan::max();
    x::errors::Error error;
    x::errors::Error transport_err;
    const auto now = this->now();
    for (auto &d: this->deliveries) {
        if (!d.period.has_value()) continue;
        const auto period = d.period->chrono();
        if (now >= d.next) {
            if (!transport_err) {
                const auto err = d.writer.encode([&](const auto payload) {
                    const auto [bytes, frame_err] = this->frame(d, payload);
                    if (frame_err) return frame_err;
                    transport_err = t.write(bytes, transport::WRITE_TIMEOUT);
                    return transport_err;
                });
                if (!error) error = err;
            }
            d.next += period;
            if (d.next <= now) d.next = now + period;
        }
        wait = std::min(wait, x::telem::TimeSpan(d.next - now));
    }
    return {wait, transport_err ? transport_err : error};
}
}
