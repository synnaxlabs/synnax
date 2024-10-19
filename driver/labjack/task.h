// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
#pragma once

#include <string>
#include <vector>
#include <map>
#include <utility>
#include <memory>
#include <thread>
#include <set>
#include <condition_variable>

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"


#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/config/config.h"

#include "driver/labjack/reader.h"

namespace labjack {

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    ReaderTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class ReaderTask final : public task::Task {
    public:
        explicit ReaderTask(
                const std::shared_ptr <task::Context> &ctx,
                synnax::Task task,
                std::shared_ptr <labjack::Source> labjack_source,
                std::shared_ptr <pipeline::Source> source,
                synnax::WriterConfig writer_config,
                const breaker::Config breaker_config);

        void exec(task::Command &cmd) override;

        void stop() override;

        void stop(const std::string &cmd_key);

        void start(const std::string &cmd_key);

        std::string name() override { return task.name; }

        static std::unique_ptr <task::Task> configure(
                const std::shared_ptr <task::Context> &ctx,
                const synnax::Task &task
        );

    private:
        std::atomic<bool> running = false;
        std::shared_ptr <task::Context> ctx;
        synnax::Task task;
        pipeline::Acquisition read_pipe;
        std::shared_ptr <labjack::Source> source;
    }; // class ReaderTask

    ///////////////////////////////////////////////////////////////////////////////////
    //                                    WriterTask                                 //
    ///////////////////////////////////////////////////////////////////////////////////
    class WriterTask final : public task::Task{
    public:
        explicit WriterTask(
                const std::shared_ptr <task::Context> &ctx,
                synnax::Task task,
                std::shared_ptr<pipeline::Sink> sink,
                std::shared_ptr<labjack::Sink> labjack_sink,
                std::shared_ptr<pipeline::Source> state_source,
                synnax::WriterConfig writer_config,
                synnax::StreamerConfig streamer_config,
                const breaker::Config breaker_config);

        void exec(task::Command &cmd) override;

        void stop() override;

        void stop(const std::string &cmd_key);

        void start(const std::string &cmd_key);

        std::string name() override { return task.name; }

        static std::unique_ptr <task::Task> configure(
                const std::shared_ptr <task::Context> &ctx,
                const synnax::Task &task
        );
    };

    private:
        std::atomic<bool> running = false;
        std::shared_ptr <task::Context> ctx;
        synnax::Task task;
        pipeline::Control cmd_pipe;
        pipeline::Acquisition state_pipe;
        std::shared_ptr <labjack::Sink> sink;
    }; // class WriterTask
}

// TODO: add a check to see if the libraries are available