// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xjson/xjson.h"

#include "driver/ni/channel/channels.h"

using json = nlohmann::json;

TEST(ChannelsTest, ParseAIAccelChan) {
    json j = {
        {"type", "ai_accel"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto accel_chan = dynamic_cast<channel::AIAccel *>(chan.get());
    ASSERT_NE(accel_chan, nullptr);
    EXPECT_EQ(accel_chan->enabled, true);
    EXPECT_EQ(accel_chan->port, 0);
    EXPECT_EQ(accel_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(accel_chan->min_val, 0);
    EXPECT_EQ(accel_chan->max_val, 1);
    EXPECT_EQ(accel_chan->sensitivity, 0);
    EXPECT_EQ(accel_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(accel_chan->excitation_config.val, 0);
    EXPECT_EQ(accel_chan->units, DAQmx_Val_g);
    accel_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(accel_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIBridgeChan) {
    json j = {
        {"type", "ai_bridge"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto bridge_chan = dynamic_cast<channel::AIBridge *>(chan.get());
    ASSERT_NE(bridge_chan, nullptr);
    EXPECT_EQ(bridge_chan->bridge_config.ni_bridge_config, DAQmx_Val_FullBridge);
    EXPECT_EQ(bridge_chan->min_val, 0);
    EXPECT_EQ(bridge_chan->max_val, 1);
    EXPECT_EQ(bridge_chan->bridge_config.nominal_bridge_resistance, 1);
    bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAICurrentChan) {
    json j = {
        {"type", "ai_current"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto current_chan = dynamic_cast<channel::AICurrent *>(chan.get());
    ASSERT_NE(current_chan, nullptr);
    EXPECT_EQ(current_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(current_chan->min_val, 0);
    EXPECT_EQ(current_chan->max_val, 1);
    EXPECT_EQ(current_chan->shunt_resistor_loc, DAQmx_Val_Default);
    EXPECT_EQ(current_chan->ext_shunt_resistor_val, 1);
    current_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(current_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIForceBridgeTableChan) {
    json j = {
        {"type", "ai_force_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto force_bridge_chan = dynamic_cast<channel::AIForceBridgeTable *>(
        chan.get()
    );
    ASSERT_NE(force_bridge_chan, nullptr);
    EXPECT_EQ(force_bridge_chan->bridge_config.ni_bridge_config, DAQmx_Val_FullBridge);
    EXPECT_EQ(force_bridge_chan->min_val, 0);
    EXPECT_EQ(force_bridge_chan->max_val, 1);
    EXPECT_EQ(force_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(
        force_bridge_chan->bridge_config.voltage_excit_source,
        DAQmx_Val_Internal
    );
    EXPECT_EQ(force_bridge_chan->bridge_config.voltage_excit_val, 0);
    EXPECT_EQ(force_bridge_chan->table_config.electrical_vals[0], 1);
    EXPECT_EQ(force_bridge_chan->table_config.electrical_vals[1], 2);
    force_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(force_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIForceBridgeTwoPointLinChan) {
    json j = {
        {"type", "ai_force_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto force_bridge_chan = dynamic_cast<channel::AIForceBridgeTwoPointLin *>(
        chan.get()
    );
    ASSERT_NE(force_bridge_chan, nullptr);
    EXPECT_EQ(force_bridge_chan->bridge_config.ni_bridge_config, DAQmx_Val_FullBridge);
    EXPECT_EQ(force_bridge_chan->min_val, 0);
    EXPECT_EQ(force_bridge_chan->max_val, 1);
    EXPECT_EQ(force_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(force_bridge_chan->two_point_lin_config.first_electrical_val, 0);
    EXPECT_EQ(force_bridge_chan->two_point_lin_config.second_electrical_val, 1);
    EXPECT_EQ(force_bridge_chan->two_point_lin_config.first_physical_val, 0);
    EXPECT_EQ(force_bridge_chan->two_point_lin_config.second_physical_val, 1);
    force_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(force_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIForceIEPEChan) {
    json j = {
        {"type", "ai_force_iepe"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto force_iepe_chan = dynamic_cast<channel::AIForceIEPE *>(chan.get());
    ASSERT_NE(force_iepe_chan, nullptr);
    EXPECT_EQ(force_iepe_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(force_iepe_chan->min_val, 0);
    EXPECT_EQ(force_iepe_chan->max_val, 1);
    EXPECT_EQ(force_iepe_chan->sensitivity, 0);
    EXPECT_EQ(force_iepe_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(force_iepe_chan->excitation_config.val, 0);
    force_iepe_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(force_iepe_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIMicrophoneChan) {
    json j = {
        {"type", "ai_microphone"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto mic_chan = dynamic_cast<channel::AIMicrophone *>(chan.get());
    ASSERT_NE(mic_chan, nullptr);
    EXPECT_EQ(mic_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(mic_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(mic_chan->excitation_config.val, 0);
    EXPECT_EQ(mic_chan->mic_sensitivity, 0);
    EXPECT_EQ(mic_chan->max_snd_press_level, 0);
    mic_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(mic_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIPressureBridgeTableChan) {
    json j = {
        {"type", "ai_pressure_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto pressure_bridge_chan = dynamic_cast<channel::AIPressureBridgeTable *>(
        chan.get()
    );
    ASSERT_NE(pressure_bridge_chan, nullptr);
    EXPECT_EQ(
        pressure_bridge_chan->bridge_config.ni_bridge_config,
        DAQmx_Val_FullBridge
    );
    EXPECT_EQ(pressure_bridge_chan->min_val, 0);
    EXPECT_EQ(pressure_bridge_chan->max_val, 1);
    EXPECT_EQ(pressure_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(
        pressure_bridge_chan->bridge_config.voltage_excit_source,
        DAQmx_Val_Internal
    );
    EXPECT_EQ(pressure_bridge_chan->bridge_config.voltage_excit_val, 0);
    pressure_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(pressure_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIPressureBridgeTwoPointLinChan) {
    json j = {
        {"type", "ai_pressure_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto
        pressure_bridge_chan = dynamic_cast<channel::AIPressureBridgeTwoPointLin *>(
            chan.get()
        );
    ASSERT_NE(pressure_bridge_chan, nullptr);
    EXPECT_EQ(
        pressure_bridge_chan->bridge_config.ni_bridge_config,
        DAQmx_Val_FullBridge
    );
    EXPECT_EQ(pressure_bridge_chan->min_val, 0);
    EXPECT_EQ(pressure_bridge_chan->max_val, 1);
    EXPECT_EQ(pressure_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(pressure_bridge_chan->two_point_lin_config.first_electrical_val, 0);
    EXPECT_EQ(pressure_bridge_chan->two_point_lin_config.second_electrical_val, 1);
    EXPECT_EQ(pressure_bridge_chan->two_point_lin_config.first_physical_val, 0);
    EXPECT_EQ(pressure_bridge_chan->two_point_lin_config.second_physical_val, 1);
    pressure_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(pressure_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIRTDChan) {
    json j = {
        {"type", "ai_rtd"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto rtd_chan = dynamic_cast<channel::AIRTD *>(chan.get());
    ASSERT_NE(rtd_chan, nullptr);
    EXPECT_EQ(rtd_chan->resistance_config, DAQmx_Val_2Wire);
    EXPECT_EQ(rtd_chan->min_val, 0);
    EXPECT_EQ(rtd_chan->max_val, 1);
    EXPECT_EQ(rtd_chan->rtd_type, DAQmx_Val_Pt3750);
    EXPECT_EQ(rtd_chan->r0, 0);
    EXPECT_EQ(rtd_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(rtd_chan->excitation_config.val, 0);
    rtd_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(rtd_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIStrainGaugeChan) {
    json j = {
        {"type", "ai_strain_gauge"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto strain_chan = dynamic_cast<channel::AIStrainGauge *>(chan.get());
    ASSERT_NE(strain_chan, nullptr);
    EXPECT_EQ(strain_chan->strain_config, DAQmx_Val_FullBridgeI);
    EXPECT_EQ(strain_chan->min_val, 0);
    EXPECT_EQ(strain_chan->max_val, 1);
    EXPECT_EQ(strain_chan->gage_factor, 0);
    EXPECT_EQ(strain_chan->initial_bridge_voltage, 0);
    EXPECT_EQ(strain_chan->nominal_gage_resistance, 0);
    EXPECT_EQ(strain_chan->poisson_ratio, 0);
    EXPECT_EQ(strain_chan->lead_wire_resistance, 0);
    EXPECT_EQ(strain_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(strain_chan->excitation_config.val, 0);
    strain_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(strain_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAITempBuiltInChan) {
    json j = {
        {"type", "ai_temp_builtin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
        {"name", ""},
        {"channel", 0},
        {"units", "DegC"},
        {"device", "cdaq1Mod2"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto temp_chan = dynamic_cast<channel::AITempBuiltIn *>(chan.get());
    ASSERT_NE(temp_chan, nullptr);
    EXPECT_EQ(temp_chan->units, DAQmx_Val_DegC);
    temp_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(temp_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIThermoChan) {
    json j = {
        {"channels.0",
         {{"type", "ai_thermocouple"},
          {"key", "ks1VnWdrSVA"},
          {"port", 0},
          {"enabled", true},
          {"name", ""},
          {"channel", 0},
          {"min_val", 0},
          {"max_val", 1},
          {"units", "DegC"},
          {"thermocouple_type", "J"},
          {"cjc_source", "Chan"},
          {"cjc_val", 0},
          {"cjc_port", 1},
          {"device", "cdaq1Mod2"}}}
    };

    xjson::Parser p(j);
    auto child = p.child("channels.0");
    const auto chan = channel::parse_input(child);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto thermo_chan = dynamic_cast<channel::AIThermocouple *>(chan.get());
    ASSERT_NE(thermo_chan, nullptr);
    EXPECT_EQ(thermo_chan->thermocouple_type, DAQmx_Val_J_Type_TC);
    EXPECT_EQ(thermo_chan->cjc_source, DAQmx_Val_Chan);
    EXPECT_EQ(thermo_chan->cjc_val, 0);
    EXPECT_EQ(thermo_chan->cjc_port, "channels_1");
    EXPECT_EQ(thermo_chan->min_val, 0);
    EXPECT_EQ(thermo_chan->max_val, 1);
    thermo_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(thermo_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAITorqueBridgeTableChan) {
    json j = {
        {"type", "ai_torque_bridge_table"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto torque_bridge_chan = dynamic_cast<channel::AITorqueBridgeTable *>(
        chan.get()
    );
    ASSERT_NE(torque_bridge_chan, nullptr);
    EXPECT_EQ(torque_bridge_chan->bridge_config.ni_bridge_config, DAQmx_Val_FullBridge);
    EXPECT_EQ(torque_bridge_chan->min_val, 0);
    EXPECT_EQ(torque_bridge_chan->max_val, 1);
    EXPECT_EQ(torque_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(
        torque_bridge_chan->bridge_config.voltage_excit_source,
        DAQmx_Val_Internal
    );
    EXPECT_EQ(torque_bridge_chan->bridge_config.voltage_excit_val, 0);
    torque_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(torque_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAITorqueBridgeTwoPointLinChan) {
    json j = {
        {"type", "ai_torque_bridge_two_point_lin"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto torque_bridge_chan = dynamic_cast<channel::AITorqueBridgeTwoPointLin *>(
        chan.get()
    );
    ASSERT_NE(torque_bridge_chan, nullptr);
    EXPECT_EQ(torque_bridge_chan->bridge_config.ni_bridge_config, DAQmx_Val_FullBridge);
    EXPECT_EQ(torque_bridge_chan->min_val, 0);
    EXPECT_EQ(torque_bridge_chan->max_val, 1);
    EXPECT_EQ(torque_bridge_chan->bridge_config.nominal_bridge_resistance, 0);
    EXPECT_EQ(torque_bridge_chan->two_point_lin_config.first_electrical_val, 0);
    EXPECT_EQ(torque_bridge_chan->two_point_lin_config.second_electrical_val, 1);
    EXPECT_EQ(torque_bridge_chan->two_point_lin_config.first_physical_val, 0);
    EXPECT_EQ(torque_bridge_chan->two_point_lin_config.second_physical_val, 1);
    torque_bridge_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(torque_bridge_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIVelocityIEPEChan) {
    json j = {
        {"type", "ai_velocity_iepe"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto velocity_chan = dynamic_cast<channel::AIVelocityIEPE *>(chan.get());
    ASSERT_NE(velocity_chan, nullptr);
    EXPECT_EQ(velocity_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(velocity_chan->min_val, 0);
    EXPECT_EQ(velocity_chan->max_val, 1);
    EXPECT_EQ(velocity_chan->sensitivity, 0);
    EXPECT_EQ(velocity_chan->excitation_config.source, DAQmx_Val_Internal);
    EXPECT_EQ(velocity_chan->excitation_config.val, 0);
    velocity_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(velocity_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAIVoltageChan) {
    json j = {
        {"type", "ai_voltage"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
        {"name", ""},
        {"channel", 0},
        {"terminal_config", "Cfg_Default"},
        {"min_val", 0},
        {"max_val", 1},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Volts"},
        {"device", "cdaq1Mod2"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto voltage_chan = dynamic_cast<channel::AIVoltage *>(chan.get());
    ASSERT_NE(voltage_chan, nullptr);
    EXPECT_EQ(voltage_chan->terminal_config, DAQmx_Val_Cfg_Default);
    EXPECT_EQ(voltage_chan->min_val, 0);
    EXPECT_EQ(voltage_chan->max_val, 1);
    voltage_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(voltage_chan->loc(), "cDAQ1Mod2/ai0");
}

TEST(ChannelsTest, ParseAOVoltageChan) {
    json j = {
        {"type", "ao_voltage"},
        {"key", "XBQejNmAyaO"},
        {"port", 0},
        {"enabled", true},
        {"channel", 0},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"min_val", 0},
        {"max_val", 1},
        {"custom_scale", {{"type", "none"}}},
        {"units", "Volts"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_output(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto voltage_chan = dynamic_cast<channel::AOVoltage *>(chan.get());
    ASSERT_NE(voltage_chan, nullptr);
    EXPECT_EQ(voltage_chan->min_val, 0);
    EXPECT_EQ(voltage_chan->max_val, 1);
    voltage_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(voltage_chan->loc(), "cDAQ1Mod2/ao0");
}

TEST(ChannelsTest, ParseAOFuncGenChan) {
    json j = {
        {"type", "ao_func_gen"},
        {"key", "AepqBDjsgwx"},
        {"port", 1},
        {"enabled", true},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"wave_type", "Sine"},
        {"frequency", 0},
        {"amplitude", 0},
        {"offset", 0}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_output(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto func_gen_chan = dynamic_cast<channel::AOFunctionGenerator *>(chan.get());
    ASSERT_NE(func_gen_chan, nullptr);
    EXPECT_EQ(func_gen_chan->wave_type, DAQmx_Val_Sine);
    EXPECT_EQ(func_gen_chan->frequency, 0);
    EXPECT_EQ(func_gen_chan->amplitude, 0);
    EXPECT_EQ(func_gen_chan->offset, 0);
    func_gen_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(func_gen_chan->loc(), "cDAQ1Mod2/ao1");
}

TEST(ChannelsTest, ParseDIChan) {
    json j = {
        {"type", "digital_input"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"line", 1},
        {"enabled", true},
        {"channel", 0},
        {"device", "cDAQ1Mod2"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto di_chan = dynamic_cast<channel::DI *>(chan.get());
    ASSERT_NE(di_chan, nullptr);
    EXPECT_EQ(di_chan->port, 0);
    EXPECT_EQ(di_chan->line, 1);
    EXPECT_EQ(di_chan->enabled, true);
    di_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(di_chan->loc(), "cDAQ1Mod2/port0/line1");
}

TEST(ChannelsTest, ParseDOChan) {
    json j = {
        {"type", "digital_output"},
        {"key", "XBQejNmAyaO"},
        {"port", 0},
        {"line", 1},
        {"enabled", true},
        {"cmd_channel", 0},
        {"state_channel", 0},
        {"device", "cDAQ1Mod2"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_output(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto do_chan = dynamic_cast<channel::DO *>(chan.get());
    ASSERT_NE(do_chan, nullptr);
    EXPECT_EQ(do_chan->port, 0);
    EXPECT_EQ(do_chan->line, 1);
    EXPECT_EQ(do_chan->enabled, true);
    do_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod2");
    EXPECT_EQ(do_chan->loc(), "cDAQ1Mod2/port0/line1");
}

TEST(ChannelsTest, ParseCIFrequencyChanHz) {
    json j = {
        {"type", "ci_frequency"},
        {"key", "ks1VnWdrSVA"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_freq_chan = dynamic_cast<channel::CIFrequency *>(chan.get());
    ASSERT_NE(ci_freq_chan, nullptr);
    EXPECT_EQ(ci_freq_chan->enabled, true);
    EXPECT_EQ(ci_freq_chan->port, 0);
    EXPECT_EQ(ci_freq_chan->min_val, 2);
    EXPECT_EQ(ci_freq_chan->max_val, 1000);
    EXPECT_EQ(ci_freq_chan->units, DAQmx_Val_Hz);
    EXPECT_EQ(ci_freq_chan->edge, DAQmx_Val_Rising);
    EXPECT_EQ(ci_freq_chan->meas_method, DAQmx_Val_DynAvg);
    EXPECT_DOUBLE_EQ(ci_freq_chan->meas_time, 0.001);
    EXPECT_EQ(ci_freq_chan->divisor, 4);
    EXPECT_EQ(ci_freq_chan->terminal, "");
    ci_freq_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_freq_chan->loc(), "cDAQ1Mod3/ctr0");
}

TEST(ChannelsTest, ParseCIFrequencyChanTicks) {
    json j = {
        {"type", "ci_frequency"},
        {"key", "ks1VnWdrSVB"},
        {"port", 1},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_freq_chan = dynamic_cast<channel::CIFrequency *>(chan.get());
    ASSERT_NE(ci_freq_chan, nullptr);
    EXPECT_EQ(ci_freq_chan->enabled, true);
    EXPECT_EQ(ci_freq_chan->port, 1);
    EXPECT_EQ(ci_freq_chan->min_val, 100);
    EXPECT_EQ(ci_freq_chan->max_val, 10000);
    EXPECT_EQ(ci_freq_chan->units, DAQmx_Val_Ticks);
    EXPECT_EQ(ci_freq_chan->edge, DAQmx_Val_Falling);
    EXPECT_EQ(ci_freq_chan->meas_method, DAQmx_Val_LowFreq1Ctr);
    EXPECT_DOUBLE_EQ(ci_freq_chan->meas_time, 0.01);
    EXPECT_EQ(ci_freq_chan->divisor, 1);
    EXPECT_EQ(ci_freq_chan->terminal, "PFI0");
    ci_freq_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_freq_chan->loc(), "cDAQ1Mod3/ctr1");
}

TEST(ChannelsTest, ParseCIEdgeCountChanRising) {
    json j = {
        {"type", "ci_edge_count"},
        {"key", "ks1VnWdrSVC"},
        {"port", 0},
        {"enabled", true},
        {"name", ""},
        {"channel", 0},
        {"active_edge", "Rising"},
        {"count_direction", "CountUp"},
        {"initial_count", 0},
        {"terminal", ""},
        {"device", "cDAQ1Mod3"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_edge_count_chan = dynamic_cast<channel::CIEdgeCount *>(chan.get());
    ASSERT_NE(ci_edge_count_chan, nullptr);
    EXPECT_EQ(ci_edge_count_chan->enabled, true);
    EXPECT_EQ(ci_edge_count_chan->port, 0);
    EXPECT_EQ(ci_edge_count_chan->edge, DAQmx_Val_Rising);
    EXPECT_EQ(ci_edge_count_chan->count_direction, DAQmx_Val_CountUp);
    EXPECT_EQ(ci_edge_count_chan->initial_count, 0);
    EXPECT_EQ(ci_edge_count_chan->terminal, "");
    ci_edge_count_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_edge_count_chan->loc(), "cDAQ1Mod3/ctr0");
}

TEST(ChannelsTest, ParseCIEdgeCountChanFalling) {
    json j = {
        {"type", "ci_edge_count"},
        {"key", "ks1VnWdrSVD"},
        {"port", 2},
        {"enabled", true},
        {"name", ""},
        {"channel", 0},
        {"active_edge", "Falling"},
        {"count_direction", "CountDown"},
        {"initial_count", 100},
        {"terminal", "PFI11"},
        {"device", "cDAQ1Mod3"}
    };

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_edge_count_chan = dynamic_cast<channel::CIEdgeCount *>(chan.get());
    ASSERT_NE(ci_edge_count_chan, nullptr);
    EXPECT_EQ(ci_edge_count_chan->enabled, true);
    EXPECT_EQ(ci_edge_count_chan->port, 2);
    EXPECT_EQ(ci_edge_count_chan->edge, DAQmx_Val_Falling);
    EXPECT_EQ(ci_edge_count_chan->count_direction, DAQmx_Val_CountDown);
    EXPECT_EQ(ci_edge_count_chan->initial_count, 100);
    EXPECT_EQ(ci_edge_count_chan->terminal, "PFI11");
    ci_edge_count_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_edge_count_chan->loc(), "cDAQ1Mod3/ctr2");
}

TEST(ChannelsTest, ParseCIPeriodChanSeconds) {
    json j = {
        {"type", "ci_period"},
        {"key", "ks1VnWdrSVE"},
        {"port", 0},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_period_chan = dynamic_cast<channel::CIPeriod *>(chan.get());
    ASSERT_NE(ci_period_chan, nullptr);
    EXPECT_EQ(ci_period_chan->enabled, true);
    EXPECT_EQ(ci_period_chan->port, 0);
    EXPECT_EQ(ci_period_chan->edge, DAQmx_Val_Rising);
    EXPECT_EQ(ci_period_chan->meas_method, DAQmx_Val_DynAvg);
    EXPECT_EQ(ci_period_chan->terminal, "");
    ci_period_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_period_chan->loc(), "cDAQ1Mod3/ctr0");
}

TEST(ChannelsTest, ParseCIPeriodChanTicks) {
    json j = {
        {"type", "ci_period"},
        {"key", "ks1VnWdrSVF"},
        {"port", 1},
        {"enabled", true},
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

    xjson::Parser p(j);
    const auto chan = channel::parse_input(p);
    ASSERT_FALSE(p.error()) << p.error();
    ASSERT_NE(chan, nullptr);
    const auto ci_period_chan = dynamic_cast<channel::CIPeriod *>(chan.get());
    ASSERT_NE(ci_period_chan, nullptr);
    EXPECT_EQ(ci_period_chan->enabled, true);
    EXPECT_EQ(ci_period_chan->port, 1);
    EXPECT_EQ(ci_period_chan->edge, DAQmx_Val_Falling);
    EXPECT_EQ(ci_period_chan->meas_method, DAQmx_Val_LowFreq1Ctr);
    EXPECT_EQ(ci_period_chan->terminal, "PFI5");
    ci_period_chan->bind_remote_info(synnax::Channel(), "cDAQ1Mod3");
    EXPECT_EQ(ci_period_chan->loc(), "cDAQ1Mod3/ctr1");
}
