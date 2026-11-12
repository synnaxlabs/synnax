// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstring>
#include <functional>
#include <map>

#include "gtest/gtest.h"

#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ni/channel/channels.h"
#include "driver/ni/channel/verify.h"
#include "driver/ni/daqmx/mock.h"

namespace driver::ni::channel {
namespace {
const std::string
    RANGE_ERROR = "Requested value is not a supported value for this "
                  "property.\n\nProperty: DAQmx_AI_Min\nRequested Value: "
                  "-20.0\nValue Must Be Greater Than: -10.0\nValue Must Be "
                  "Less Than:  10.0\n\nDevice: Dev1\n\nTask Name: _task\n";

const std::string RATE_ERROR = "Requested value is not a supported value for this "
                               "property.\n\nProperty: DAQmx_SampClk_Rate\nRequested "
                               "Value: 60.0e3\nMaximum Value: 51.2e3\nMinimum Value: "
                               "1.0\n\nDevice: Dev1\n";

const std::string EXCITATION_ERROR = "Requested value is not a supported value for "
                                     "this property.\n\nProperty: DAQmx_AI_Excit_Val\n";

const std::string TERMINAL_CONFIG_ERROR = "Requested value is not a supported value "
                                          "for this property.\n\nProperty: "
                                          "DAQmx_AI_TermCfg\nRequested Value: "
                                          "DAQmx_Val_Diff\nPossible Values: "
                                          "DAQmx_Val_RSE\n\nChannel Name: channels_0\n";

/// @brief a DAQmx API for one device that rejects voltage channels with a minimum
/// below -10 at Verify.
class FakeAPI final : public daqmx::MockAPI {
public:
    /// @brief the error text that Verify returns for each failure, when set.
    std::string verify_error = RANGE_ERROR;
    /// @brief whether the range error names the channel that fails.
    bool names_channel = false;
    /// @brief the error text that CreateAIVoltageChan returns, when not empty.
    std::string create_error;
    /// @brief the minimum value of each channel in each open task, by channel name.
    std::map<TaskHandle, std::map<std::string, double>> tasks;

    int32 CreateTask(const char[], TaskHandle *task) override {
        *task = reinterpret_cast<TaskHandle>(++this->next_handle);
        this->tasks[*task] = {};
        return 0;
    }

    int32 ClearTask(const TaskHandle task) override {
        this->tasks.erase(task);
        return 0;
    }

    int32 CreateAIVoltageChan(
        const TaskHandle task,
        const char[],
        const char name[],
        int32,
        const float64 min_val,
        float64,
        int32,
        const char[]
    ) override {
        if (!this->create_error.empty()) return this->fail(this->create_error);
        this->tasks[task][name] = min_val;
        return 0;
    }

    int32 TaskControl(const TaskHandle task, int32) override {
        if (this->verify_error != RANGE_ERROR) return this->fail(this->verify_error);
        for (const auto &[name, min_val]: this->tasks[task]) {
            if (min_val >= -10) continue;
            if (!this->names_channel) return this->fail(RANGE_ERROR);
            return this->fail(RANGE_ERROR + "Channel Name: " + name + "\n");
        }
        return 0;
    }

    int32 GetExtendedErrorInfo(char buf[], const uInt32 size) override {
        if (buf == nullptr) return static_cast<int32>(this->error.size() + 1);
        std::strncpy(buf, this->error.c_str(), size);
        return 0;
    }

private:
    std::uintptr_t next_handle = 0;
    std::string error;

    int32 fail(const std::string &text) {
        this->error = text;
        return DAQmxErrorInvalidAttributeValue;
    }
};

/// @brief returns a voltage channel config with the given minimum value.
x::json::json voltage_channel(const double min_val) {
    return {
        {"type", "ai_voltage"},
        {"key", "key"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", min_val},
        {"max_val", 10},
        {"units", "Volts"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "dev"},
    };
}

/// @brief parses voltage channels with the given minimum values and binds them to
/// Dev1.
std::vector<std::unique_ptr<Input>> parse_channels(const std::vector<double> &mins) {
    x::json::json cfg = {{"channels", x::json::json::array()}};
    for (const auto min_val: mins)
        cfg["channels"].push_back(voltage_channel(min_val));
    x::json::Parser parser(cfg);
    auto channels = parser.map<std::unique_ptr<Input>>(
        "channels",
        [](x::json::Parser &ch) -> std::pair<std::unique_ptr<Input>, bool> {
            return {parse_input(ch, "ni_analog_read"), true};
        }
    );
    EXPECT_NIL(parser.error());
    for (const auto &ch: channels)
        ch->bind_remote_info(synnax::channel::Channel(), "Dev1");
    return channels;
}

struct Result {
    x::errors::Error err;
    std::shared_ptr<FakeAPI> api;
};

/// @brief applies the channels to a task on the fake API and verifies it.
Result run_verify(
    const std::vector<double> &mins,
    const std::function<void(FakeAPI &)> &setup = [](FakeAPI &) {}
) {
    auto api = std::make_shared<FakeAPI>();
    setup(*api);
    const auto dmx = std::make_shared<daqmx::SugaredAPI>(api);
    const auto channels = parse_channels(mins);
    TaskHandle task = nullptr;
    EXPECT_NIL(dmx->CreateTask("", &task));
    for (const auto &ch: channels)
        if (const auto err = ch->apply(dmx, task)) return {err, api};
    return {verify(dmx, task, channels), api};
}

/// @brief returns the field errors of a validation error.
x::json::json fields_of(const x::errors::Error &err) {
    EXPECT_TRUE(err.matches(x::errors::VALIDATION));
    return x::json::json::parse(err.data)["errors"];
}
}

/// @brief it should pass a task whose channels DAQmx accepts.
TEST(VerifyTest, PassesValidTask) {
    ASSERT_NIL(run_verify({-10, -5}).err);
}

/// @brief it should put a range error on the channel that DAQmx names.
TEST(VerifyTest, PutsErrorOnNamedChannel) {
    const auto [err, _] = run_verify({-10, -20}, [](FakeAPI &api) {
        api.names_channel = true;
    });
    const auto fields = fields_of(err);
    ASSERT_EQ(fields.size(), 1);
    EXPECT_EQ(fields[0]["path"], "channels.1.min_val");
    EXPECT_EQ(fields[0]["message"], "Must be between -10 and 10");
}

/// @brief it should verify each channel alone to find the channel that DAQmx does not
/// name, and clear each task it makes.
TEST(VerifyTest, FindsUnnamedChannel) {
    const auto [err, api] = run_verify({-10, -5, -20});
    const auto fields = fields_of(err);
    ASSERT_EQ(fields.size(), 1);
    EXPECT_EQ(fields[0]["path"], "channels.2.min_val");
    EXPECT_EQ(api->tasks.size(), 1);
}

/// @brief it should put a sample clock error on the task field.
TEST(VerifyTest, PutsRateErrorOnTask) {
    const auto [err, _] = run_verify({-10}, [](FakeAPI &api) {
        api.verify_error = RATE_ERROR;
    });
    const auto fields = fields_of(err);
    ASSERT_EQ(fields.size(), 1);
    EXPECT_EQ(fields[0]["path"], "sample_rate");
    EXPECT_EQ(fields[0]["message"], "Must be between 1 and 51200");
}

/// @brief it should return an error on a property the config does not set as is.
TEST(VerifyTest, KeepsUnknownPropertyError) {
    const auto [err, _] = run_verify({-10}, [](FakeAPI &api) {
        api.verify_error = EXCITATION_ERROR;
    });
    EXPECT_FALSE(err.matches(x::errors::VALIDATION));
    EXPECT_TRUE(err.matches(daqmx::CRITICAL_ERROR));
    EXPECT_EQ(err.data, EXCITATION_ERROR);
}

/// @brief it should put an error from creating a channel on its field, with the first
/// sentence of the DAQmx text as the message.
TEST(VerifyTest, PutsCreateErrorOnField) {
    const auto [err, _] = run_verify({-10}, [](FakeAPI &api) {
        api.create_error = TERMINAL_CONFIG_ERROR;
    });
    const auto fields = fields_of(err);
    ASSERT_EQ(fields.size(), 1);
    EXPECT_EQ(fields[0]["path"], "channels.0.terminal_config");
    EXPECT_EQ(
        fields[0]["message"],
        "Requested value is not a supported value for this property."
    );
}

/// @brief it should lead an error from creating a channel with the DAQmx channel name
/// when the error names no field.
TEST(VerifyTest, PrefixesCreateErrorWithoutField) {
    const auto [err, _] = run_verify({-10}, [](FakeAPI &api) {
        api.create_error = "Physical channel specified does not exist on this device.";
    });
    EXPECT_FALSE(err.matches(x::errors::VALIDATION));
    EXPECT_EQ(
        err.data,
        "channels_0: Physical channel specified does not exist on this device."
    );
}

/// @brief it should state one limit when DAQmx gives only one.
TEST(VerifyTest, MessageStatesOneLimit) {
    EXPECT_EQ(
        field_message("Bad value.\nValue Must Be Greater Than: 0.5\n"),
        "Must be greater than 0.5"
    );
    EXPECT_EQ(
        field_message("Bad value.\nMaximum Value: 1.0e6\n"),
        "Must be less than 1000000"
    );
}
}
