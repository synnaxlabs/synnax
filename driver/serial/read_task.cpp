// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <set>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/serial/json.gen.h"
#include "x/cpp/json/json.h"

#include "driver/icd/retrieve.h"
#include "driver/serial/read_task.h"

namespace driver::serial {
Source::Source(
    std::shared_ptr<Ports> ports,
    Device device,
    std::unique_ptr<Ports::Claim> claim,
    icd::Receiver receiver,
    std::vector<synnax::channel::Channel> bound,
    const bool data_saving_disabled
):
    ports(std::move(ports)),
    device(std::move(device)),
    claim(std::move(claim)),
    receiver(std::move(receiver)),
    bound(std::move(bound)),
    data_saving_disabled(data_saving_disabled) {}

synnax::framer::WriterConfig Source::writer_config() const {
    return {
        .channels = this->receiver.channels(),
        .mode = common::data_saving_writer_mode(this->data_saving_disabled),
    };
}

std::vector<synnax::channel::Channel> Source::channels() const {
    return this->bound;
}

x::errors::Error Source::start(const std::string &, x::breaker::Breaker &) {
    auto [lease, err] = this->ports->acquire(this->device);
    this->lease = std::move(lease);
    this->opens = 0;
    return err;
}

x::errors::Error Source::stop() {
    this->lease = {};
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &, x::telem::Frame &fr) {
    auto guard = this->lease.conn->lock();
    const auto [t, err] = guard.transport();
    if (err) return {.error = err};
    if (const auto opens = guard.opens(); opens != this->opens) {
        this->opens = opens;
        this->receiver.clear();
    }
    auto res = this->receiver.read(*t, fr);
    if (res.error) guard.fail(res.error);
    return res;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Ports> &ports
) {
    x::json::Parser parser(task.config);
    const auto cfg = synnax::serial::ReadConfig::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    if (cfg.entries.empty())
        parser.field_err("entries", "at least one entry is required");
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, dev_err] = retrieve_device(ctx->client->devices, cfg.device);
    if (dev_err) return {common::ConfigureResult{}, dev_err};
    std::set<synnax::icd::Key> icds;
    std::set<synnax::channel::Key> channels;
    for (const auto &entry: cfg.entries) {
        icds.insert(entry.icd);
        for (const auto &rm: entry.messages)
            for (const auto &b: rm.bindings)
                if (b.channel != 0) channels.insert(b.channel);
    }
    auto [resources, retrieve_err] = icd::retrieve(*ctx->client, icds, channels);
    if (retrieve_err) return {common::ConfigureResult{}, retrieve_err};
    auto [receiver, receiver_err] = icd::Receiver::create(
        cfg.entries,
        resources.icds,
        resources.channels,
        false
    );
    if (receiver_err) return {common::ConfigureResult{}, receiver_err};
    auto [claim, claim_err] = ports->claim_read(dev.props.port, task);
    if (claim_err) return {common::ConfigureResult{}, claim_err};
    std::vector<synnax::channel::Channel> bound;
    bound.reserve(resources.channels.size());
    for (const auto &[_, ch]: resources.channels)
        bound.push_back(ch);
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::ReadTask>(
                task,
                ctx,
                common::unlimited_retry_config(task.name),
                std::make_unique<Source>(
                    ports,
                    std::move(dev),
                    std::move(claim),
                    std::move(receiver),
                    std::move(bound),
                    cfg.data_saving_disabled
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}
}
