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
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "client/cpp/channel/types.gen.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/frame.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/common.h"
#include "driver/common/write_task.h"
#include "driver/icd/sender.h"
#include "driver/serial/ports.h"
#include "driver/serial/serial.h"
#include "driver/task/task.h"
#include "driver/transport/connection.h"

namespace driver::serial {
/// @brief WRITE_TASK_TYPE is the type of the task that sends ICD messages on a serial
/// port.
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";

/// @brief Sink sends the ICD messages of the targets of a write task on the port of one
/// device, taking turns with the other tasks on the device. Targets with a period send
/// from a thread of their own while the sink runs.
class Sink final : public common::Sink {
    std::shared_ptr<Ports> ports;
    Device device;
    /// @brief mu guards sender and running.
    std::mutex mu;
    std::condition_variable cv;
    icd::Sender sender;
    bool running = false;
    /// @brief conn is the connection to the port while the sink runs.
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
    /// @param ports the ports that the tasks of the integration share.
    /// @param device the device that the task sends to.
    /// @param sender sends the messages of the targets.
    /// @param commands the command channels of the targets.
    Sink(
        std::shared_ptr<Ports> ports,
        Device device,
        icd::Sender sender,
        std::vector<synnax::channel::Key> commands
    );

    /// @brief acquires the port of the device, as Ports::acquire does, and starts the
    /// targets with a period.
    x::errors::Error
    start(const std::string &cmd_key, x::breaker::Breaker &cancel) override;

    /// @brief stops the targets with a period and releases the port, which closes when
    /// no other task on the device holds it.
    x::errors::Error stop() override;

    /// @brief sends the messages of the commands in frame for each target with no
    /// period, and keeps them for each target with a period, also while the port is
    /// closed.
    /// @returns the error of the port, of an encode, or of the framing. A closed port
    /// returns its error only when a target with no period has a message to send.
    x::errors::Error write(x::telem::Frame &frame) override;

    /// @brief returns the warning of the last failed send of the targets with a period,
    /// or an empty warning after a send that worked.
    [[nodiscard]] common::Warning held_warning() override;
};

/// @brief configures a serial write task. Its sink shares the port of its device with
/// the other tasks on the device through ports.
/// @returns VALIDATION when the config names no device or no target, the error of the
/// retrieve of the device, its ICDs, or their channels, and the error of
/// icd::Sender::create.
std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Ports> &ports
);
}
