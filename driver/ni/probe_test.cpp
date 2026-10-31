// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstring>
#include <memory>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/ni/json.gen.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ni/daqmx/mock.h"
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/probe.h"

namespace driver::ni {
namespace {
const std::string DEVICE = "Dev1";
const std::string ERROR_MESSAGE = "Device not found.";

/// @brief an in-memory DAQmx API that holds the attributes of one device and of its
/// physical channels. It does not support an attribute that it does not hold.
class FakeAPI final : public daqmx::MockAPI {
    template<typename T>
    static int32 get(const std::unordered_map<int32, T> &values, int32 attr, T *value) {
        const auto it = values.find(attr);
        if (it == values.end()) return DAQmxErrorAttrNotSupported;
        *value = it->second;
        return 0;
    }

    template<typename T>
    static int32 get_array(
        const std::unordered_map<int32, std::vector<T>> &values,
        const int32 attr,
        T value[],
        const uInt32 size
    ) {
        const auto it = values.find(attr);
        if (it == values.end()) return DAQmxErrorAttrNotSupported;
        if (size == 0) return static_cast<int32>(it->second.size());
        std::ranges::copy(it->second, value);
        return 0;
    }

    template<typename T>
    static const std::unordered_map<int32, T> &
    of(const std::unordered_map<std::string, std::unordered_map<int32, T>> &channels,
       const char channel[]) {
        static const std::unordered_map<int32, T> none;
        const auto it = channels.find(channel);
        return it == channels.end() ? none : it->second;
    }

public:
    std::unordered_map<int32, int32> ints;
    std::unordered_map<int32, std::string> strings;
    std::unordered_map<int32, float64> doubles;
    std::unordered_map<int32, std::vector<float64>> double_arrays;
    /// @brief attributes of the physical channels, keyed by channel name.
    std::unordered_map<std::string, std::unordered_map<int32, int32>> channel_ints;
    std::unordered_map<std::string, std::unordered_map<int32, uInt32>> channel_uints;
    std::unordered_map<std::string, std::unordered_map<int32, std::vector<int32>>>
        channel_arrays;
    /// @brief what DAQmx_AI_TermCfg reads back on a channel made with Cfg_Default.
    int32 default_terminal_config = DAQmx_Val_Diff;
    /// @brief the status that CreateAIVoltageChan and CreateAICurrentChan return.
    int32 create_channel_status = 0;
    /// @brief the statuses of device attributes that fail with an error.
    std::unordered_map<int32, int32> failures;

    int32
    GetDeviceAttributeInt32(const char[], const int32 attr, int32 *value) override {
        return get(this->ints, attr, value);
    }

    int32 GetDeviceAttributeString(
        const char[],
        const int32 attr,
        char value[],
        const uInt32 size
    ) override {
        if (const auto it = this->failures.find(attr); it != this->failures.end())
            return it->second;
        const auto it = this->strings.find(attr);
        if (it == this->strings.end()) return DAQmxErrorAttrNotSupported;
        if (size == 0) return static_cast<int32>(it->second.size() + 1);
        std::strncpy(value, it->second.c_str(), size);
        return 0;
    }

    int32
    GetDeviceAttributeDouble(const char[], const int32 attr, float64 *value) override {
        return get(this->doubles, attr, value);
    }

    int32 GetDeviceAttributeDoubleArray(
        const char[],
        const int32 attr,
        float64 value[],
        const uInt32 size
    ) override {
        return get_array(this->double_arrays, attr, value, size);
    }

    int32 GetPhysicalChanAttributeInt32(
        const char channel[],
        const int32 attr,
        int32 *value
    ) override {
        return get(of(this->channel_ints, channel), attr, value);
    }

    int32 GetPhysicalChanAttributeUInt32(
        const char channel[],
        const int32 attr,
        uInt32 *value
    ) override {
        return get(of(this->channel_uints, channel), attr, value);
    }

    int32 GetPhysicalChanAttributeInt32Array(
        const char channel[],
        const int32 attr,
        int32 value[],
        const uInt32 size
    ) override {
        return get_array(of(this->channel_arrays, channel), attr, value, size);
    }

    int32 CreateTask(const char sessionName[], TaskHandle *task) override {
        MockAPI::CreateTask(sessionName, task);
        *task = this;
        return 0;
    }

    int32 CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        const int32 terminalConfig,
        const float64 minVal,
        const float64 maxVal,
        const int32 units,
        const char customScaleName[]
    ) override {
        MockAPI::CreateAIVoltageChan(
            task,
            physicalChannel,
            nameToAssignToChannel,
            terminalConfig,
            minVal,
            maxVal,
            units,
            customScaleName
        );
        return this->create_channel_status;
    }

    int32 CreateAICurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        const int32 terminalConfig,
        const float64 minVal,
        const float64 maxVal,
        const int32 units,
        const int32 shuntResistorLoc,
        const float64 extShuntResistorVal,
        const char customScaleName[]
    ) override {
        MockAPI::CreateAICurrentChan(
            task,
            physicalChannel,
            nameToAssignToChannel,
            terminalConfig,
            minVal,
            maxVal,
            units,
            shuntResistorLoc,
            extShuntResistorVal,
            customScaleName
        );
        return this->create_channel_status;
    }

    int32 GetChanAttributeInt32(
        TaskHandle,
        const char[],
        const int32,
        int32 *value
    ) override {
        *value = this->default_terminal_config;
        return 0;
    }

    int32 GetExtendedErrorInfo(char errorString[], const uInt32 bufferSize) override {
        if (errorString == nullptr) return static_cast<int32>(ERROR_MESSAGE.size() + 1);
        std::strncpy(errorString, ERROR_MESSAGE.c_str(), bufferSize);
        return 0;
    }

    /// @brief adds the analog input ports ai0 to ai<count - 1>, each accepting the
    /// terminal configuration bits and measurement types given.
    void add_analog_inputs(
        const int count,
        const int32 configs,
        const std::vector<int32> &types
    ) {
        std::string list;
        for (int i = 0; i < count; i++) {
            const auto name = DEVICE + "/ai" + std::to_string(i);
            list += (i == 0 ? "" : ", ") + name;
            this->channel_ints[name][DAQmx_PhysicalChan_AI_TermCfgs] = configs;
            this->channel_arrays[name]
                                [DAQmx_PhysicalChan_AI_SupportedMeasTypes] = types;
        }
        this->strings[DAQmx_Dev_AI_PhysicalChans] = list;
    }
};

/// @brief returns the hardware that probe reads from api.
::synnax::ni::Hardware probe_api(const std::shared_ptr<FakeAPI> &api) {
    daqmx::SugaredAPI dmx(api);
    return ASSERT_NIL_P(probe(dmx, DEVICE));
}

/// @brief returns the negative port of each analog input port, keyed by port.
std::unordered_map<std::int32_t, std::optional<std::int32_t>>
negatives(const ::synnax::ni::Hardware &hw) {
    std::unordered_map<std::int32_t, std::optional<std::int32_t>> out;
    for (const auto &ai: hw.analog_inputs)
        out[ai.port] = ai.negative;
    return out;
}

const int32 RSE_NRSE_DIFF = DAQmx_Val_Bit_TermCfg_RSE | DAQmx_Val_Bit_TermCfg_NRSE |
                            DAQmx_Val_Bit_TermCfg_Diff;
const int32 RSE_DIFF = DAQmx_Val_Bit_TermCfg_RSE | DAQmx_Val_Bit_TermCfg_Diff;
}

/// @brief it should pair port n with n + 8 on an M Series device, for positives 0-7.
TEST(ProbeTest, testPairsMSeriesPorts) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_MSeriesDAQ;
    api->strings[DAQmx_Dev_ProductType] = "PCI-6221";
    api->add_analog_inputs(16, RSE_NRSE_DIFF, {DAQmx_Val_Voltage});
    const auto hw = probe_api(api);
    EXPECT_EQ(hw.product_category, "m_series_daq");
    EXPECT_EQ(hw.product_type, "PCI-6221");
    ASSERT_EQ(hw.analog_inputs.size(), 16);
    const auto pairs = negatives(hw);
    for (std::int32_t port = 0; port < 8; port++)
        EXPECT_EQ(pairs.at(port), port + 8) << "ai" << port;
    for (std::int32_t port = 8; port < 16; port++)
        EXPECT_EQ(pairs.at(port), std::nullopt) << "ai" << port;
}

/// @brief it should pair port n with n + 8 on an NI 9205, for positives 0-7 and 16-23.
TEST(ProbeTest, testPairsByProductType) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_CSeriesModule;
    api->strings[DAQmx_Dev_ProductType] = "NI 9205";
    api->add_analog_inputs(32, RSE_NRSE_DIFF, {DAQmx_Val_Voltage});
    const auto pairs = negatives(probe_api(api));
    EXPECT_EQ(pairs.at(0), 8);
    EXPECT_EQ(pairs.at(7), 15);
    EXPECT_EQ(pairs.at(8), std::nullopt);
    EXPECT_EQ(pairs.at(16), 24);
    EXPECT_EQ(pairs.at(23), 31);
    EXPECT_EQ(pairs.at(24), std::nullopt);
}

/// @brief it should pair port n with n + 4 on a USB-6008, for positives 0-3.
TEST(ProbeTest, testPairsUSBPorts) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_USBDAQ;
    api->strings[DAQmx_Dev_ProductType] = "USB-6008";
    api->add_analog_inputs(8, RSE_DIFF, {DAQmx_Val_Voltage});
    const auto pairs = negatives(probe_api(api));
    for (std::int32_t port = 0; port < 4; port++)
        EXPECT_EQ(pairs.at(port), port + 4) << "ai" << port;
    for (std::int32_t port = 4; port < 8; port++)
        EXPECT_EQ(pairs.at(port), std::nullopt) << "ai" << port;
}

/// @brief it should not pair ports that do not accept RSE, such as those of a
/// simultaneous X Series device.
TEST(ProbeTest, testDoesNotPairPortsWithoutRSE) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_XSeriesDAQ;
    api->strings[DAQmx_Dev_ProductType] = "PXIe-6358";
    api->add_analog_inputs(16, DAQmx_Val_Bit_TermCfg_Diff, {DAQmx_Val_Voltage});
    for (const auto &[port, negative]: negatives(probe_api(api)))
        EXPECT_EQ(negative, std::nullopt) << "ai" << port;
}

/// @brief it should not pair the ports of a device outside the pair table.
TEST(ProbeTest, testDoesNotPairOtherDevices) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_CSeriesModule;
    api->strings[DAQmx_Dev_ProductType] = "NI 9221";
    api->add_analog_inputs(16, RSE_DIFF, {DAQmx_Val_Voltage});
    for (const auto &[port, negative]: negatives(probe_api(api)))
        EXPECT_EQ(negative, std::nullopt) << "ai" << port;
}

/// @brief it should read back what Cfg_Default gives with a task on the narrowest
/// voltage range, then clear the task.
TEST(ProbeTest, testReadsBackDefaultTerminalConfig) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_MSeriesDAQ;
    api->double_arrays[DAQmx_Dev_AI_VoltageRngs] = {-10, 10, -0.2, 0.2, -5, 5};
    api->add_analog_inputs(2, RSE_NRSE_DIFF, {DAQmx_Val_Voltage});
    api->default_terminal_config = DAQmx_Val_RSE;
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 2);
    for (const auto &ai: hw.analog_inputs) {
        EXPECT_EQ(
            ai.terminal_configs,
            std::vector<std::string>({"RSE", "NRSE", "Diff"})
        );
        EXPECT_EQ(ai.default_terminal_config, "RSE");
    }
    const auto chans = api->calls_to("CreateAIVoltageChan");
    ASSERT_EQ(chans.size(), 2);
    EXPECT_EQ(chans[0]["args"]["physicalChannel"], "Dev1/ai0");
    EXPECT_EQ(chans[0]["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(chans[0]["args"]["minVal"], -0.2);
    EXPECT_EQ(chans[0]["args"]["maxVal"], 0.2);
    EXPECT_EQ(api->calls_to("CreateTask").size(), 2);
    EXPECT_EQ(api->calls_to("ClearTask").size(), 2);
    EXPECT_TRUE(api->calls_to("StartTask").empty());
}

/// @brief it should leave the default absent and clear the task when DAQmx rejects the
/// readback channel.
TEST(ProbeTest, testLeavesDefaultAbsentWhenReadbackFails) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(1, RSE_NRSE_DIFF, {DAQmx_Val_Voltage});
    api->create_channel_status = DAQmxErrorInvalidDeviceID;
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 1);
    EXPECT_EQ(hw.analog_inputs[0].default_terminal_config, std::nullopt);
    EXPECT_EQ(api->calls_to("ClearTask").size(), 1);
    EXPECT_FALSE(is_complete(hw));
}

/// @brief it should read back the default through a current channel on the narrowest
/// current range when the port takes no voltage channel.
TEST(ProbeTest, testReadsBackDefaultThroughCurrentChannel) {
    auto api = std::make_shared<FakeAPI>();
    api->double_arrays[DAQmx_Dev_AI_CurrentRngs] = {-0.02, 0.02, -0.01, 0.01};
    api->add_analog_inputs(1, RSE_DIFF, {DAQmx_Val_Current});
    api->default_terminal_config = DAQmx_Val_RSE;
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 1);
    EXPECT_EQ(hw.analog_inputs[0].default_terminal_config, "RSE");
    EXPECT_TRUE(is_complete(hw));
    EXPECT_TRUE(api->calls_to("CreateAIVoltageChan").empty());
    const auto chans = api->calls_to("CreateAICurrentChan");
    ASSERT_EQ(chans.size(), 1);
    EXPECT_EQ(chans[0]["args"]["physicalChannel"], "Dev1/ai0");
    EXPECT_EQ(chans[0]["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(chans[0]["args"]["minVal"], -0.01);
    EXPECT_EQ(chans[0]["args"]["maxVal"], 0.01);
    EXPECT_EQ(chans[0]["args"]["units"], DAQmx_Val_Amps);
    EXPECT_EQ(chans[0]["args"]["shuntResistorLoc"], DAQmx_Val_Default);
    EXPECT_EQ(api->calls_to("ClearTask").size(), 1);
    EXPECT_TRUE(api->calls_to("StartTask").empty());
}

/// @brief it should leave the absent values of the hardware out of its JSON, so that
/// the JSON parses back to the same hardware.
TEST(ProbeTest, testLeavesAbsentValuesOutOfJSON) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(1, RSE_NRSE_DIFF, {DAQmx_Val_Voltage});
    api->create_channel_status = DAQmxErrorInvalidDeviceID;
    const auto j = probe_api(api).to_json();
    EXPECT_FALSE(j.contains("ai_min_rate"));
    EXPECT_FALSE(j.contains("ao_max_rate"));
    ASSERT_EQ(j["analog_inputs"].size(), 1);
    EXPECT_FALSE(j["analog_inputs"][0].contains("default_terminal_config"));
    EXPECT_FALSE(j["analog_inputs"][0].contains("negative"));
    EXPECT_EQ(::synnax::ni::Hardware::parse(x::json::Parser(j)).to_json(), j);
}

/// @brief it should take the one terminal configuration of a port as its default
/// without a task, whatever channels the port takes.
TEST(ProbeTest, testTakesOnlyTerminalConfigAsDefault) {
    for (const auto type: {DAQmx_Val_Voltage, DAQmx_Val_Current, DAQmx_Val_Bridge}) {
        auto api = std::make_shared<FakeAPI>();
        api->add_analog_inputs(1, DAQmx_Val_Bit_TermCfg_Diff, {type});
        const auto hw = probe_api(api);
        ASSERT_EQ(hw.analog_inputs.size(), 1);
        EXPECT_EQ(hw.analog_inputs[0].default_terminal_config, "Diff") << type;
        EXPECT_TRUE(api->calls_to("CreateTask").empty()) << type;
    }
}

/// @brief it should leave the default absent on a port that accepts no terminal
/// configuration.
TEST(ProbeTest, testNoDefaultWithoutTerminalConfigs) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(1, 0, {DAQmx_Val_Voltage});
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 1);
    EXPECT_EQ(hw.analog_inputs[0].default_terminal_config, std::nullopt);
    EXPECT_TRUE(is_complete(hw));
    EXPECT_TRUE(api->calls_to("CreateTask").empty());
}

/// @brief it should leave the default absent, and the hardware complete, on a port that
/// accepts more than one terminal configuration and takes no voltage or current
/// channel.
TEST(ProbeTest, testNoDefaultWithoutVoltageOrCurrent) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(1, RSE_DIFF, {DAQmx_Val_Bridge});
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 1);
    EXPECT_EQ(hw.analog_inputs[0].default_terminal_config, std::nullopt);
    EXPECT_TRUE(is_complete(hw));
    EXPECT_TRUE(api->calls_to("CreateTask").empty());
}

/// @brief it should map DAQmx types to channel types and drop those with no channel
/// type.
TEST(ProbeTest, testMapsChannelTypes) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(
        1,
        DAQmx_Val_Bit_TermCfg_Diff,
        {DAQmx_Val_Voltage,
         DAQmx_Val_Rosette_Strain_Gage,
         DAQmx_Val_Temp_Thrmstr,
         DAQmx_Val_Position_LVDT}
    );
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.analog_inputs.size(), 1);
    EXPECT_EQ(
        hw.analog_inputs[0].types,
        std::vector<std::string>(
            {"ai_voltage", "ai_thermistor_iex", "ai_thermistor_vex"}
        )
    );
}

/// @brief it should read the counters, the counter input types, and the digital ports,
/// merging a port that reads and writes.
TEST(ProbeTest, testReadsCountersAndDigitalPorts) {
    auto api = std::make_shared<FakeAPI>();
    api->strings[DAQmx_Dev_CI_PhysicalChans] = "Dev1/ctr0, Dev1/ctr1";
    api->strings[DAQmx_Dev_CO_PhysicalChans] = "Dev1/ctr0";
    api->channel_arrays["Dev1/ctr0"][DAQmx_PhysicalChan_CI_SupportedMeasTypes] = {
        DAQmx_Val_Freq,
        DAQmx_Val_Pulse_Freq,
    };
    api->strings[DAQmx_Dev_DI_Ports] = "Dev1/port0, Dev1/port1";
    api->strings[DAQmx_Dev_DO_Ports] = "Dev1/port0, Dev1/port2";
    api->channel_uints["Dev1/port0"][DAQmx_PhysicalChan_DI_PortWidth] = 32;
    api->channel_uints["Dev1/port0"][DAQmx_PhysicalChan_DO_PortWidth] = 32;
    api->channel_uints["Dev1/port1"][DAQmx_PhysicalChan_DI_PortWidth] = 8;
    api->channel_uints["Dev1/port2"][DAQmx_PhysicalChan_DO_PortWidth] = 8;
    const auto hw = probe_api(api);
    ASSERT_EQ(hw.counter_inputs.size(), 2);
    EXPECT_EQ(hw.counter_inputs[0].port, 0);
    EXPECT_EQ(hw.counter_inputs[0].types, std::vector<std::string>({"ci_frequency"}));
    EXPECT_EQ(hw.counter_inputs[1].port, 1);
    EXPECT_TRUE(hw.counter_inputs[1].types.empty());
    ASSERT_EQ(hw.counter_outputs.size(), 1);
    EXPECT_EQ(hw.counter_outputs[0].port, 0);
    EXPECT_TRUE(hw.counter_outputs[0].types.empty());
    ASSERT_EQ(hw.digital_ports.size(), 3);
    EXPECT_EQ(hw.digital_ports[0].port, 0);
    EXPECT_EQ(hw.digital_ports[0].lines, 32);
    EXPECT_TRUE(hw.digital_ports[0].input);
    EXPECT_TRUE(hw.digital_ports[0].output);
    EXPECT_EQ(hw.digital_ports[1].port, 1);
    EXPECT_TRUE(hw.digital_ports[1].input);
    EXPECT_FALSE(hw.digital_ports[1].output);
    EXPECT_EQ(hw.digital_ports[2].port, 2);
    EXPECT_FALSE(hw.digital_ports[2].input);
    EXPECT_TRUE(hw.digital_ports[2].output);
}

/// @brief it should read the rates and ranges of each subsystem that has ports.
TEST(ProbeTest, testReadsRatesAndRanges) {
    auto api = std::make_shared<FakeAPI>();
    api->add_analog_inputs(1, DAQmx_Val_Bit_TermCfg_Diff, {DAQmx_Val_Current});
    api->strings[DAQmx_Dev_AO_PhysicalChans] = "Dev1/ao0";
    api->channel_ints["Dev1/ao0"]
                     [DAQmx_PhysicalChan_AO_TermCfgs] = DAQmx_Val_Bit_TermCfg_RSE;
    api->channel_arrays["Dev1/ao0"][DAQmx_PhysicalChan_AO_SupportedOutputTypes] = {
        DAQmx_Val_Voltage,
    };
    api->doubles[DAQmx_Dev_AI_MinRate] = 0.1;
    api->doubles[DAQmx_Dev_AI_MaxSingleChanRate] = 250000;
    api->doubles[DAQmx_Dev_AI_MaxMultiChanRate] = 250000;
    api->doubles[DAQmx_Dev_AO_MaxRate] = 833333;
    api->double_arrays[DAQmx_Dev_AI_CurrentRngs] = {-0.02, 0.02};
    api->double_arrays[DAQmx_Dev_AO_VoltageRngs] = {-10, 10};
    const auto hw = probe_api(api);
    EXPECT_EQ(hw.ai_min_rate, 0.1);
    EXPECT_EQ(hw.ai_max_single_channel_rate, 250000);
    EXPECT_EQ(hw.ai_max_multi_channel_rate, 250000);
    EXPECT_EQ(hw.ao_max_rate, 833333);
    EXPECT_TRUE(hw.ai_voltage_ranges.empty());
    ASSERT_EQ(hw.ai_current_ranges.size(), 1);
    EXPECT_EQ(hw.ai_current_ranges[0].lower, -0.02);
    EXPECT_EQ(hw.ai_current_ranges[0].upper, 0.02);
    ASSERT_EQ(hw.ao_voltage_ranges.size(), 1);
    EXPECT_EQ(hw.ao_voltage_ranges[0].upper, 10);
    ASSERT_EQ(hw.analog_outputs.size(), 1);
    EXPECT_EQ(hw.analog_outputs[0].terminal_configs, std::vector<std::string>({"RSE"}));
    EXPECT_EQ(hw.analog_outputs[0].types, std::vector<std::string>({"ao_voltage"}));
}

/// @brief it should give a chassis empty lists and strip the device name from its
/// terminals.
TEST(ProbeTest, testReadsChassis) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = DAQmx_Val_CompactDAQChassis;
    api->strings[DAQmx_Dev_ProductType] = "cDAQ-9178";
    api->strings[DAQmx_Dev_Terminals] = "/Dev1/PFI0, /Dev1/PFI1, /Dev1/20MHzTimebase";
    const auto hw = probe_api(api);
    EXPECT_EQ(hw.product_category, "compact_daq_chassis");
    EXPECT_TRUE(hw.analog_inputs.empty());
    EXPECT_TRUE(hw.analog_outputs.empty());
    EXPECT_TRUE(hw.counter_inputs.empty());
    EXPECT_TRUE(hw.counter_outputs.empty());
    EXPECT_TRUE(hw.digital_ports.empty());
    EXPECT_EQ(hw.ai_min_rate, std::nullopt);
    EXPECT_EQ(
        hw.terminals,
        std::vector<std::string>({"PFI0", "PFI1", "20MHzTimebase"})
    );
}

/// @brief it should name a category it does not know "unknown".
TEST(ProbeTest, testNamesUnknownCategory) {
    auto api = std::make_shared<FakeAPI>();
    api->ints[DAQmx_Dev_ProductCategory] = 99999;
    EXPECT_EQ(probe_api(api).product_category, "unknown");
}

/// @brief it should fail when DAQmx fails with an error other than an unsupported
/// attribute.
TEST(ProbeTest, testFailsOnOtherErrors) {
    auto api = std::make_shared<FakeAPI>();
    api->failures[DAQmx_Dev_AI_PhysicalChans] = DAQmxErrorInvalidDeviceID;
    daqmx::SugaredAPI dmx(api);
    const auto [hw, err] = probe(dmx, DEVICE);
    EXPECT_EQ(err.type, daqmx::CRITICAL_ERROR.sub("200220").type);
    EXPECT_EQ(err.data, ERROR_MESSAGE);
}
}
