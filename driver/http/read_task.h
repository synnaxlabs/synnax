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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/json/convert.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"

#include "driver/common/read_task.h"
#include "driver/http/device/device.h"
#include "driver/http/http.h"
#include "driver/task/task.h"

namespace driver::http {
const std::string READ_TASK_TYPE = INTEGRATION_NAME + "_read";

/// @brief optional timestamp extraction info for an index channel.
struct TimeInfo {
    /// @brief JSON Pointer to the timestamp in the response body.
    x::json::json::json_pointer pointer;
    /// @brief format of the timestamp value.
    x::json::TimeFormat format;

    TimeInfo(x::json::json::json_pointer pointer, x::json::TimeFormat format):
        pointer(std::move(pointer)), format(format) {}

    explicit TimeInfo(x::json::Parser &parser):
        pointer(parser.field<std::string>("pointer")) {
        auto [fmt, err] = x::json::parse_time_format(
            parser.field<std::string>("format")
        );
        if (err) parser.field_err("format", err.message());
        format = fmt;
    }
};

/// @brief a single field to extract from an endpoint's JSON response.
struct ReadField {
    /// @brief JSON Pointer to the value in the response.
    x::json::json::json_pointer pointer;
    /// @brief Synnax channel key to write the extracted value to.
    synnax::channel::Key channel_key;
    /// @brief if the Synnax channel is a timestamp, the format of the JSON value.
    std::optional<x::json::TimeFormat> time_format;
    /// @brief optional timestamp source for this field's index channel.
    std::optional<TimeInfo> time_info;
};

/// @brief a single HTTP endpoint to poll.
struct ReadEndpoint {
    /// @brief static request configuration.
    device::RequestConfig request;
    /// @brief optional static body to send with the request.
    std::string body;
    /// @brief fields to extract from the response.
    std::vector<ReadField> fields;
};

/// @brief resolved index channel source info.
struct IndexSource {
    /// @brief key of the index channel.
    synnax::channel::Key index_key;
    /// @brief index into the endpoints vector this source belongs to.
    int endpoint_index;
    /// @brief if set, extract the timestamp from the response JSON.
    std::optional<TimeInfo> time_info;
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
    /// @brief whether to be strict about type conversions.
    bool strict;
    /// @brief endpoints to poll.
    std::vector<ReadEndpoint> endpoints;
    /// @brief resolved index sources.
    std::vector<IndexSource> index_sources;
    /// @brief mapping of channel keys to their Synnax channel definitions.
    std::map<synnax::channel::Key, synnax::channel::Channel> channels;

    /// @brief parses a read task config from a Synnax task definition.
    static std::pair<ReadTaskConfig, x::errors::Error>
    parse(const std::shared_ptr<task::Context> &, const synnax::task::Task &);
};

/// @brief source that polls HTTP endpoints and writes extracted values to a frame.
class ReadTaskSource : public common::Source {
    ReadTaskConfig cfg_;
    device::Client client_;
    std::vector<synnax::channel::Channel> channels_;

public:
    ReadTaskSource(ReadTaskConfig, device::Client);

    [[nodiscard]] synnax::framer::WriterConfig writer_config() const override;

    [[nodiscard]] std::vector<synnax::channel::Channel> channels() const override;

    common::ReadResult read(x::breaker::Breaker &, x::telem::Frame &) override;
};

/// @brief configures an HTTP read task from a Synnax task definition.
std::pair<common::ConfigureResult, x::errors::Error>
configure_read(const std::shared_ptr<task::Context> &, const synnax::task::Task &);
}
