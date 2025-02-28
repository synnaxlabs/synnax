// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <regex>

/// module
#include "x/cpp/xerrors/errors.h"

/// internal
#include "nlohmann/json.hpp"
#include "driver/errors/errors.h"
#include "driver/ni/daqmx/daqmx.h"

using json = nlohmann::json;

static const std::regex STATUS_CODE_REGEX(R"(Status Code:\s*(-?\d+))");
static const std::regex CHANNEL_REGEX(R"(Channel Name:\s*(\S+))");
static const std::regex PHYSICAL_CHANNEL_REGEX(R"(Physical Channel Name:\s*(\S+))");
static const std::regex DEVICE_REGEX(R"(Device:\s*(\S+))");
static const std::regex POSSIBLE_VALUES_REGEX(R"(Possible Values:\s*([\w\s,.-]+))");
static const std::regex MAX_VALUE_REGEX(R"(Maximum Value:\s*([\d.\s,eE-]+))");
static const std::regex MIN_VALUE_REGEX(R"(Minimum Value:\s*([\d.\s,eE-]+))");
static const std::regex PROPERTY_REGEX(R"(Property:\s*(\S+))");
static const std::regex TASK_NAME_REGEX(R"(Task Name:\s*(\S+))");
static const std::regex TASK_NAME_LINE_REGEX(R"(\nTask Name:.*\n?)");
static const std::regex STATUS_CODE_LINE_REGEX(R"(\nStatus Code:.*$)");

const xerrors::Error BASE_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("ni");
const xerrors::Error FIELD_ERROR = BASE_ERROR.sub("field");
const xerrors::Error ANALOG_WRITE_OUT_OF_BOUNDS = BASE_ERROR.sub("200561");

using Status = int32;

static std::string get_error_msg(
    const std::shared_ptr<DAQmx> &dmx,
    const Status status
) {
    if (status == 0) return "";
    const size_t bytes_to_alloc = dmx->GetExtendedErrorInfo(nullptr, 0);
    std::vector<char> err_buf(bytes_to_alloc);  
    dmx->GetExtendedErrorInfo(err_buf.data(), err_buf.size());
    return std::string(err_buf.data());
}

struct FieldErrorInfo {
    std::string path;
    std::string message;

    std::string to_string() const {
        return nlohmann::to_string(json{
            {"path", path},
            {"message", message}
        });
    }
};

static xerrors::Error parse_error(
    const std::shared_ptr<DAQmx> &dmx,
    const Status status
) {
    if (status == 0) return xerrors::NIL;
    const auto err_msg = get_error_msg(dmx, status);
    return xerrors::Error{BASE_ERROR.sub(std::to_string(status).substr(1)), err_msg};
}
