// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <memory>
#include <mutex>
#include <set>
#include <string>
#include <thread>
#include <tuple>
#include <utility>
#include <variant>
#include <vector>

#include "driver/bus/tasks.h"
#include "driver/icd/retrieve.h"
#include "driver/transport/errors.h"

namespace driver::bus {
namespace {
/// @returns true when err came from the transport, which closes on any error.
bool from_transport(const x::errors::Error &err) {
    return err.matches(
        std::vector{
            transport::UNREACHABLE_ERROR,
            transport::CONFIG_ERROR,
            transport::CRITICAL_ERROR,
        }
    );
}
}

Sink::Sink(
    Acquire acquire,
    icd::Sender sender,
    std::vector<synnax::channel::Key> commands
):
    common::Sink(std::move(commands)),
    acquire(std::move(acquire)),
    sender(std::move(sender)) {}

x::errors::Error Sink::start(const std::string &, x::breaker::Breaker &) {
    auto [conn, err] = this->acquire();
    if (err) return err;
    this->conn = std::move(conn);
    {
        std::lock_guard lock(this->held_mu);
        this->held = {};
    }
    if (!this->sender.periodic()) return x::errors::NIL;
    this->running = true;
    this->sends = std::thread(&Sink::run, this);
    return x::errors::NIL;
}

x::errors::Error Sink::stop() {
    {
        std::lock_guard lock(this->mu);
        this->running = false;
    }
    this->cv.notify_all();
    if (this->sends.joinable()) this->sends.join();
    this->conn = nullptr;
    return x::errors::NIL;
}

x::errors::Error Sink::write(x::telem::Frame &frame) {
    std::lock_guard lock(this->mu);
    auto guard = this->conn->lock();
    const auto [t, err] = guard.transport();
    if (err) return err;
    auto [warning, write_err] = this->sender.write(*t, frame);
    if (from_transport(write_err)) guard.fail(write_err);
    this->warn(std::move(warning));
    return write_err;
}

common::Warning Sink::held_warning() {
    std::lock_guard lock(this->held_mu);
    return this->held;
}

void Sink::run() {
    std::unique_lock lock(this->mu);
    while (this->running) {
        const auto wait = this->send();
        this->cv.wait_for(lock, wait.chrono(), [this] { return !this->running; });
    }
}

x::telem::TimeSpan Sink::send() {
    auto guard = this->conn->lock();
    auto [t, err] = guard.transport();
    x::telem::TimeSpan wait;
    if (err)
        // A closed transport sends nothing until the connection may open it again.
        wait = x::telem::TimeSpan(
            this->conn->retry_at() - std::chrono::steady_clock::now()
        );
    else {
        std::tie(wait, err) = this->sender.send(*t);
        if (from_transport(err)) guard.fail(err);
    }
    common::Warning warning;
    if (err) warning = {.message = "Failed to send message", .description = err.data()};
    {
        std::lock_guard lock(this->held_mu);
        if (warning == this->held) return wait;
        this->held = std::move(warning);
    }
    this->warning_changed();
    return wait;
}

std::pair<common::ConfigureResult, x::errors::Error> configure_targets(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const synnax::task::StartConfig &cfg,
    const std::vector<synnax::icd::WriteTarget> &targets,
    Acquire acquire
) {
    std::set<synnax::icd::Key> icds;
    std::set<synnax::channel::Key> commands;
    for (const auto &target: targets) {
        icds.insert(target.icd);
        for (const auto &b: target.message.bindings)
            if (const auto *cb = std::get_if<synnax::icd::ChannelWriteBinding>(&b);
                cb != nullptr && cb->channel != 0)
                commands.insert(cb->channel);
    }
    auto [resources, retrieve_err] = icd::retrieve(*ctx->client, icds, commands);
    if (retrieve_err) return {common::ConfigureResult{}, retrieve_err};
    auto [sender, sender_err] = icd::Sender::create(
        targets,
        resources.icds,
        resources.channels,
        false
    );
    if (sender_err) return {common::ConfigureResult{}, sender_err};
    return {
        common::ConfigureResult{
            .task = std::make_unique<common::WriteTask>(
                task,
                ctx,
                common::unlimited_retry_config(task.name),
                std::make_unique<Sink>(
                    std::move(acquire),
                    std::move(sender),
                    std::vector(commands.begin(), commands.end())
                )
            ),
            .auto_start = cfg.auto_start,
        },
        x::errors::NIL,
    };
}
}
