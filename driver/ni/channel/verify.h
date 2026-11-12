// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdlib>
#include <iomanip>
#include <memory>
#include <sstream>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/strings/strings.h"

#include "driver/ni/daqmx/sugared.h"

namespace driver::ni::channel {
/// @brief the channel config field that each DAQmx channel property sets.
const std::unordered_map<std::string, std::string> CHANNEL_FIELDS = {
    {"DAQmx_AI_Min", "min_val"},
    {"DAQmx_AI_Max", "max_val"},
    {"DAQmx_AO_Min", "min_val"},
    {"DAQmx_AO_Max", "max_val"},
    {"DAQmx_CI_Min", "min_val"},
    {"DAQmx_CI_Max", "max_val"},
    {"DAQmx_AI_TermCfg", "terminal_config"},
    {"DAQmx_AO_TermCfg", "terminal_config"},
    {"DAQmx_AI_Thrmcpl_CJCSrc", "cjc.source"},
};

/// @brief the task config field that each DAQmx timing property sets.
const std::unordered_map<std::string, std::string> TASK_FIELDS = {
    {"DAQmx_SampClk_Rate", "sample_rate"},
    {"DAQmx_SampClk_Src", "timing_source"},
};

/// @brief returns the value of the "label: value" line in DAQmx error text, or an
/// empty string when the text has no such line.
inline std::string line_value(const std::string &text, const std::string &label) {
    const auto key = label + ":";
    std::istringstream lines(text);
    std::string line;
    while (std::getline(lines, line))
        if (line.starts_with(key)) return x::strings::trim(line.substr(key.size()));
    return "";
}

/// @brief formats a DAQmx number with no trailing zeros, so "60.0" reads "60".
inline std::string format_number(const std::string &value) {
    char *end = nullptr;
    const double number = std::strtod(value.c_str(), &end);
    if (end == value.c_str()) return value;
    std::ostringstream out;
    out << std::setprecision(10) << number;
    return out.str();
}

/// @brief returns the config values in a DAQmx "Possible Values" list, such as "RSE"
/// for "DAQmx_Val_RSE".
inline std::vector<std::string> possible_values(const std::string &text) {
    const std::string prefix = "DAQmx_Val_";
    std::vector<std::string> values;
    std::istringstream list(line_value(text, "Possible Values"));
    std::string value;
    while (std::getline(list, value, ',')) {
        value = x::strings::trim(value);
        if (value.starts_with(prefix)) value = value.substr(prefix.size());
        if (!value.empty()) values.push_back(value);
    }
    return values;
}

/// @brief returns a short message for DAQmx error text: the limits or the values it
/// gives, or else its first sentence.
inline std::string field_message(const std::string &text) {
    auto low = line_value(text, "Value Must Be Greater Than");
    auto high = line_value(text, "Value Must Be Less Than");
    if (low.empty()) low = line_value(text, "Minimum Value");
    if (high.empty()) high = line_value(text, "Maximum Value");
    if (!low.empty() && !high.empty())
        return "Must be between " + format_number(low) + " and " + format_number(high);
    if (!low.empty()) return "Must be greater than " + format_number(low);
    if (!high.empty()) return "Must be less than " + format_number(high);
    const auto values = possible_values(text);
    if (values.size() == 1) return "Must be " + values[0];
    if (!values.empty()) return "Must be one of " + x::strings::join(values, ", ");
    const auto line = text.substr(0, text.find('\n'));
    const auto end = line.find(". ");
    return x::strings::trim(end == std::string::npos ? line : line.substr(0, end + 1));
}

/// @brief returns a validation error that puts err on the config field at path.
inline x::errors::Error
field_err(const x::errors::Error &err, const std::string &path) {
    x::json::json field = {{"path", path}, {"message", field_message(err.data)}};
    const x::json::json data = {{"errors", x::json::json::array({field})}};
    return {x::errors::VALIDATION, data.dump()};
}

/// @brief returns true when a channel config of type Config has the field.
template<typename Config>
bool has_field(const Config &cfg, const std::string &field) {
    if (field == "min_val" || field == "max_val") return requires { cfg.min_val; };
    if (field == "terminal_config") return requires { cfg.terminal_config; };
    if (field == "cjc.source") return requires { cfg.cjc; };
    return false;
}

/// @brief returns the field of ch that the DAQmx property in text sets, or an empty
/// string when the property sets no field that ch has.
template<typename Channel>
std::string channel_field(const std::string &text, const Channel &ch) {
    const auto it = CHANNEL_FIELDS.find(line_value(text, "Property"));
    if (it == CHANNEL_FIELDS.end() || !ch.has(it->second)) return "";
    return it->second;
}

/// @brief returns err from ch as a field error when its DAQmx property sets a field
/// that ch has. Any other error comes back led by the DAQmx channel name.
template<typename Channel>
x::errors::Error at_channel(const x::errors::Error &err, const Channel &ch) {
    if (!err) return err;
    const auto field = channel_field(err.data, ch);
    if (field.empty()) return {err, ch.cfg_path + ": " + err.data};
    return field_err(err, ch.path + "." + field);
}

/// @brief returns err as a field error when its DAQmx property is a task field.
inline x::errors::Error at_task(const x::errors::Error &err) {
    if (!err) return err;
    const auto it = TASK_FIELDS.find(line_value(err.data, "Property"));
    if (it == TASK_FIELDS.end()) return err;
    return field_err(err, it->second);
}

/// @brief applies ch to a scratch task and runs DAQmx Verify on it.
/// @returns the error that ch gets alone, and the error from creating or clearing the
/// scratch task.
template<typename Channel>
std::pair<x::errors::Error, x::errors::Error>
verify_alone(const std::shared_ptr<daqmx::SugaredAPI> &dmx, const Channel &ch) {
    TaskHandle scratch = nullptr;
    if (const auto err = dmx->CreateTask("", &scratch)) return {x::errors::NIL, err};
    auto err = ch.apply(dmx, scratch);
    if (!err) err = dmx->TaskControl(scratch, DAQmx_Val_Task_Verify);
    return {err, dmx->ClearTask(scratch)};
}

/// @brief runs DAQmx Verify on the task, which holds the given channels. A failure on a
/// field that the config sets comes back as a field error. When DAQmx does not name
/// the channel, each channel is verified alone in a scratch task to find it. When no
/// channel fails alone, the error comes back as DAQmx gave it.
template<typename Channels>
x::errors::Error verify(
    const std::shared_ptr<daqmx::SugaredAPI> &dmx,
    TaskHandle task,
    const Channels &channels
) {
    const auto err = dmx->TaskControl(task, DAQmx_Val_Task_Verify);
    if (!err) return err;
    const auto property = line_value(err.data, "Property");
    if (TASK_FIELDS.contains(property)) return at_task(err);
    if (!CHANNEL_FIELDS.contains(property)) return err;
    const auto name = line_value(err.data, "Channel Name");
    for (const auto &ch: channels)
        if (!name.empty() && ch->cfg_path == name) return at_channel(err, *ch);
    for (const auto &ch: channels) {
        const auto [alone, scratch_err] = verify_alone(dmx, *ch);
        if (scratch_err) return scratch_err;
        if (line_value(alone.data, "Property") == property) return at_channel(err, *ch);
    }
    return err;
}
}
