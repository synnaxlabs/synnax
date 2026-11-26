// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/http/types.gen.h"

#include "driver/common/write_task.h"
#include "driver/http/http.h"
#include "driver/http/processor/processor.h"
#include "driver/http/types/types.h"
#include "driver/icd/writer.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string WRITE_TASK_TYPE = INTEGRATION_NAME + "_write";

/// @brief configuration for an HTTP write task.
struct WriteTaskConfig : ::synnax::http::WriteConfig {
    /// @brief parses a write task config from a Synnax task definition.
    /// @param task the Synnax task definition containing the configuration JSON.
    /// @returns the parsed config paired with an error.
    static std::pair<WriteTaskConfig, x::errors::Error>
    parse(const synnax::task::Task &task);
};

/// @brief sink that sends the commands of each enabled endpoint as HTTP requests whose
/// bodies hold the message of its ICD, one request per sample index.
class WriteTaskSink final : public common::Sink {
    std::shared_ptr<Processor> processor;
    /// @brief the request of each enabled endpoint, parallel to writers.
    std::vector<Request> requests;
    /// @brief the message writer of each enabled endpoint.
    std::vector<driver::icd::Writer> writers;

public:
    /// @param cmd_keys the command channels of the enabled endpoints.
    /// @param processor the shared HTTP processor for executing requests.
    /// @param requests the request of each enabled endpoint, parallel to writers.
    /// @param writers the message writer of each enabled endpoint.
    WriteTaskSink(
        std::vector<synnax::channel::Key> cmd_keys,
        std::shared_ptr<Processor> processor,
        std::vector<Request> requests,
        std::vector<driver::icd::Writer> writers
    );

    /// @brief sends the commands in frame to each enabled endpoint, one request per
    /// row, in order. A failed request drops the later commands of its endpoint in
    /// frame, and the other endpoints still send theirs.
    /// @param frame the frame containing command channel values.
    /// @returns nil, or an error that joins each failure. Its type is that of the first
    /// failure that is not temporary, or TEMPORARY_ERROR when each one is temporary.
    x::errors::Error write(x::telem::Frame &frame) override;
};

/// @brief configures an HTTP write task from a Synnax task definition.
/// @param ctx the task context providing access to the Synnax client.
/// @param task the Synnax task definition containing the configuration JSON.
/// @param processor the shared HTTP processor for executing requests.
/// @returns the configured result paired with an error, which is the error of
/// build_writers when an endpoint cannot send its message.
std::pair<common::ConfigureResult, x::errors::Error> configure_write(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task,
    const std::shared_ptr<Processor> &processor
);

/// @brief builds the message writer of each write endpoint from the message of its ICD.
/// @param client the client that retrieves the ICDs and the command channels of the
/// endpoints.
/// @param endpoints the endpoints to build writers for.
/// @returns one writer per enabled endpoint, in order. Returns CONFIGURATION_ERROR that
/// names the endpoint when it sends no message, when its ICD does not exist, or when
/// driver::icd::Writer rejects its message. Returns the error of a failed retrieve as
/// is.
[[nodiscard]] std::pair<std::vector<driver::icd::Writer>, x::errors::Error>
build_writers(
    const synnax::Synnax &client,
    const std::vector<::synnax::http::WriteEndpoint> &endpoints
);
}
