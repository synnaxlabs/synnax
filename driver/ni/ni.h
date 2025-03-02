// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "client/cpp/synnax.h"

/// internal
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/syscfg/sugared.h"
#include "driver/task/task.h"

namespace ni {
const std::string MAKE = "NI";
const std::string INTEGRATION_NAME = "ni";

/// @brief a factory for instantiating and operating NI data acquisition, control,
/// and device scanning tasks.
class Factory final : public task::Factory {
    /// @brief the daqmx library used to communicate with NI hardware.
    std::shared_ptr<daqmx::SugaredAPI> dmx;
    /// @brief the system configuration library used to get information
    /// about devices.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;

    /// @brief checks whether the factory is healthy and capable of creating tasks.
    /// If not, the factory will automatically send an error back through the
    /// task state and return false.
    [[nodiscard]] bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

public:
    Factory(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg
    );

    /// @brief creates a new NI factory, loading the DAQmx and system configuration
    /// libraries.
    static std::unique_ptr<Factory> create();

    /// @brief implements task::Factory to process task configuration requests.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    /// @brief implements task::Factory to configure initial tasks such as the
    /// device scanner.
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;
};

/// @brief a utility structure for managing the state of national instruments tasks.
struct TaskStateHandler {
    /// @brief the task context used to communicate state changes back to Synnax.
    const std::shared_ptr<task::Context> ctx;
    /// @brief the raw synnax task.
    const synnax::Task task;
    /// @brief the accumulated error in the task state.
    xerrors::Error err;
    /// @brief the wrapped raw task state that will be sent back to Synnax.
    task::State wrapped;

    TaskStateHandler(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : ctx(ctx), task(task) {
        this->wrapped.task = task.key;
        this->wrapped.variant = "success";
    }

    /// @brief register the provided error in the task state. If err is nil, then it
    /// will be ignored, and false will be returned. Otherwise, the provided error will
    /// override any other accumulated errors.
    bool error(const xerrors::Error &err) {
        if (!err) return false;
        this->wrapped.variant = "error";
        this->err = err;
        return true;
    }

    /// @brief sends the provided warning string to the task. If the task is in error
    /// state, the warning will not be sent.
    void send_warning(const std::string &warning) {
        // If theres already an error bound, communicate it instead.
        if (!this->err) {
            this->wrapped.variant = "warning";
            this->wrapped.details["running"] = true;
            this->wrapped.details["message"] = warning;
        }
        this->ctx->set_state(this->wrapped);
    }

    /// @brief sends a start message to the task state, using the provided command
    /// key as part of the state. If an error has been accumulated, then the error
    /// will be sent as part of the state. If the error is nil, then the task will
    /// be marked as running.
    void send_start(const std::string &cmd_key) {
        this->wrapped.key = cmd_key;
        if (!this->err) {
            this->wrapped.details["running"] = true;
            this->wrapped.details["message"] = "Task started successfully";
        } else {
            this->wrapped.variant = "error";
            this->wrapped.details["running"] = false;
            this->wrapped.details["message"] = this->err.message();
        }
        this->ctx->set_state(this->wrapped);
    }

    /// @brief sends a stop message to the task state, using the provided command
    /// key as part of the state. If an error has been accumulated, then the error
    /// will be sent as part of the state. Regardless of the error state, the task
    /// will be marked as not running.
    void send_stop(const std::string &key) {
        this->wrapped.key = key;
        this->wrapped.details["running"] = false;
        if (this->err) {
            this->wrapped.variant = "error";
            this->wrapped.details["message"] = this->err.message();
        } else
            this->wrapped.details["message"] = "Task stopped successfully";
        this->ctx->set_state(this->wrapped);
    }
};
} // namespace ni
