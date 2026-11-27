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

#include "driver/bus/tasks.h"
#include "driver/icd/retrieve.h"

namespace driver::bus {
Source::Source(
    Acquire acquire,
    std::unique_ptr<Holders::Claim> claim,
    icd::Receiver receiver,
    std::vector<synnax::channel::Channel> bound,
    const bool data_saving_disabled
):
    acquire(std::move(acquire)),
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
    auto [conn, err] = this->acquire();
    this->conn = std::move(conn);
    this->opens = 0;
    return err;
}

x::errors::Error Source::stop() {
    this->conn = nullptr;
    return x::errors::NIL;
}

common::ReadResult Source::read(x::breaker::Breaker &, x::telem::Frame &fr) {
    auto guard = this->conn->lock();
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

std::pair<common::ConfigureResult, x::errors::Error> configure_entries(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const synnax::task::PersistConfig &cfg,
    const std::vector<synnax::icd::ReadEntry> &entries,
    const std::string &device,
    Acquire acquire,
    Readers &readers
) {
    std::set<synnax::icd::Key> icds;
    std::set<synnax::channel::Key> channels;
    for (const auto &entry: entries) {
        icds.insert(entry.icd);
        for (const auto &rm: entry.messages)
            for (const auto &b: rm.bindings)
                if (b.channel != 0) channels.insert(b.channel);
    }
    auto [resources, retrieve_err] = icd::retrieve(*ctx->client, icds, channels);
    if (retrieve_err) return {common::ConfigureResult{}, retrieve_err};
    auto [receiver, receiver_err] = icd::Receiver::create(
        entries,
        resources.icds,
        resources.channels,
        false
    );
    if (receiver_err) return {common::ConfigureResult{}, receiver_err};
    auto [claim, claim_err] = readers.claim(device, task);
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
                    std::move(acquire),
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
