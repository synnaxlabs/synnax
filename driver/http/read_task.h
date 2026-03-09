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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/json/convert.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/read_task.h"
#include "driver/common/sample_clock.h"
#include "driver/http/http.h"
#include "driver/http/processor/processor.h"
#include "driver/http/types/types.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief a single field to extract from an endpoint's JSON response.
struct ReadField {
    /// @brief whether this field is enabled.
    bool enabled = true;
    /// @brief JSON Pointer to the value in the response.
    x::json::json::json_pointer pointer;
    /// @brief Synnax channel key to write the extracted value to.
    synnax::channel::Key channel_key;
    /// @brief if the Synnax channel is a timestamp, the format of the JSON value.
    std::optional<x::json::TimeFormat> time_format;
    /// @brief optional mapping of string values to numbers for enum-style parsing
    /// (e.g., "ON" -> 1, "OFF" -> 0).
    x::json::EnumMap enum_values;
};

/// @brief a single HTTP endpoint to poll.
struct ReadEndpoint {
    /// @brief static request configuration.
    RequestConfig request;
    /// @brief optional static body to send with the request.
    std::string body;
    /// @brief fields to extract from the response.
    std::vector<ReadField> fields;
};

/// @brief a group of fields that share the same index channel and must be
/// written atomically — either all fields succeed or the entire group is
/// skipped.
struct SamplingGroup {
    /// @brief index channel key, or 0 if the fields have no index.
    synnax::channel::Key index_key = 0;
    /// @brief whether the index channel is software-timed (not an explicit
    /// field — needs a timestamp written automatically).
    bool software_timed_index = false;
    /// @brief index of the endpoint this group's fields live on.
    size_t endpoint_index;
    /// @brief indices into ReadEndpoint::fields for the fields in this group.
    std::vector<size_t> field_indices;
};

/// @brief configuration for an HTTP read task.
struct ReadTaskConfig {
    /// @brief key of the device to read from.
    std::string device;
    /// @brief whether to persist data to disk.
    bool data_saving;
    /// @brief whether to auto-start the task.
    bool auto_start;
    /// @brief polling rate (used for both sample_rate and stream_rate).
    x::telem::Rate rate;
    /// @brief endpoints to poll.
    std::vector<ReadEndpoint> endpoints;
    /// @brief sampling groups computed at parse time.
    std::vector<SamplingGroup> groups;
    /// @brief index channels that need software timing, mapped to their endpoint
    /// index. These are index channels referenced by data channels but not
    /// explicitly listed as fields.
    std::map<synnax::channel::Key, int> software_timed_indexes;
    /// @brief mapping of channel keys to their Synnax channel definitions.
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    /// @brief parses a read task config from a Synnax task definition.
    /// @param ctx the task context providing access to the Synnax client.
    /// @param task the Synnax task definition containing the configuration JSON.
    /// @returns the parsed config paired with an error.
    static std::pair<ReadTaskConfig, x::errors::Error>
    parse(const std::shared_ptr<task::Context> &ctx, const synnax::task::Task &task);
};

/// @brief source that polls HTTP endpoints and writes extracted values to a frame.
class ReadTaskSource : public common::Source {
    ReadTaskConfig cfg;
    Processor *processor;
    std::vector<Request> requests;
    common::SoftwareTimedSampleClock sample_clock;
    std::vector<synnax::channel::Channel> chs;
    std::vector<x::json::json> parsed_bodies;

public:
    /// @param cfg the read task configuration.
    /// @param processor the shared HTTP processor for executing requests.
    /// @param requests pre-built requests (one per endpoint).
    ReadTaskSource(
        ReadTaskConfig cfg,
        Processor *processor,
        std::vector<Request> requests
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
