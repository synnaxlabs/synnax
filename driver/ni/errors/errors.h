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
#include "driver/ni/errors/fields.h"

// Static regex patterns
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

xerrors::Error parse_error(const std::string& s) {
    auto processed_s = s;
    
    processed_s = std::regex_replace(processed_s, TASK_NAME_LINE_REGEX, "");

    // // Extract status code
    // std::string sc;
    // std::smatch status_code_match;
    // if (std::regex_search(processed_s, status_code_match, STATUS_CODE_REGEX))
    //     sc = status_code_match[1].str();

    // Remove the redundant Status Code line at the end
    processed_s = std::regex_replace(processed_s, STATUS_CODE_LINE_REGEX, "");

    // Extract device name and channel
    std::string device;
    std::smatch device_match;
    if (std::regex_search(processed_s, device_match, DEVICE_REGEX))
        device = device_match[1].str();

    // Extract physical channel name or channel name
    std::string cn;
    std::smatch physical_channel_match;
    if (std::regex_search(processed_s, physical_channel_match, PHYSICAL_CHANNEL_REGEX)) {
        cn = physical_channel_match[1].str();
        if (!device.empty()) cn = make_channel_name(device, cn);
    } else {
        std::smatch channel_match;
        if (std::regex_search(processed_s, channel_match, CHANNEL_REGEX))
            cn = channel_match[1].str();
    }

    // Extract and process other fields
    std::string p = "";
    std::smatch property_match;
    if (std::regex_search(processed_s, property_match, PROPERTY_REGEX))
        p = property_match[1].str();
    if (sc == "-200170") p = "port";

    // Extract possible values
    std::string possible_values;
    std::smatch possible_values_match;
    if (std::regex_search(processed_s, possible_values_match, POSSIBLE_VALUES_REGEX)) {
        possible_values = possible_values_match[1].str();
        size_t pos = possible_values.find("Channel Name");
        if (pos != std::string::npos)
            possible_values.erase(pos, std::string("Channel Name").length());
    }

    // Extract maximum and minimum values
    std::string max_value;
    std::string min_value;
    std::smatch value_match;
    if (std::regex_search(processed_s, value_match, MAX_VALUE_REGEX))
        max_value = value_match[1].str();
    if (std::regex_search(processed_s, value_match, MIN_VALUE_REGEX))
        min_value = value_match[1].str();

    // Set the path
    std::string mapped_channel = get_mapped_channel(cn);
    err_info["path"] = mapped_channel.empty() ? (cn.empty() ? "" : cn + ".") : mapped_channel + ".";

    // Check if the property is in the field map
    if (FIELD_MAP.count(p) == 0)
        err_info["path"] = err_info["path"].get<std::string>() + p;
    else
        err_info["path"] = err_info["path"].get<std::string>() + FIELD_MAP.at(p);

    // Construct the error message
    std::string error_message = "NI Error " + sc + ": " + processed_s + "\nPath: " + 
                               err_info["path"].get<std::string>();
    if (!cn.empty()) error_message += " Channel: " + cn;
    if (!possible_values.empty()) error_message += " Possible Values: " + possible_values;
    if (!max_value.empty()) error_message += " Maximum Value: " + max_value;
    if (!min_value.empty()) error_message += " Minimum Value: " + min_value;
    err_info["message"] = error_message;

    json j = json::array();
    j.push_back(err_info);

    LOG(INFO) << err_info.dump(4);
}