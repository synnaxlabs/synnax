// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"
#include "nlohmann/json.hpp"

#include "x/cpp/json/json.h"
#include "x/cpp/test/test.h"

#include "driver/ni/channel/channels.h"
#include "driver/ni/daqmx/fake.h"

namespace driver::ni::channel {
namespace {
std::shared_ptr<daqmx::SugaredAPI> fake_dmx(std::shared_ptr<daqmx::FakeAPI> &fake) {
    fake = std::make_shared<daqmx::FakeAPI>();
    return std::make_shared<daqmx::SugaredAPI>(fake);
}
}

TEST(ChannelsTest, ParseAIAccelChan) {
    x::json::json j = {
        {"type", "ai_accel"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"sensitivity", 0},
        {"current_excit_source", "Internal"},
        {"current_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"units", "g"},
        {"sensitivity_units", "mVoltsPerG"},
        {"device", "cDAQ1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIAccelChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["sensitivity"], 0);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_g);
}

TEST(ChannelsTest, ParseAIBridgeChan) {
    x::json::json j = {
        {"type", "ai_bridge"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 1},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"min_val", 0},
        {"max_val", 1},
        {"units", "mVoltsPerVolt"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIBridgeChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAICurrentChan) {
    x::json::json j = {
        {"type", "ai_current"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Amps"},
        {"shunt_resistor_loc", "Default"},
        {"ext_shunt_resistor_val", 1},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAICurrentChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["shuntResistorLoc"], DAQmx_Val_Default);
    EXPECT_EQ(call["args"]["extShuntResistorVal"], 1);
}

TEST(ChannelsTest, ParseAIForceBridgeTableChan) {
    x::json::json j = {
        {"type", "ai_force_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"electrical_vals", {1, 2, 3}},
        {"electrical_units", "mVoltsPerVolt"},
        {"physical_vals", {1, 2, 3}},
        {"physical_units", "Newtons"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Newtons"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIForceBridgeTableChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAIForceBridgeTwoPointLinChan) {
    x::json::json j = {
        {"type", "ai_force_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"first_electrical_val", 0},
        {"second_electrical_val", 1},
        {"electrical_units", "mVoltsPerVolt"},
        {"first_physical_val", 0},
        {"second_physical_val", 1},
        {"physical_units", "Newtons"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Newtons"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIForceBridgeTwoPointLinChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAIForceIEPEChan) {
    x::json::json j = {
        {"type", "ai_force_iepe"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"sensitivity", 0},
        {"current_excit_source", "Internal"},
        {"current_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Newtons"},
        {"sensitivity_units", "mVoltsPerNewton"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIForceIEPEChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["sensitivity"], 0);
}

TEST(ChannelsTest, ParseAIMicrophoneChan) {
    x::json::json j = {
        {"type", "ai_microphone"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"current_excit_source", "Internal"},
        {"current_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Pascals"},
        {"mic_sensitivity", 0},
        {"max_snd_press_level", 0},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIMicrophoneChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["micSensitivity"], 0);
    EXPECT_EQ(call["args"]["maxSndPressLevel"], 0);
}

TEST(ChannelsTest, ParseAIPressureBridgeTableChan) {
    x::json::json j = {
        {"type", "ai_pressure_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"electrical_vals", {1, 2}},
        {"electrical_units", "mVoltsPerVolt"},
        {"physical_vals", {1, 2}},
        {"physical_units", "PoundsPerSquareInch"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Pascals"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIPressureBridgeTableChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAIPressureBridgeTwoPointLinChan) {
    x::json::json j = {
        {"type", "ai_pressure_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"first_electrical_val", 0},
        {"second_electrical_val", 1},
        {"electrical_units", "mVoltsPerVolt"},
        {"first_physical_val", 0},
        {"second_physical_val", 1},
        {"physical_units", "PoundsPerSquareInch"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Pascals"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIPressureBridgeTwoPointLinChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAIRTDChan) {
    x::json::json j = {
        {"type", "ai_rtd"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"resistance_config", "2Wire"},
        {"current_excit_source", "Internal"},
        {"current_excit_val", 0},
        {"rtd_type", "Pt3750"},
        {"r0", 0},
        {"units", "DegC"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIRTDChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["resistanceConfig"], DAQmx_Val_2Wire);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["rtdType"], DAQmx_Val_Pt3750);
    EXPECT_EQ(call["args"]["r0"], 0);
}

TEST(ChannelsTest, ParseAIStrainGaugeChan) {
    x::json::json j = {
        {"type", "ai_strain_gauge"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Strain"},
        {"strain_config", "full-bridge-I"},
        {"gage_factor", 0},
        {"initial_bridge_voltage", 0},
        {"nominal_gage_resistance", 0},
        {"poisson_ratio", 0},
        {"lead_wire_resistance", 0},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIStrainGageChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["strainConfig"], DAQmx_Val_FullBridgeI);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["gageFactor"], 0);
    EXPECT_EQ(call["args"]["nominalGageResistance"], 0);
    EXPECT_EQ(call["args"]["poissonRatio"], 0);
    EXPECT_EQ(call["args"]["leadWireResistance"], 0);
}

TEST(ChannelsTest, ParseAITempBuiltInChan) {
    x::json::json j = {
        {"type", "ai_temp_builtin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"units", "DegC"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAITempBuiltInSensorChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/_boardTempSensor_vs_aignd");
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_DegC);
}

TEST(ChannelsTest, ParseAIThermoChan) {
    x::json::json j = {
        {"channels.0",
         {{"type", "ai_thermocouple"},
          {"key", "ks1VnWdrSVA"},
          {"port", 0},
          {"disabled", false},
          {"name", ""},
          {"channel", 0},
          {"min_val", 0},
          {"max_val", 1},
          {"units", "DegC"},
          {"thermocouple_type", "J"},
          {"cjc", {{"source", "chan"}, {"port", 1}}},
          {"device", "cdaq1Mod2"}}}
    };
    x::json::Parser p(j);
    auto child = p.child("channels.0");
    const auto chan = channel::parse_input(child, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIThrmcplChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["thermocoupleType"], DAQmx_Val_J_Type_TC);
    EXPECT_EQ(call["args"]["cjcSource"], DAQmx_Val_Chan);
    EXPECT_EQ(call["args"]["cjcVal"], 0);
}

/// @brief a constant CJC source carries the reference temperature and no port.
TEST(ChannelsTest, ParseAIThermocoupleConstCJC) {
    x::json::json j = {
        {"channels.0",
         {{"type", "ai_thermocouple"},
          {"key", "ks1VnWdrSVA"},
          {"port", 0},
          {"disabled", false},
          {"name", ""},
          {"channel", 0},
          {"min_val", 0},
          {"max_val", 1},
          {"units", "DegC"},
          {"thermocouple_type", "J"},
          {"cjc", {{"source", "const_val"}, {"val", 25.5}}},
          {"device", "cdaq1Mod2"}}}
    };
    x::json::Parser p(j);
    auto child = p.child("channels.0");
    const auto chan = channel::parse_input(child, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIThrmcplChan");
    ASSERT_EQ(calls.size(), 1);
    EXPECT_EQ(calls[0]["args"]["cjcSource"], DAQmx_Val_ConstVal);
    EXPECT_EQ(calls[0]["args"]["cjcVal"], 25.5);
}

/// @brief a built-in CJC source carries neither the reference nor the port.
TEST(ChannelsTest, ParseAIThermocoupleBuiltInCJC) {
    x::json::json j = {
        {"channels.0",
         {{"type", "ai_thermocouple"},
          {"key", "ks1VnWdrSVA"},
          {"port", 0},
          {"disabled", false},
          {"name", ""},
          {"channel", 0},
          {"min_val", 0},
          {"max_val", 1},
          {"units", "DegC"},
          {"thermocouple_type", "J"},
          {"cjc", {{"source", "built_in"}}},
          {"device", "cdaq1Mod2"}}}
    };
    x::json::Parser p(j);
    auto child = p.child("channels.0");
    const auto chan = channel::parse_input(child, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIThrmcplChan");
    ASSERT_EQ(calls.size(), 1);
    EXPECT_EQ(calls[0]["args"]["cjcSource"], DAQmx_Val_BuiltIn);
    EXPECT_EQ(calls[0]["args"]["cjcVal"], 0);
}

TEST(ChannelsTest, ParseAITorqueBridgeTableChan) {
    x::json::json j = {
        {"type", "ai_torque_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"electrical_vals", {1, 2, 3}},
        {"electrical_units", "mVoltsPerVolt"},
        {"physical_vals", {1, 2, 3}},
        {"physical_units", "NewtonMeters"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "NewtonMeters"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAITorqueBridgeTableChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAITorqueBridgeTwoPointLinChan) {
    x::json::json j = {
        {"type", "ai_torque_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"bridge_config", "FullBridge"},
        {"nominal_bridge_resistance", 0},
        {"voltage_excit_source", "Internal"},
        {"voltage_excit_val", 0},
        {"first_electrical_val", 0},
        {"second_electrical_val", 1},
        {"electrical_units", "mVoltsPerVolt"},
        {"first_physical_val", 0},
        {"second_physical_val", 1},
        {"physical_units", "NewtonMeters"},
        {"custom_scale", {{"type", "none"}}},
        {"units", "NewtonMeters"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAITorqueBridgeTwoPointLinChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAIVelocityIEPEChan) {
    x::json::json j = {
        {"type", "ai_velocity_iepe"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"sensitivity", 0},
        {"current_excit_source", "Internal"},
        {"current_excit_val", 0},
        {"custom_scale", {{"type", "none"}}},
        {"units", "MetersPerSecond"},
        {"sensitivity_units", "MillivoltsPerMillimeterPerSecond"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIVelocityIEPEChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
    EXPECT_EQ(call["args"]["sensitivity"], 0);
}

TEST(ChannelsTest, ParseAIVoltageChan) {
    x::json::json j = {
        {"type", "ai_voltage"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Volts"},
        {"device", "cdaq1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_analog_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAIVoltageChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ai0");
    EXPECT_EQ(call["args"]["terminalConfig"], DAQmx_Val_Cfg_Default);
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAOVoltageChan) {
    x::json::json j = {
        {"type", "ao_voltage"},
        {"key", "XBQejNmAyaO"},
        {"port", 0},
        {"disabled", false},
        {"channel", 0},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Volts"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_output(p, "ni_analog_write");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAOVoltageChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ao0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1);
}

TEST(ChannelsTest, ParseAOFuncGenChan) {
    x::json::json j = {
        {"type", "ao_func_gen"},
        {"key", "AepqBDjsgwx"},
        {"port", 1},
        {"disabled", false},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"wave_type", "Sine"},
        {"frequency", 0},
        {"amplitude", 0},
        {"offset", 0}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_output(p, "ni_analog_write");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateAOFuncGenChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["physicalChannel"], "cDAQ1Mod2/ao1");
    EXPECT_EQ(call["args"]["type"], DAQmx_Val_Sine);
    EXPECT_EQ(call["args"]["freq"], 0);
    EXPECT_EQ(call["args"]["amplitude"], 0);
    EXPECT_EQ(call["args"]["offset"], 0);
}

TEST(ChannelsTest, ParseDIChan) {
    x::json::json j = {
        {"type", "digital_input"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"line", 1},
        {"disabled", false},
        {"channel", 0},
        {"device", "cDAQ1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_digital_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateDIChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["lines"], "cDAQ1Mod2/port0/line1");
}

TEST(ChannelsTest, ParseDOChan) {
    x::json::json j = {
        {"type", "digital_output"},
        {"key", "XBQejNmAyaO"},
        {"port", 0},
        {"line", 1},
        {"disabled", false},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"device", "cDAQ1Mod2"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_output(p, "ni_digital_write");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod2");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateDOChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["lines"], "cDAQ1Mod2/port0/line1");
}

TEST(ChannelsTest, ParseCIFrequencyChanHz) {
    x::json::json j = {
        {"type", "ci_frequency"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 2},
        {"max_val", 1000},
        {"units", "Hz"},
        {"edge", "Rising"},
        {"meas_method", "DynamicAvg"},
        {"meas_time", 0.001},
        {"divisor", 4},
        {"terminal", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIFreqChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["minVal"], 2);
    EXPECT_EQ(call["args"]["maxVal"], 1000);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Hz);
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Rising);
    EXPECT_EQ(call["args"]["measMethod"], DAQmx_Val_DynAvg);
    EXPECT_EQ(call["args"]["measTime"], 0.001);
    EXPECT_EQ(call["args"]["divisor"], 4);
}

TEST(ChannelsTest, ParseCIFrequencyChanTicks) {
    x::json::json j = {
        {"type", "ci_frequency"},
        {"key", "ks1VnWdrSVB"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 100},
        {"max_val", 10000},
        {"units", "Ticks"},
        {"edge", "Falling"},
        {"meas_method", "LowFreq1Ctr"},
        {"meas_time", 0.01},
        {"divisor", 1},
        {"terminal", "PFI0"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIFreqChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["minVal"], 100);
    EXPECT_EQ(call["args"]["maxVal"], 10000);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Ticks);
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Falling);
    EXPECT_EQ(call["args"]["measMethod"], DAQmx_Val_LowFreq1Ctr);
    EXPECT_EQ(call["args"]["measTime"], 0.01);
    EXPECT_EQ(call["args"]["divisor"], 1);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIEdgeCountChanRising) {
    x::json::json j = {
        {"type", "ci_edge_count"},
        {"key", "ks1VnWdrSVC"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"active_edge", "Rising"},
        {"count_direction", "CountUp"},
        {"initial_count", 0},
        {"terminal", ""},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCICountEdgesChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Rising);
    EXPECT_EQ(call["args"]["countDirection"], DAQmx_Val_CountUp);
    EXPECT_EQ(call["args"]["initialCount"], 0);
}

TEST(ChannelsTest, ParseCIEdgeCountChanFalling) {
    x::json::json j = {
        {"type", "ci_edge_count"},
        {"key", "ks1VnWdrSVD"},
        {"port", 2},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"active_edge", "Falling"},
        {"count_direction", "CountDown"},
        {"initial_count", 100},
        {"terminal", "PFI11"},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCICountEdgesChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr2");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Falling);
    EXPECT_EQ(call["args"]["countDirection"], DAQmx_Val_CountDown);
    EXPECT_EQ(call["args"]["initialCount"], 100);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIPeriodChanSeconds) {
    x::json::json j = {
        {"type", "ci_period"},
        {"key", "ks1VnWdrSVE"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Seconds"},
        {"starting_edge", "Rising"},
        {"meas_method", "DynamicAvg"},
        {"meas_time", 0.001},
        {"divisor", 4},
        {"terminal", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIPeriodChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Rising);
    EXPECT_EQ(call["args"]["measMethod"], DAQmx_Val_DynAvg);
}

TEST(ChannelsTest, ParseCIPeriodChanTicks) {
    x::json::json j = {
        {"type", "ci_period"},
        {"key", "ks1VnWdrSVF"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Ticks"},
        {"starting_edge", "Falling"},
        {"meas_method", "LowFreq1Ctr"},
        {"meas_time", 0.001},
        {"divisor", 4},
        {"terminal", "PFI5"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIPeriodChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Falling);
    EXPECT_EQ(call["args"]["measMethod"], DAQmx_Val_LowFreq1Ctr);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIPulseWidthChanSeconds) {
    x::json::json j = {
        {"type", "ci_pulse_width"},
        {"key", "ks1VnWdrSVG"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Seconds"},
        {"starting_edge", "Rising"},
        {"terminal", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIPulseWidthChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
}

TEST(ChannelsTest, ParseCIPulseWidthChanTicks) {
    x::json::json j = {
        {"type", "ci_pulse_width"},
        {"key", "ks1VnWdrSVH"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Ticks"},
        {"starting_edge", "Falling"},
        {"terminal", "PFI9"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIPulseWidthChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCISemiPeriodChanSeconds) {
    x::json::json j = {
        {"type", "ci_semi_period"},
        {"key", "ks1VnWdrSVI"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Seconds"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCISemiPeriodChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
}

TEST(ChannelsTest, ParseCISemiPeriodChanTicks) {
    x::json::json j = {
        {"type", "ci_semi_period"},
        {"key", "ks1VnWdrSVJ"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 0.1},
        {"units", "Ticks"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCISemiPeriodChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
}

TEST(ChannelsTest, ParseCITwoEdgeSepChanSeconds) {
    x::json::json j = {
        {"type", "ci_two_edge_sep"},
        {"key", "ks1VnWdrSVK"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 1},
        {"units", "Seconds"},
        {"first_edge", "Rising"},
        {"second_edge", "Falling"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCITwoEdgeSepChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["firstEdge"], DAQmx_Val_Rising);
    EXPECT_EQ(call["args"]["secondEdge"], DAQmx_Val_Falling);
}

TEST(ChannelsTest, ParseCITwoEdgeSepChanTicks) {
    x::json::json j = {
        {"type", "ci_two_edge_sep"},
        {"key", "ks1VnWdrSVL"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0.000001},
        {"max_val", 1},
        {"units", "Ticks"},
        {"first_edge", "Falling"},
        {"second_edge", "Rising"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCITwoEdgeSepChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["firstEdge"], DAQmx_Val_Falling);
    EXPECT_EQ(call["args"]["secondEdge"], DAQmx_Val_Rising);
}

TEST(ChannelsTest, ParseCILinearVelocityChanMetersPerSecond) {
    x::json::json j = {
        {"type", "ci_velocity_linear"},
        {"key", "ks1VnWdrSVW"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 10},
        {"units", "m/s"},
        {"decoding_type", "X4"},
        {"dist_per_pulse", 0.001},
        {"terminal_a", "PFI0"},
        {"terminal_b", "PFI1"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCILinVelocityChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 10);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_MetersPerSecond);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X4);
    EXPECT_EQ(call["args"]["distPerPulse"], 0.001);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCILinearVelocityChanInchesPerSecond) {
    x::json::json j = {
        {"type", "ci_velocity_linear"},
        {"key", "ks1VnWdrSVX"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 100},
        {"units", "in/s"},
        {"decoding_type", "X2"},
        {"dist_per_pulse", 0.01},
        {"terminal_a", ""},
        {"terminal_b", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCILinVelocityChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 100);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_InchesPerSecond);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X2);
    EXPECT_EQ(call["args"]["distPerPulse"], 0.01);
}

TEST(ChannelsTest, ParseCIAngularVelocityChanRPM) {
    x::json::json j = {
        {"type", "ci_velocity_angular"},
        {"key", "ks1VnWdrSVY"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 1000},
        {"units", "RPM"},
        {"decoding_type", "X4"},
        {"pulses_per_rev", 24},
        {"terminal_a", "PFI2"},
        {"terminal_b", "PFI3"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIAngVelocityChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 1000);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_RPM);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X4);
    EXPECT_EQ(call["args"]["pulsesPerRev"], 24);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIAngularVelocityChanRadiansPerSecond) {
    x::json::json j = {
        {"type", "ci_velocity_angular"},
        {"key", "ks1VnWdrSVZ"},
        {"port", 2},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 0},
        {"max_val", 100},
        {"units", "Radians/s"},
        {"decoding_type", "X1"},
        {"pulses_per_rev", 100},
        {"terminal_a", ""},
        {"terminal_b", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIAngVelocityChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr2");
    EXPECT_EQ(call["args"]["minVal"], 0);
    EXPECT_EQ(call["args"]["maxVal"], 100);
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_RadiansPerSecond);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X1);
    EXPECT_EQ(call["args"]["pulsesPerRev"], 100);
}

TEST(ChannelsTest, ParseCILinearPositionChanMeters) {
    x::json::json j = {
        {"type", "ci_position_linear"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", -10},
        {"max_val", 10},
        {"units", "Meters"},
        {"decoding_type", "X4"},
        {"dist_per_pulse", 0.001},
        {"initial_pos", 0.0},
        {"z_index_enable", true},
        {"z_index_val", 0.0},
        {"z_index_phase", "AHighBHigh"},
        {"terminal_a", "PFI0"},
        {"terminal_b", "PFI1"},
        {"terminal_z", "PFI2"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCILinEncoderChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Meters);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X4);
    EXPECT_EQ(call["args"]["distPerPulse"], 0.001);
    EXPECT_EQ(call["args"]["initialPos"], 0.0);
    EXPECT_EQ(call["args"]["zidxEnable"], 1);
    EXPECT_EQ(call["args"]["zidxVal"], 0.0);
    EXPECT_EQ(call["args"]["zidxPhase"], DAQmx_Val_AHighBHigh);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCILinearPositionChanInches) {
    x::json::json j = {
        {"type", "ci_position_linear"},
        {"key", "ks1VnWdrSVB"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", -100},
        {"max_val", 100},
        {"units", "Inches"},
        {"decoding_type", "X2"},
        {"dist_per_pulse", 0.01},
        {"initial_pos", 5.0},
        {"z_index_enable", false},
        {"z_index_val", 0.0},
        {"z_index_phase", "AHighBLow"},
        {"terminal_a", ""},
        {"terminal_b", ""},
        {"terminal_z", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCILinEncoderChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Inches);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X2);
    EXPECT_EQ(call["args"]["distPerPulse"], 0.01);
    EXPECT_EQ(call["args"]["initialPos"], 5.0);
    EXPECT_EQ(call["args"]["zidxEnable"], 0);
    EXPECT_EQ(call["args"]["zidxVal"], 0.0);
    EXPECT_EQ(call["args"]["zidxPhase"], DAQmx_Val_AHighBLow);
}

TEST(ChannelsTest, ParseCIAngularPositionChanDegrees) {
    x::json::json j = {
        {"type", "ci_position_angular"},
        {"key", "ks1VnWdrSVC"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", -180},
        {"max_val", 180},
        {"units", "Degrees"},
        {"decoding_type", "X4"},
        {"pulses_per_rev", 24},
        {"initial_angle", 0.0},
        {"z_index_enable", true},
        {"z_index_val", 0.0},
        {"z_index_phase", "AHighBHigh"},
        {"terminal_a", "PFI10"},
        {"terminal_b", "PFI12"},
        {"terminal_z", "PFI11"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIAngEncoderChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Degrees);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X4);
    EXPECT_EQ(call["args"]["pulsesPerRev"], 24);
    EXPECT_EQ(call["args"]["initialAngle"], 0.0);
    EXPECT_EQ(call["args"]["zidxEnable"], 1);
    EXPECT_EQ(call["args"]["zidxVal"], 0.0);
    EXPECT_EQ(call["args"]["zidxPhase"], DAQmx_Val_AHighBHigh);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIAngularPositionChanRadians) {
    x::json::json j = {
        {"type", "ci_position_angular"},
        {"key", "ks1VnWdrSVD"},
        {"port", 2},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", -3.14},
        {"max_val", 3.14},
        {"units", "Radians"},
        {"decoding_type", "X1"},
        {"pulses_per_rev", 100},
        {"initial_angle", 1.57},
        {"z_index_enable", false},
        {"z_index_val", 0.0},
        {"z_index_phase", "ALowBLow"},
        {"terminal_a", ""},
        {"terminal_b", ""},
        {"terminal_z", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIAngEncoderChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr2");
    EXPECT_EQ(call["args"]["units"], DAQmx_Val_Radians);
    EXPECT_EQ(call["args"]["decodingType"], DAQmx_Val_X1);
    EXPECT_EQ(call["args"]["pulsesPerRev"], 100);
    EXPECT_EQ(call["args"]["initialAngle"], 1.57);
    EXPECT_EQ(call["args"]["zidxEnable"], 0);
    EXPECT_EQ(call["args"]["zidxVal"], 0.0);
    EXPECT_EQ(call["args"]["zidxPhase"], DAQmx_Val_ALowBLow);
}

TEST(ChannelsTest, ParseCIDutyCycleChanRising) {
    x::json::json j = {
        {"type", "ci_duty_cycle"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 2},
        {"max_val", 10000},
        {"active_edge", "Rising"},
        {"terminal", "PFI0"},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIDutyCycleChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr0");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Rising);
    EXPECT_EQ(fake->calls_to("SetChanAttributeString").size() >= 1, true);
}

TEST(ChannelsTest, ParseCIDutyCycleChanFalling) {
    x::json::json j = {
        {"type", "ci_duty_cycle"},
        {"key", "ks1VnWdrSVB"},
        {"port", 1},
        {"disabled", false},
        {"name", ""},
        {"channel", 0},
        {"min_val", 10},
        {"max_val", 5000},
        {"active_edge", "Falling"},
        {"terminal", ""},
        {"custom_scale", {{"type", "none"}}},
        {"device", "cDAQ1Mod3"}
    };
    x::json::Parser p(j);
    const auto chan = channel::parse_input(p, "ni_counter_read");
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    EXPECT_EQ(chan->enabled, true);
    chan->bind_remote_info(synnax::channel::Channel(), "cDAQ1Mod3");
    std::shared_ptr<daqmx::FakeAPI> fake;
    const auto dmx = fake_dmx(fake);
    ASSERT_NIL(chan->apply(dmx, nullptr));
    const auto calls = fake->calls_to("CreateCIDutyCycleChan");
    ASSERT_EQ(calls.size(), 1);
    const auto &call = calls[0];
    EXPECT_EQ(call["args"]["counter"], "cDAQ1Mod3/ctr1");
    EXPECT_EQ(call["args"]["edge"], DAQmx_Val_Falling);
}
}
