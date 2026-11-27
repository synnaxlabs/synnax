// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <condition_variable>
#include <cstdint>
#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/icd/types.gen.h"
#include "client/cpp/task/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/bus/device.h"
#include "driver/bus/holders.h"
#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/common/write_task.h"
#include "driver/icd/receiver.h"
#include "driver/icd/sender.h"
#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::bus {
/// @brief Acquire acquires the connection to the device of a task, which the tasks on
/// the device share.
/// @returns transport::CONFIG_ERROR or transport::CRITICAL_ERROR when the device cannot
/// open with its settings. A device that cannot be reached now does not fail.
using Acquire = std::function<
    std::pair<std::shared_ptr<transport::Connection>, x::errors::Error>()>;

/// @brief Source reads the ICD messages of the entries of a read task from the
/// connection to one device, taking turns with the other tasks on the device.
class Source final : public common::Source {
    Acquire acquire;
    /// @brief claim holds the device for the task until it is destroyed.
    std::unique_ptr<Holders::Claim> claim;
    icd::Receiver receiver;
    /// @brief bound holds the channel of each binding.
    std::vector<synnax::channel::Channel> bound;
    bool data_saving_disabled;
    /// @brief conn is the connection to the device while the source runs.
    std::shared_ptr<transport::Connection> conn;
    /// @brief opens is the open count of the connection at the last read.
    std::uint64_t opens = 0;

public:
    /// @param acquire acquires the connection to the device.
    /// @param claim the read claim on the device.
    /// @param receiver reads the messages of the entries.
    /// @param bound the channel of each binding of the entries.
    /// @param data_saving_disabled true when the task streams its data without saving
    /// it.
    Source(
        Acquire acquire,
        std::unique_ptr<Holders::Claim> claim,
        icd::Receiver receiver,
        std::vector<synnax::channel::Channel> bound,
        bool data_saving_disabled
    );

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief acquires the connection to the device.
    x::errors::Error
    start(const std::string &cmd_key, x::breaker::Breaker &cancel) override;

    /// @brief releases the connection, which closes when no other task on the device
    /// holds it.
    x::errors::Error stop() override;

    /// @brief reads into fr the messages that arrive in one turn on the connection.
    /// @returns transport::UNREACHABLE_ERROR while the device cannot be reached, and
    /// the error of an open that reopening cannot fix.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;
};

/// @brief Sink sends the ICD messages of the targets of a write task on the connection
/// to one device, taking turns with the other tasks on the device. Targets with a
/// period send from a thread of their own while the sink runs.
class Sink final : public common::Sink {
    Acquire acquire;
    /// @brief mu guards sender and running.
    std::mutex mu;
    std::condition_variable cv;
    icd::Sender sender;
    bool running = false;
    /// @brief conn is the connection to the device while the sink runs.
    std::shared_ptr<transport::Connection> conn;
    /// @brief sends runs the targets with a period.
    std::thread sends;
    /// @brief held_mu guards held.
    std::mutex held_mu;
    /// @brief held is the warning of the last send of the targets with a period.
    common::Warning held;

    /// @brief sends the targets with a period until the sink stops.
    void run();

    /// @brief sends the targets with a period that are due, and holds a warning when
    /// the send fails. Requires mu.
    /// @returns the wait until the next send.
    x::telem::TimeSpan send();

public:
    /// @param acquire acquires the connection to the device.
    /// @param sender sends the messages of the targets.
    /// @param commands the command channels of the targets.
    Sink(
        Acquire acquire,
        icd::Sender sender,
        std::vector<synnax::channel::Key> commands
    );

    /// @brief acquires the connection to the device, and starts the targets with a
    /// period.
    x::errors::Error
    start(const std::string &cmd_key, x::breaker::Breaker &cancel) override;

    /// @brief stops the targets with a period and releases the connection, which closes
    /// when no other task on the device holds it.
    x::errors::Error stop() override;

    /// @brief sends the messages of the commands in frame for each target with no
    /// period, and keeps them for each target with a period, also while the transport
    /// is closed.
    /// @returns the error of the transport, of an encode, or of the framing. A closed
    /// transport returns its error only when a target with no period has a message to
    /// send.
    x::errors::Error write(x::telem::Frame &frame) override;

    /// @brief returns the warning of the last failed send of the targets with a period,
    /// or an empty warning after a send that worked.
    [[nodiscard]] common::Warning held_warning() override;
};

/// @brief configures a read task that reads entries from the device with key device.
/// Its source shares the connection that acquire gives with the other tasks on the
/// device.
/// @param cfg the options of the task.
/// @param readers holds the device for the task.
/// @returns the error of the retrieve of the ICDs of entries or their channels, the
/// error of icd::Receiver::create, and the error of Readers::claim.
std::pair<common::ConfigureResult, x::errors::Error> configure_entries(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const synnax::task::PersistConfig &cfg,
    const std::vector<synnax::icd::ReadEntry> &entries,
    const std::string &device,
    Acquire acquire,
    Readers &readers
);

/// @brief configures a write task that sends targets to a device. Its sink shares the
/// connection that acquire gives with the other tasks on the device.
/// @param cfg the options of the task.
/// @returns the error of the retrieve of the ICDs of targets or their channels, and the
/// error of icd::Sender::create.
std::pair<common::ConfigureResult, x::errors::Error> configure_targets(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const synnax::task::StartConfig &cfg,
    const std::vector<synnax::icd::WriteTarget> &targets,
    Acquire acquire
);

/// @brief configures a read task whose config is a Config, on a device whose properties
/// are Properties.
/// @param readers holds the device for the task.
/// @param acquire gives the Acquire of the connection to the device.
/// @returns VALIDATION when the config names no device or no entry, the error of the
/// retrieve of the device, and the error of configure_entries.
template<typename Config, typename Properties>
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    Readers &readers,
    const std::function<Acquire(const Device<Properties> &)> &acquire
) {
    x::json::Parser parser(task.config);
    const auto cfg = Config::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    if (cfg.entries.empty())
        parser.field_err("entries", "at least one entry is required");
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, err] = retrieve_device<Properties>(ctx->client->devices, cfg.device);
    if (err) return {common::ConfigureResult{}, err};
    return configure_entries(
        ctx,
        task,
        cfg,
        cfg.entries,
        dev.key,
        acquire(dev),
        readers
    );
}

/// @brief configures a write task whose config is a Config, on a device whose
/// properties are Properties.
/// @param acquire gives the Acquire of the connection to the device.
/// @returns VALIDATION when the config names no device or no target, the error of the
/// retrieve of the device, and the error of configure_targets.
template<typename Config, typename Properties>
std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::function<Acquire(const Device<Properties> &)> &acquire
) {
    x::json::Parser parser(task.config);
    const auto cfg = Config::parse(parser);
    if (cfg.device.empty()) parser.field_err("device", "this field is required");
    if (cfg.targets.empty())
        parser.field_err("targets", "at least one target is required");
    if (!parser.ok()) return {common::ConfigureResult{}, parser.error()};
    auto [dev, err] = retrieve_device<Properties>(ctx->client->devices, cfg.device);
    if (err) return {common::ConfigureResult{}, err};
    return configure_targets(ctx, task, cfg, cfg.targets, acquire(dev));
}
}
