// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/http/types.gen.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/http/device/device.h"
#include "driver/http/http.h"
#include "driver/http/processor/processor.h"
#include "driver/http/types/types.h"
#include "driver/icd/reader.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief configuration for an HTTP read task.
struct ReadTaskConfig : ::synnax::http::ReadConfig {
    /// @brief parses a read task config from a Synnax task definition. The config
    /// holds only its enabled endpoints.
    /// @returns VALIDATION when the task has no device, no enabled endpoint, or an
    /// enabled endpoint with an invalid request.
    static std::pair<ReadTaskConfig, x::errors::Error>
    parse(const synnax::task::Task &task);
};

/// @brief Readers reads the responses of the endpoints of a read task.
struct Readers {
    /// @brief endpoints holds the reader of each endpoint, in endpoint order.
    std::vector<driver::icd::Reader> endpoints;
    /// @brief channels holds the channel of each binding, by key.
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;
    /// @brief pointers holds the JSON pointer of the field of each binding of a JSON
    /// message, by the key of its channel.
    std::map<synnax::channel::Key, std::string> pointers;
};

/// @brief source that polls HTTP endpoints and writes extracted values to a frame.
class ReadTaskSource : public common::Source {
    ReadTaskConfig cfg;
    std::shared_ptr<Processor> processor;
    std::vector<Request> requests;
    Readers readers;
    /// @brief keys holds each channel that the readers write.
    std::vector<synnax::channel::Key> keys;
    common::SoftwareTimedSampleClock sample_clock;

public:
    /// @param cfg the read task configuration.
    /// @param processor the shared HTTP processor for executing requests.
    /// @param requests pre-built requests (one per endpoint).
    /// @param readers the readers of the responses (one per endpoint).
    ReadTaskSource(
        ReadTaskConfig cfg,
        std::shared_ptr<Processor> processor,
        std::vector<Request> requests,
        Readers readers
    );

    /// @brief returns the writer configuration for the task.
    /// @returns the writer configuration for the task.
    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    /// @brief returns the channels used in this task.
    /// @returns the channels used in this task.
    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    /// @brief reads data from the task.
    /// @param breaker the breaker used to stop the read.
    /// @param fr the frame to write the data to.
    /// @returns the read result.
    common::ReadResult
    read(x::breaker::Breaker &breaker, x::telem::Frame &frame) override;
};

/// @brief builds the static request for each read endpoint.
/// @param conn the resolved device connection.
/// @param endpoints the endpoints to build requests for.
/// @returns one request per endpoint, in order. An endpoint whose method carries no
/// request body drops the body it stores, so a value left behind by a method change
/// never reaches the wire or sets a content type.
[[nodiscard]] std::vector<Request> build_requests(
    const device::ConnectionConfig &conn,
    const std::vector<::synnax::http::ReadEndpoint> &endpoints
);

/// @brief builds the message reader of each read endpoint from the messages of its ICD.
/// @param client the client that retrieves the ICDs and the channels of the endpoints.
/// @param endpoints the endpoints to build readers for.
/// @returns CONFIGURATION_ERROR that names the endpoint when it reads no messages, when
/// its ICD does not exist, or when driver::icd::Reader rejects its messages, and that
/// names both endpoints when two of them write to one channel (an index channel
/// included). Returns the error of a failed retrieve as is.
[[nodiscard]] std::pair<Readers, x::errors::Error> build_readers(
    const synnax::Synnax &client,
    const std::vector<::synnax::http::ReadEndpoint> &endpoints
);

/// @brief configures an HTTP read task from a Synnax task definition.
/// @param ctx the task context providing access to the Synnax client.
/// @param task the Synnax task definition containing the configuration JSON.
/// @param processor the shared HTTP processor for executing requests.
/// @returns the configured result paired with an error.
std::pair<common::ConfigureResult, x::errors::Error> configure_read(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Processor> &processor
);
}
