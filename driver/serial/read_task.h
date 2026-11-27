// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "client/cpp/framer/framer.h"
#include "client/cpp/serial/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"

#include "driver/common/common.h"
#include "driver/common/read_task.h"
#include "driver/icd/receiver.h"
#include "driver/serial/serial.h"
#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief READ_TASK_TYPE is the type of the task that reads ICD messages from a serial
/// port.
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief Source reads the ICD messages of the entries of a read task from the port of
/// one device, taking turns with the other tasks on the device.
class Source final : public common::Source {
    std::shared_ptr<transport::Connections> connections;
    /// @brief device is the key of the device.
    std::string device;
    synnax::serial::Properties props;
    icd::Receiver receiver;
    /// @brief bound holds the channel of each binding.
    std::vector<synnax::channel::Channel> bound;
    bool data_saving_disabled;
    /// @brief conn is the connection to the port while the source runs.
    std::shared_ptr<transport::Connection> conn;
    /// @brief opens is the open count of the connection at the last read.
    std::uint64_t opens = 0;

public:
    /// @param connections the connections that the tasks of the integration share.
    /// @param device the key of the device.
    /// @param props the properties of the port of the device.
    /// @param receiver reads the messages of the entries.
    /// @param bound the channel of each binding of the entries.
    /// @param data_saving_disabled true when the task streams its data without saving
    /// it.
    Source(
        std::shared_ptr<transport::Connections> connections,
        std::string device,
        synnax::serial::Properties props,
        icd::Receiver receiver,
        std::vector<synnax::channel::Channel> bound,
        bool data_saving_disabled
    );

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief acquires the port of the device, as acquire does.
    x::errors::Error
    start(const std::string &cmd_key, x::breaker::Breaker &cancel) override;

    /// @brief releases the port, which closes when no other task on the device holds
    /// it.
    x::errors::Error stop() override;

    /// @brief reads into fr the messages that arrive in one turn on the port.
    /// @returns transport::UNREACHABLE_ERROR while the port is missing, in use, or
    /// failed, and the error of an open that reopening cannot fix.
    common::ReadResult read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override;
};

/// @brief configures a serial read task. Its source shares the port of its device with
/// the other tasks on the device through connections.
/// @returns VALIDATION when the config names no device or no entry, the error of the
/// retrieve of the device, its ICDs, or their channels, and the error of
/// icd::Receiver::create.
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<transport::Connections> &connections
);
}
