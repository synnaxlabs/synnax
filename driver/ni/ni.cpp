// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/ni.h"
#include "driver/ni/scale.h"
#include <map>
#include <regex>

static inline const std::map<std::string, std::string> FIELD_MAP = {
    {"DAQmx_AI_Max", "max_val"},
    {"DAQmx_AI_Min", "min_val"},
    {"DAQmx_AI_CustomScaleName", "custom_scale_name"},
    {"DAQmx_AI_MeasType", "meas_type"},
    {"DAQmx_AI_Voltage_Units", "voltage_units"},
    {"DAQmx_AI_Voltage_dBRef", "voltage_db_ref"},
    {"DAQmx_AI_Voltage_ACRMS_Units", "voltage_acrms_units"},
    {"DAQmx_AI_Temp_Units", "temp_units"},
    {"DAQmx_AI_Thrmcpl_Type", "thrmcpl_type"},
    {"DAQmx_AI_Thrmcpl_ScaleType", "thrmcpl_scale_type"},
    {"DAQmx_AI_Thrmcpl_CJCSrc", "cjc_source"},
    {"DAQmx_AI_Thrmcpl_CJCVal", "cjc_val"},
    {"DAQmx_AI_Thrmcpl_CJCChan", "cjc_port"},
    {"DAQmx_AI_RTD_Type", "rtd_type"},
    {"DAQmx_AI_RTD_R0", "rtd_r0"},
    {"DAQmx_AI_RTD_A", "rtd_a"},
    {"DAQmx_AI_RTD_B", "rtd_b"},
    {"DAQmx_AI_RTD_C", "rtd_c"},
    {"DAQmx_AI_Thrmstr_A", "thrmstr_a"},
    {"DAQmx_AI_Thrmstr_B", "thrmstr_b"},
    {"DAQmx_AI_Thrmstr_C", "thrmstr_c"},
    {"DAQmx_AI_Thrmstr_R1", "thrmstr_r1"},
    {"DAQmx_AI_ForceReadFromChan", "force_read_from_chan"},
    {"DAQmx_AI_Current_Units", "current_units"},
    {"DAQmx_AI_Current_ACRMS_Units", "current_acrms_units"},
    {"DAQmx_AI_Strain_Units", "strain_units"},
    {"DAQmx_AI_StrainGage_ForceReadFromChan", "straingage_force_read_from_chan"},
    {"DAQmx_AI_StrainGage_GageFactor", "straingage_gage_factor"},
    {"DAQmx_AI_StrainGage_PoissonRatio", "straingage_poisson_ratio"},
    {"DAQmx_AI_StrainGage_Cfg", "straingage_cfg"},
    {"DAQmx_AI_RosetteStrainGage_RosetteType", "rosettestraingage_rosette_type"},
    {"DAQmx_AI_RosetteStrainGage_Orientation", "rosettestraingage_orientation"},
    {"DAQmx_AI_RosetteStrainGage_StrainChans", "rosettestraingage_strain_chans"},
    {
        "DAQmx_AI_RosetteStrainGage_RosetteMeasType",
        "rosettestraingage_rosette_meas_type"
    },
    {"DAQmx_AI_Resistance_Units", "resistance_units"},
    {"DAQmx_AI_Freq_Units", "freq_units"},
    {"DAQmx_AI_Freq_ThreshVoltage", "freq_thresh_voltage"},
    {"DAQmx_AI_Freq_Hyst", "freq_hyst"},
    {"DAQmx_AI_LVDT_Units", "lvdt_units"},
    {"DAQmx_AI_LVDT_Sensitivity", "lvdt_sensitivity"},
    {"DAQmx_AI_LVDT_SensitivityUnits", "lvdt_sensitivity_units"},
    {"DAQmx_AI_RVDT_Units", "rvdt_units"},
    {"DAQmx_AI_RVDT_Sensitivity", "rvdt_sensitivity"},
    {"DAQmx_AI_RVDT_SensitivityUnits", "rvdt_sensitivity_units"},
    {"DAQmx_AI_EddyCurrentProxProbe_Units", "eddy_current_prox_probe_units"},
    {
        "DAQmx_AI_EddyCurrentProxProbe_Sensitivity",
        "eddy_current_prox_probe_sensitivity"
    },
    {
        "DAQmx_AI_EddyCurrentProxProbe_SensitivityUnits",
        "eddy_current_prox_probe_sensitivity_units"
    },
    {
        "DAQmx_AI_SoundPressure_MaxSoundPressureLvl",
        "sound_pressure_max_sound_pressure_lvl"
    },
    {"DAQmx_AI_SoundPressure_Units", "sound_pressure_units"},
    {"DAQmx_AI_SoundPressure_dBRef", "sound_pressure_db_ref"},
    {"DAQmx_AI_Microphone_Sensitivity", "microphone_sensitivity"},
    {"DAQmx_AI_Accel_Units", "accel_units"},
    {"DAQmx_AI_Accel_dBRef", "accel_db_ref"},
    {"DAQmx_AI_Accel_4WireDCVoltage_Sensitivity", "accel_4wire_dc_voltage_sensitivity"},
    {
        "DAQmx_AI_Accel_4WireDCVoltage_SensitivityUnits",
        "accel_4wire_dc_voltage_sensitivity_units"
    },
    {"DAQmx_AI_Accel_Sensitivity", "accel_sensitivity"},
    {"DAQmx_AI_Accel_SensitivityUnits", "accel_sensitivity_units"},
    {"DAQmx_AI_Accel_Charge_Sensitivity", "accel_charge_sensitivity"},
    {"DAQmx_AI_Accel_Charge_SensitivityUnits", "accel_charge_sensitivity_units"},
    {"DAQmx_AI_Velocity_Units", "velocity_units"},
    {"DAQmx_AI_Velocity_IEPESensor_dBRef", "velocity_iepe_sensor_db_ref"},
    {"DAQmx_AI_Velocity_IEPESensor_Sensitivity", "velocity_iepe_sensor_sensitivity"},
    {
        "DAQmx_AI_Velocity_IEPESensor_SensitivityUnits",
        "velocity_iepe_sensor_sensitivity_units"
    },
    {"DAQmx_AI_Force_Units", "force_units"},
    {"DAQmx_AI_Force_IEPESensor_Sensitivity", "force_iepe_sensor_sensitivity"},
    {
        "DAQmx_AI_Force_IEPESensor_SensitivityUnits",
        "force_iepe_sensor_sensitivity_units"
    },
    {"DAQmx_AI_Pressure_Units", "pressure_units"},
    {"DAQmx_AI_Torque_Units", "torque_units"},
    {"DAQmx_AI_Bridge_Units", "bridge_units"},
    {"DAQmx_AI_Bridge_ElectricalUnits", "bridge_electrical_units"},
    {"DAQmx_AI_Bridge_PhysicalUnits", "bridge_physical_units"},
    {"DAQmx_AI_Bridge_ScaleType", "bridge_scale_type"},
    {
        "DAQmx_AI_Bridge_TwoPointLin_First_ElectricalVal",
        "bridge_two_point_lin_first_electrical_val"
    },
    {
        "DAQmx_AI_Bridge_TwoPointLin_First_PhysicalVal",
        "bridge_two_point_lin_first_physical_val"
    },
    {
        "DAQmx_AI_Bridge_TwoPointLin_Second_ElectricalVal",
        "bridge_two_point_lin_second_electrical_val"
    },
    {
        "DAQmx_AI_Bridge_TwoPointLin_Second_PhysicalVal",
        "bridge_two_point_lin_second_physical_val"
    },
    {"DAQmx_AI_Bridge_Table_ElectricalVals", "bridge_table_electrical_vals"},
    {"DAQmx_AI_Bridge_Table_PhysicalVals", "bridge_table_physical_vals"},
    {"DAQmx_AI_Bridge_Poly_ForwardCoeff", "bridge_poly_forward_coeff"},
    {"DAQmx_AI_Bridge_Poly_ReverseCoeff", "bridge_poly_reverse_coeff"},
    {"DAQmx_AI_Charge_Units", "charge_units"},
    {"DAQmx_AI_Is_TEDS", "is_teds"},
    {"DAQmx_AI_TEDS_Units", "teds_units"},
    {"DAQmx_AI_Coupling", "coupling"},
    {"DAQmx_AI_Impedance", "impedance"},
    {"DAQmx_AI_TermCfg", "term_cfg"},
    {"DAQmx_AI_InputSrc", "input_src"},
    {"DAQmx_AI_ResistanceCfg", "resistance_cfg"},
    {"DAQmx_AI_LeadWireResistance", "lead_wire_resistance"},
    {"DAQmx_AI_Bridge_Cfg", "bridge_cfg"},
    {"DAQmx_AI_Bridge_NomResistance", "bridge_nom_resistance"},
    {"DAQmx_AI_Bridge_InitialVoltage", "bridge_initial_voltage"},
    {"DAQmx_AI_Bridge_InitialRatio", "bridge_initial_ratio"},
    {"DAQmx_AI_Bridge_ShuntCal_Enable", "bridge_shunt_cal_enable"},
    {"DAQmx_AI_Bridge_ShuntCal_Select", "bridge_shunt_cal_select"},
    {"DAQmx_AI_Bridge_ShuntCal_ShuntCalASrc", "bridge_shunt_cal_shunt_cal_a_src"},
    {"DAQmx_AI_Bridge_ShuntCal_GainAdjust", "bridge_shunt_cal_gain_adjust"},
    {
        "DAQmx_AI_Bridge_ShuntCal_ShuntCalAResistance",
        "bridge_shunt_cal_shunt_cal_a_resistance"
    },
    {
        "DAQmx_AI_Bridge_ShuntCal_ShuntCalAActualResistance",
        "bridge_shunt_cal_shunt_cal_a_actual_resistance"
    },
    {
        "DAQmx_AI_Bridge_ShuntCal_ShuntCalBResistance",
        "bridge_shunt_cal_shunt_cal_b_resistance"
    },
    {
        "DAQmx_AI_Bridge_ShuntCal_ShuntCalBActualResistance",
        "bridge_shunt_cal_shunt_cal_b_actual_resistance"
    },
    {"DAQmx_AI_Bridge_Balance_CoarsePot", "bridge_balance_coarse_pot"},
    {"DAQmx_AI_Bridge_Balance_FinePot", "bridge_balance_fine_pot"},
    {"DAQmx_AI_CurrentShunt_Loc", "current_shunt_loc"},
    {"DAQmx_AI_CurrentShunt_Resistance", "current_shunt_resistance"},
    {"DAQmx_AI_Excit_Sense", "excit_sense"},
    {"DAQmx_AI_Excit_Sense", "excit_sense"},
    {"DAQmx_AI_Excit_Src", "excit_src"},
    {"DAQmx_AI_Excit_Val", "excit_val"},
    {"DAQmx_AI_Excit_UseForScaling", "excit_use_for_scaling"},
    {"DAQmx_AI_Excit_UseMultiplexed", "excit_use_multiplexed"},
    {"DAQmx_AI_Excit_ActualVal", "excit_actual_val"},
    {"DAQmx_AI_Excit_DCorAC", "excit_dcorac"},
    {"DAQmx_AI_Excit_VoltageOrCurrent", "excit_voltage_or_current"},
    {"DAQmx_AI_Excit_IdleOutputBehavior", "excit_idle_output_behavior"},
    {"DAQmx_AI_ACExcit_Freq", "ac_excit_freq"},
    {"DAQmx_AI_ACExcit_SyncEnable", "ac_excit_sync_enable"},
    {"DAQmx_AI_ACExcit_WireMode", "ac_excit_wire_mode"},
    {"DAQmx_AI_SensorPower_Voltage", "sensor_power_voltage"},
    {"DAQmx_AI_SensorPower_Cfg", "sensor_power_cfg"},
    {"DAQmx_AI_SensorPower_Type", "sensor_power_type"},
    {"DAQmx_AI_OpenThrmcplDetectEnable", "open_thrmcpl_detect_enable"},
    {"DAQmx_AI_Thrmcpl_LeadOffsetVoltage", "thrmcpl_lead_offset_voltage"},
    {"DAQmx_AI_Atten", "atten"},
    {"DAQmx_AI_ProbeAtten", "probe_atten"},
    {"DAQmx_AI_Lowpass_Enable", "lowpass_enable"},
    {"DAQmx_AI_Lowpass_CutoffFreq", "lowpass_cutoff_freq"},
    {"DAQmx_AI_Lowpass_SwitchCap_ClkSrc", "lowpass_switch_cap_clk_src"},
    {"DAQmx_AI_Lowpass_SwitchCap_ExtClkFreq", "lowpass_switch_cap_ext_clk_freq"},
    {"DAQmx_AI_Lowpass_SwitchCap_ExtClkDiv", "lowpass_switch_cap_ext_clk_div"},
    {"DAQmx_AI_Lowpass_SwitchCap_OutClkDiv", "lowpass_switch_cap_out_clk_div"},
    {"DAQmx_AI_DigFltr_Enable", "dig_fltr_enable"},
    {"DAQmx_AI_DigFltr_Type", "dig_fltr_type"},
    {"DAQmx_AI_DigFltr_Response", "dig_fltr_response"},
    {"DAQmx_AI_DigFltr_Order", "dig_fltr_order"},
    {"DAQmx_AI_DigFltr_Lowpass_CutoffFreq", "dig_fltr_lowpass_cutoff_freq"},
    {"DAQmx_AI_DigFltr_Highpass_CutoffFreq", "dig_fltr_highpass_cutoff_freq"},
    {"DAQmx_AI_DigFltr_Bandpass_CenterFreq", "dig_fltr_bandpass_center_freq"},
    {"DAQmx_AI_DigFltr_Bandpass_Width", "dig_fltr_bandpass_width"},
    {"DAQmx_AI_DigFltr_Notch_CenterFreq", "dig_fltr_notch_center_freq"},
    {"DAQmx_AI_DigFltr_Notch_Width", "dig_fltr_notch_width"},
    {"DAQmx_AI_DigFltr_Coeff", "dig_fltr_coeff"},
    {"DAQmx_AI_Filter_Enable", "filter_enable"},
    {"DAQmx_AI_Filter_Freq", "filter_freq"},
    {"DAQmx_AI_Filter_Response", "filter_response"},
    {"DAQmx_AI_Filter_Order", "filter_order"},
    {"DAQmx_AI_FilterDelay", "filter_delay"},
    {"DAQmx_AI_FilterDelayUnits", "filter_delay_units"},
    {"DAQmx_AI_RemoveFilterDelay", "remove_filter_delay"},
    {"DAQmx_AI_FilterDelayAdjustment", "filter_delay_adjustment"},
    {"DAQmx_AI_AveragingWinSize", "averaging_win_size"},
    {"DAQmx_AI_ResolutionUnits", "resolution_units"},
    {"DAQmx_AI_Resolution", "resolution"},
    {"DAQmx_AI_RawSampSize", "raw_samp_size"},
    {"DAQmx_AI_RawSampJustification", "raw_samp_justification"},
    {"DAQmx_AI_ADCTimingMode", "adc_timing_mode"},
    {"DAQmx_AI_ADCCustomTimingMode", "adc_custom_timing_mode"},
    {"DAQmx_AI_Dither_Enable", "dither_enable"},
    {"DAQmx_AI_ChanCal_HasValidCalInfo", "chan_cal_has_valid_cal_info"},
    {"DAQmx_AI_ChanCal_EnableCal", "chan_cal_enable_cal"},
    {"DAQmx_AI_ChanCal_ApplyCalIfExp", "chan_cal_apply_cal_if_exp"},
    {"DAQmx_AI_ChanCal_ScaleType", "chan_cal_scale_type"},
    {"DAQmx_AI_ChanCal_Table_PreScaledVals", "chan_cal_table_pre_scaled_vals"},
    {"DAQmx_AI_ChanCal_Table_ScaledVals", "chan_cal_table_scaled_vals"},
    {"DAQmx_AI_ChanCal_Poly_ForwardCoeff", "chan_cal_poly_forward_coeff"},
    {"DAQmx_AI_ChanCal_Poly_ReverseCoeff", "chan_cal_poly_reverse_coeff"},
    {"DAQmx_AI_ChanCal_OperatorName", "chan_cal_operator_name"},
    {"DAQmx_AI_ChanCal_Desc", "chan_cal_desc"},
    {"DAQmx_AI_ChanCal_Verif_RefVals", "chan_cal_verif_ref_vals"},
    {"DAQmx_AI_ChanCal_Verif_AcqVals", "chan_cal_verif_acq_vals"},
    {"DAQmx_AI_Rng_High", "rng_high"},
    {"DAQmx_AI_Rng_Low", "rng_low"},
    {"DAQmx_AI_DCOffset", "dc_offset"},
    {"DAQmx_AI_Gain", "gain"},
    {"DAQmx_AI_SampAndHold_Enable", "samp_and_hold_enable"},
    {"DAQmx_AI_AutoZeroMode", "auto_zero_mode"},
    {"DAQmx_AI_ChopEnable", "chop_enable"},
    {"DAQmx_AI_DataXferMaxRate", "data_xfer_max_rate"},
    {"DAQmx_AI_DataXferMech", "data_xfer_mech"},
    {"DAQmx_AI_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_AI_DataXferCustomThreshold", "data_xfer_custom_threshold"},
    {"DAQmx_AI_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_AI_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_AI_MemMapEnable", "mem_map_enable"},
    {"DAQmx_AI_RawDataCompressionType", "raw_data_compression_type"},
    {
        "DAQmx_AI_LossyLSBRemoval_CompressedSampSize",
        "lossy_lsb_removal_compressed_samp_size"
    },
    {"DAQmx_AI_DevScalingCoeff", "dev_scaling_coeff"},
    {"DAQmx_AI_EnhancedAliasRejectionEnable", "enhanced_alias_rejection_enable"},
    {"DAQmx_AI_OpenChanDetectEnable", "open_chan_detect_enable"},
    {
        "DAQmx_AI_InputLimitsFaultDetect_UpperLimit",
        "input_limits_fault_detect_upper_limit"
    },
    {
        "DAQmx_AI_InputLimitsFaultDetect_LowerLimit",
        "input_limits_fault_detect_lower_limit"
    },
    {"DAQmx_AI_InputLimitsFaultDetectEnable", "input_limits_fault_detect_enable"},
    {"DAQmx_AI_PowerSupplyFaultDetectEnable", "power_supply_fault_detect_enable"},
    {"DAQmx_AI_OvercurrentDetectEnable", "overcurrent_detect_enable"},
    {"DAQmx_AO_Max", "max"},
    {"DAQmx_AO_Min", "min"},
    {"DAQmx_AO_CustomScaleName", "custom_scale_name"},
    {"DAQmx_AO_OutputType", "output_type"},
    {"DAQmx_AO_Voltage_Units", "voltage_units"},
    {"DAQmx_AO_Voltage_CurrentLimit", "voltage_current_limit"},
    {"DAQmx_AO_Current_Units", "current_units"},
    {"DAQmx_AO_FuncGen_Type", "func_gen_type"},
    {"DAQmx_AO_FuncGen_Freq", "func_gen_freq"},
    {"DAQmx_AO_FuncGen_Amplitude", "func_gen_amplitude"},
    {"DAQmx_AO_FuncGen_Offset", "func_gen_offset"},
    {"DAQmx_AO_FuncGen_Square_DutyCycle", "func_gen_square_duty_cycle"},
    {"DAQmx_AO_FuncGen_ModulationType", "func_gen_modulation_type"},
    {"DAQmx_AO_FuncGen_FMDeviation", "func_gen_fm_deviation"},
    {"DAQmx_AO_OutputImpedance", "output_impedance"},
    {"DAQmx_AO_LoadImpedance", "load_impedance"},
    {"DAQmx_AO_IdleOutputBehavior", "idle_output_behavior"},
    {"DAQmx_AO_TermCfg", "term_cfg"},
    {"DAQmx_AO_ResolutionUnits", "resolution_units"},
    {"DAQmx_AO_Resolution", "resolution"},
    {"DAQmx_AO_DAC_Rng_High", "dac_rng_high"},
    {"DAQmx_AO_DAC_Rng_Low", "dac_rng_low"},
    {"DAQmx_AO_DAC_Ref_ConnToGnd", "dac_ref_conn_to_gnd"},
    {"DAQmx_AO_DAC_Ref_AllowConnToGnd", "dac_ref_allow_conn_to_gnd"},
    {"DAQmx_AO_DAC_Ref_Src", "dac_ref_src"},
    {"DAQmx_AO_DAC_Ref_Src", "dac_ref_src"},
    {"DAQmx_AO_DAC_Ref_ExtSrc", "dac_ref_ext_src"},
    {"DAQmx_AO_DAC_Ref_Val", "dac_ref_val"},
    {"DAQmx_AO_DAC_Offset_Src", "dac_offset_src"},
    {"DAQmx_AO_DAC_Offset_ExtSrc", "dac_offset_ext_src"},
    {"DAQmx_AO_DAC_Offset_Val", "dac_offset_val"},
    {"DAQmx_AO_ReglitchEnable", "reglitch_enable"},
    {"DAQmx_AO_FilterDelay", "filter_delay"},
    {"DAQmx_AO_FilterDelayUnits", "filter_delay_units"},
    {"DAQmx_AO_FilterDelayAdjustment", "filter_delay_adjustment"},
    {"DAQmx_AO_Gain", "gain"},
    {"DAQmx_AO_UseOnlyOnBrdMem", "use_only_on_brd_mem"},
    {"DAQmx_AO_DataXferMech", "data_xfer_mech"},
    {"DAQmx_AO_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_AO_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_AO_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_AO_MemMapEnable", "mem_map_enable"},
    {"DAQmx_AO_DevScalingCoeff", "dev_scaling_coeff"},
    {"DAQmx_AO_EnhancedImageRejectionEnable", "enhanced_image_rejection_enable"},
    {"DAQmx_DI_InvertLines", "invert_lines"},
    {"DAQmx_DI_NumLines", "num_lines"},
    {"DAQmx_DI_DigFltr_Enable", "dig_fltr_enable"},
    {"DAQmx_DI_DigFltr_MinPulseWidth", "dig_fltr_min_pulse_width"},
    {"DAQmx_DI_DigFltr_EnableBusMode", "dig_fltr_enable_bus_mode"},
    {"DAQmx_DI_DigFltr_TimebaseSrc", "dig_fltr_timebase_src"},
    {"DAQmx_DI_DigFltr_TimebaseRate", "dig_fltr_timebase_rate"},
    {"DAQmx_DI_DigSync_Enable", "dig_sync_enable"},
    {"DAQmx_DI_Tristate", "tristate"},
    {"DAQmx_DI_LogicFamily", "logic_family"},
    {"DAQmx_DI_DataXferMech", "data_xfer_mech"},
    {"DAQmx_DI_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_DI_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_DI_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_DI_MemMapEnable", "mem_map_enable"},
    {"DAQmx_DI_AcquireOn", "acquire_on"},
    {"DAQmx_DO_OutputDriveType", "output_drive_type"},
    {"DAQmx_DO_InvertLines", "invert_lines"},
    {"DAQmx_DO_NumLines", "num_lines"},
    {"DAQmx_DO_Tristate", "tristate"},
    {"DAQmx_DO_LineStates_StartState", "line_states_start_state"},
    {"DAQmx_DO_LineStates_PausedState", "line_states_paused_state"},
    {"DAQmx_DO_LineStates_DoneState", "line_states_done_state"},
    {"DAQmx_DO_LogicFamily", "logic_family"},
    {"DAQmx_DO_Overcurrent_Limit", "overcurrent_limit"},
    {"DAQmx_DO_Overcurrent_AutoReenable", "overcurrent_auto_reenable"},
    {"DAQmx_DO_Overcurrent_ReenablePeriod", "overcurrent_reenable_period"},
    {"DAQmx_DO_UseOnlyOnBrdMem", "use_only_on_brd_mem"},
    {"DAQmx_DO_DataXferMech", "data_xfer_mech"},
    {"DAQmx_DO_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_DO_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_DO_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_DO_MemMapEnable", "mem_map_enable"},
    {"DAQmx_DO_GenerateOn", "generate_on"},
    {"DAQmx_CI_Max", "max"},
    {"DAQmx_CI_Min", "min"},
    {"DAQmx_CI_CustomScaleName", "custom_scale_name"},
    {"DAQmx_CI_MeasType", "meas_type"},
    {"DAQmx_CI_Freq_Units", "freq_units"},
    {"DAQmx_CI_Freq_Term", "freq_term"},
    {"DAQmx_CI_Freq_TermCfg", "freq_term_cfg"},
    {"DAQmx_CI_Freq_LogicLvlBehavior", "freq_logic_lvl_behavior"},
    {"DAQmx_CI_Freq_DigFltr_Enable", "freq_dig_fltr_enable"},
    {"DAQmx_CI_Freq_DigFltr_MinPulseWidth", "freq_dig_fltr_min_pulse_width"},
    {"DAQmx_CI_Freq_DigFltr_TimebaseSrc", "freq_dig_fltr_timebase_src"},
    {"DAQmx_CI_Freq_DigFltr_TimebaseRate", "freq_dig_fltr_timebase_rate"},
    {"DAQmx_CI_Freq_DigSync_Enable", "freq_dig_sync_enable"},
    {"DAQmx_CI_Freq_StartingEdge", "freq_starting_edge"},
    {"DAQmx_CI_Freq_MeasMeth", "freq_meas_meth"},
    {"DAQmx_CI_Freq_EnableAveraging", "freq_enable_averaging"},
    {"DAQmx_CI_Freq_MeasTime", "freq_meas_time"},
    {"DAQmx_CI_Freq_Div", "freq_div"},
    {"DAQmx_CI_Period_Units", "period_units"},
    {"DAQmx_CI_Period_Term", "period_term"},
    {"DAQmx_CI_Period_TermCfg", "period_term_cfg"},
    {"DAQmx_CI_Period_LogicLvlBehavior", "period_logic_lvl_behavior"},
    {"DAQmx_CI_Period_DigFltr_Enable", "period_dig_fltr_enable"},
    {"DAQmx_CI_Period_DigFltr_MinPulseWidth", "period_dig_fltr_min_pulse_width"},
    {"DAQmx_CI_Period_DigFltr_TimebaseSrc", "period_dig_fltr_timebase_src"},
    {"DAQmx_CI_Period_DigFltr_TimebaseRate", "period_dig_fltr_timebase_rate"},
    {"DAQmx_CI_Period_DigSync_Enable", "period_dig_sync_enable"},
    {"DAQmx_CI_Period_StartingEdge", "period_starting_edge"},
    {"DAQmx_CI_Period_MeasMeth", "period_meas_meth"},
    {"DAQmx_CI_Period_EnableAveraging", "period_enable_averaging"},
    {"DAQmx_CI_Period_MeasTime", "period_meas_time"},
    {"DAQmx_CI_Period_Div", "period_div"},
    {"DAQmx_CI_CountEdges_Term", "count_edges_term"},
    {"DAQmx_CI_CountEdges_TermCfg", "count_edges_term_cfg"},
    {"DAQmx_CI_CountEdges_LogicLvlBehavior", "count_edges_logic_lvl_behavior"},
    {"DAQmx_CI_CountEdges_DigFltr_Enable", "count_edges_dig_fltr_enable"},
    {
        "DAQmx_CI_CountEdges_DigFltr_MinPulseWidth",
        "count_edges_dig_fltr_min_pulse_width"
    },
    {"DAQmx_CI_CountEdges_DigFltr_TimebaseSrc", "count_edges_dig_fltr_timebase_src"},
    {"DAQmx_CI_CountEdges_DigFltr_TimebaseRate", "count_edges_dig_fltr_timebase_rate"},
    {"DAQmx_CI_CountEdges_DigSync_Enable", "count_edges_dig_sync_enable"},
    {"DAQmx_CI_CountEdges_Dir", "count_edges_dir"},
    {"DAQmx_CI_CountEdges_DirTerm", "count_edges_dir_term"},
    {"DAQmx_CI_CountEdges_CountDir_TermCfg", "count_edges_count_dir_term_cfg"},
    {
        "DAQmx_CI_CountEdges_CountDir_LogicLvlBehavior",
        "count_edges_count_dir_logic_lvl_behavior"
    },
    {
        "DAQmx_CI_CountEdges_CountDir_DigFltr_Enable",
        "count_edges_count_dir_dig_fltr_enable"
    },
    {
        "DAQmx_CI_CountEdges_CountDir_DigFltr_MinPulseWidth",
        "count_edges_count_dir_dig_fltr_min_pulse_width"
    },
    {
        "DAQmx_CI_CountEdges_CountDir_DigFltr_TimebaseSrc",
        "count_edges_count_dir_dig_fltr_timebase_src"
    },
    {
        "DAQmx_CI_CountEdges_CountDir_DigFltr_TimebaseRate",
        "count_edges_count_dir_dig_fltr_timebase_rate"
    },
    {
        "DAQmx_CI_CountEdges_CountDir_DigSync_Enable",
        "count_edges_count_dir_dig_sync_enable"
    },
    {"DAQmx_CI_CountEdges_InitialCnt", "count_edges_initial_cnt"},
    {"DAQmx_CI_CountEdges_ActiveEdge", "count_edges_active_edge"},
    {"DAQmx_CI_CountEdges_CountReset_Enable", "count_edges_count_reset_enable"},
    {
        "DAQmx_CI_CountEdges_CountReset_ResetCount",
        "count_edges_count_reset_reset_count"
    },
    {"DAQmx_CI_CountEdges_CountReset_Term", "count_edges_count_reset_term"},
    {"DAQmx_CI_SampClkOverrunSentinelVal", "samp_clk_overrun_sentinel_val"},
    {"DAQmx_CI_DataXferMech", "data_xfer_mech"},
    {"DAQmx_CI_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_CI_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_CI_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_CI_MemMapEnable", "mem_map_enable"},
    {"DAQmx_CI_NumPossiblyInvalidSamps", "num_possibly_invalid_samps"},
    {"DAQmx_CI_DupCountPrevent", "dup_count_prevent"},
    {"DAQmx_CI_Prescaler", "prescaler"},
    {"DAQmx_CI_MaxMeasPeriod", "max_meas_period"},
    {"DAQmx_CO_OutputType", "output_type"},
    {"DAQmx_CO_Pulse_IdleState", "pulse_idle_state"},
    {"DAQmx_CO_Pulse_Term", "pulse_term"},
    {"DAQmx_CO_Pulse_Time_Units", "pulse_time_units"},
    {"DAQmx_CO_Pulse_HighTime", "pulse_high_time"},
    {"DAQmx_CO_Pulse_LowTime", "pulse_low_time"},
    {"DAQmx_CO_Pulse_Time_InitialDelay", "pulse_time_initial_delay"},
    {"DAQmx_CO_Pulse_DutyCyc", "pulse_duty_cyc"},
    {"DAQmx_CO_Pulse_Freq_Units", "pulse_freq_units"},
    {"DAQmx_CO_Pulse_Freq", "pulse_freq"},
    {"DAQmx_CO_Pulse_Freq_InitialDelay", "pulse_freq_initial_delay"},
    {"DAQmx_CO_Pulse_HighTicks", "pulse_high_ticks"},
    {"DAQmx_CO_Pulse_LowTicks", "pulse_low_ticks"},
    {"DAQmx_CO_Pulse_Ticks_InitialDelay", "pulse_ticks_initial_delay"},
    {"DAQmx_CO_CtrTimebaseSrc", "ctr_timebase_src"},
    {"DAQmx_CO_CtrTimebaseRate", "ctr_timebase_rate"},
    {"DAQmx_CO_CtrTimebaseActiveEdge", "ctr_timebase_active_edge"},
    {"DAQmx_CO_CtrTimebase_DigFltr_Enable", "ctr_timebase_dig_fltr_enable"},
    {
        "DAQmx_CO_CtrTimebase_DigFltr_MinPulseWidth",
        "ctr_timebase_dig_fltr_min_pulse_width"
    },
    {"DAQmx_CO_CtrTimebase_DigFltr_TimebaseSrc", "ctr_timebase_dig_fltr_timebase_src"},
    {
        "DAQmx_CO_CtrTimebase_DigFltr_TimebaseRate",
        "ctr_timebase_dig_fltr_timebase_rate"
    },
    {"DAQmx_CO_CtrTimebase_DigSync_Enable", "ctr_timebase_dig_sync_enable"},
    {"DAQmx_CO_Count", "count"},
    {"DAQmx_CO_OutputState", "output_state"},
    {"DAQmx_CO_AutoIncrCnt", "auto_incr_cnt"},
    {"DAQmx_CO_CtrTimebaseMasterTimebaseDiv", "ctr_timebase_master_timebase_div"},
    {"DAQmx_CO_PulseDone", "pulse_done"},
    {"DAQmx_CO_EnableInitialDelayOnRetrigger", "enable_initial_delay_on_retrigger"},
    {"DAQmx_CO_ConstrainedGenMode", "constrained_gen_mode"},
    {"DAQmx_CO_UseOnlyOnBrdMem", "use_only_on_brd_mem"},
    {"DAQmx_CO_DataXferMech", "data_xfer_mech"},
    {"DAQmx_CO_DataXferReqCond", "data_xfer_req_cond"},
    {"DAQmx_CO_UsbXferReqSize", "usb_xfer_req_size"},
    {"DAQmx_CO_UsbXferReqCount", "usb_xfer_req_count"},
    {"DAQmx_CO_MemMapEnable", "mem_map_enable"},
    {"DAQmx_CO_Prescaler", "prescaler"},
    {"DAQmx_CO_RdyForNewVal", "rdy_for_new_val"},
    {"DAQmx_ChanType", "chan_type"},
    {"DAQmx_PhysicalChanName", "physical_chan_name"},
    {"DAQmx_ChanDescr", "chan_descr"},
    {"DAQmx_ChanIsGlobal", "chan_is_global"},
    {"DAQmx_Chan_SyncUnlockBehavior", "chan_sync_unlock_behavior"},
    {"DAQmx_SampClk_Rate", "sample_rate"}
};


///////////////////////////////////////////////////////////////////////////////////
//                                    NiSource                                   //
///////////////////////////////////////////////////////////////////////////////////
// Also verify data type of channel
void ni::Source::get_index_keys() {
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel: this->reader_config.channels) {
        auto [channel_info, err] = this->ctx->client->channels.retrieve(
            channel.channel_key);
        if (err) {
            this->log_error(
                "failed to retrieve channel " + std::to_string(channel.channel_key));
            return;
        } else {
            index_keys.insert(channel_info.index);
        }
    }
    // now iterate through the set and add all the index channels as configs
    for (auto it = index_keys.begin(); it != index_keys.end(); ++it) {
        auto index_key = *it;
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err != freighter::NIL) {
            this->log_error("failed to retrieve channel " + std::to_string(index_key));
            return;
        } else {
            ni::ChannelConfig index_channel;
            index_channel.channel_key = channel_info.key;
            index_channel.channel_type = "index";
            index_channel.name = channel_info.name;
            this->reader_config.channels.push_back(index_channel);
            // LOG(INFO) << "[NI Reader] index channel " << index_channel.channel_key << " and name: " << index_channel.name <<" added to task " << this->reader_config.task_name;
        }
    }
}


ni::Source::Source(
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task) : task_handle(task_handle), ctx(ctx), task(task) {
}

void ni::Source::parse_config(config::Parser &parser) {
    // Get Acquisition Rate and Stream Rates
    this->reader_config.sample_rate.value = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate.value = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.required<std::string>("device");
    this->reader_config.timing_source = "none";
    // parser.required<std::string>("timing_source"); TODO: uncomment this when ui provides timing source
    if (parser.optional<bool>("test", false))
        this->reader_config.device_name = parser.required<std::string>(
            "device_location");
    else {
        auto [dev, err] = this->ctx->client->hardware.retrieveDevice(
            this->reader_config.device_key);
        if (err != freighter::NIL) {
            this->log_error(
                "failed to retrieve device " + this->reader_config.device_name);
            return;
        }
        this->reader_config.device_name = dev.location;
    }
    this->parse_channels(parser);
}

int ni::Source::init() {
    // Create parser
    auto config_parser = config::Parser(this->task.config);
    this->reader_config.task_name = this->task.name;
    this->reader_config.task_key = this->task.key;
    // Parse configuration and make sure it is valid
    this->parse_config(config_parser);
    if (!config_parser.ok()) {
        json error_json = config_parser.error_json();
        error_json["running"] = false;
        // Log error
        this->log_error(
            "failed to parse configuration for " + this->reader_config.task_name +
            " Parser Error: " +
            config_parser.error_json().dump());
        this->ctx->setState({
            .task = task.key,
            .variant = "error",
            .details = config_parser.error_json()
        });
        return -1;
    }
    this->get_index_keys();
    this->validate_channels();
    // Create breaker
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    this->breaker = breaker::Breaker(breaker_config);
    int err = this->create_channels();
    if (err) {
        this->log_error(
            "failed to create channels for " + this->reader_config.task_name);
        return -1;
    }
    // Configure buffer size and read resources
    if (this->reader_config.sample_rate < this->reader_config.stream_rate) {
        this->log_error(
            "Failed while configuring timing for NI hardware for task " + this->
            reader_config.task_name);
        this->err_info["error type"] = "Configuration Error";
        this->err_info["error details"] = "Stream rate is greater than sample rate";
        this->err_info["running"] = false;

        this->ctx->setState({
            .task = this->task.key,
            .variant = "error",
            .details = err_info
        });
        return -1;
    }
    if (this->configure_timing())
        this->log_error(
            "[NI Reader] Failed while configuring timing for NI hardware for task " +
            this->reader_config.task_name);

    return 0;
}

freighter::Error  ni::Source::cycle(){
    if (this->breaker.running() || !this->ok()) return freighter::NIL;
    if (this->check_ni_error(ni::NiDAQmxInterface::StartTask(this->task_handle))) {
        this->log_error(
            "failed while starting reader for task " + this->reader_config.task_name +
            " requires reconfigure");
        // this->clear_task();
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    if (this->check_ni_error(ni::NiDAQmxInterface::StopTask(this->task_handle))) {
        this->log_error(
            "failed while stopping reader for task " + this->reader_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    return freighter::NIL;
}

freighter::Error ni::Source::start() {
    if (this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.start();
    if (this->check_ni_error(ni::NiDAQmxInterface::StartTask(this->task_handle))) {
        this->log_error(
            "failed while starting reader for task " + this->reader_config.task_name +
            " requires reconfigure");
        this->clear_task();
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    this->sample_thread = std::thread(&ni::Source::acquire_data, this);
    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", true},
            {"message", "Task started successfully"}
        }
    });
    return freighter::NIL;
}

freighter::Error ni::Source::stop() {
    if (!this->breaker.running() || !this->ok()) return freighter::NIL;
    this->breaker.stop();
    if (this->sample_thread.joinable()) this->sample_thread.join();
    if (this->check_ni_error(ni::NiDAQmxInterface::StopTask(this->task_handle))) {
        this->log_error(
            "failed while stopping reader for task " + this->reader_config.task_name);
        return freighter::Error(driver::CRITICAL_HARDWARE_ERROR);
    }
    data_queue.reset();
    LOG(INFO) << "[NI Reader] stopped reader for task " << this->reader_config.
            task_name;
    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Task stopped successfully"}
        }
    });
    return freighter::NIL;
}


void ni::Source::clear_task() {
    if (this->check_ni_error(ni::NiDAQmxInterface::ClearTask(this->task_handle))) {
        this->log_error(
            "failed while clearing reader for task " + this->reader_config.task_name);
    }
}


ni::Source::~Source() {
    this->clear_task();
    if(this->sample_thread.joinable()) this->sample_thread.join();
    LOG(INFO) << "[NI Reader] joined sample thread";
}

int ni::Source::check_ni_error(int32 error) {
    if (error < 0) {
        char errBuff[4096] = {'\0'};

        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 4096);

        std::string s(errBuff);
        jsonify_error(errBuff);

        this->ctx->setState({
            .task = this->task.key,
            .variant = "error",
            .details = err_info
        });

        LOG(ERROR) << "[NI Reader] Vendor error: " << s;
        this->ok_state = false;
        return -1;
    }
    return 0;
}

bool ni::Source::ok() {
    return this->ok_state;
}


std::vector<synnax::ChannelKey> ni::Source::getChannelKeys() {
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel: this->reader_config.channels) keys.push_back(
        channel.channel_key);
    return keys;
}

void ni::Source::log_error(std::string err_msg) {
    LOG(ERROR) << "[NI Reader] " << err_msg;
    this->ok_state = false;
    return;
}

void ni::Source::stoppedWithErr(const freighter::Error &err) {
    this->log_error("stopped with error: " + err.message());
    json j = json(err.message());
    this->ctx->setState({
        .task = this->reader_config.task_key,
        .variant = "error",
        .details = {
            {"running", false},
            {"message", j}
        }
    });
    this->stop();
    this->clear_task();
}

void ni::Source::jsonify_error(std::string s) {
    // TODO get rid of the fields outside of the errors array
    this->err_info["running"] = false;

    // Define regex patterns
    std::regex statusCodeRegex(R"(Status Code:\s*(-?\d+))");
    std::regex channelRegex(R"(Channel Name:\s*(\S+))");
    std::regex physicalChannelRegex(R"(Physical Channel Name:\s*(\S+))");
    std::regex deviceRegex(R"(Device:\s*(\S+))");
    std::regex possibleValuesRegex(R"(Possible Values:\s*([\w\s,.-]+))");
    std::regex maxValueRegex(R"(Maximum Value:\s*([\d.\s,eE-]+))");
    std::regex minValueRegex(R"(Minimum Value:\s*([\d.\s,eE-]+))");
    std::regex propertyRegex(R"(Property:\s*(\S+))");

    // Extract the entire message
    std::string message = s; // Start with the entire string

    // Define a vector of field names to look for
    std::vector<std::string> fields = {
        "Property:", "Status Code:", "Possible Values:", "Maximum Value:",
        "Minimum Value:", "Channel Name:", "Physical Channel Name:",
        "Device:", "Task Name:"
    };

    // Find the position of the first occurrence of any field
    size_t firstFieldPos = std::string::npos;
    for (const auto& field : fields) {
        size_t pos = s.find("\n" + field);
        if (pos != std::string::npos && (firstFieldPos == std::string::npos || pos < firstFieldPos)) {
            firstFieldPos = pos;
        }
    }

    // If we found a field, extract the message up to that point
    if (firstFieldPos != std::string::npos) {
        message = s.substr(0, firstFieldPos);
    }

    // Trim trailing whitespace and newlines
    message = std::regex_replace(message, std::regex("\\s+$"), "");

    // Extract status code
    std::string sc = "";
    std::smatch statusCodeMatch;
    if (std::regex_search(s, statusCodeMatch, statusCodeRegex)) {
        sc = statusCodeMatch[1].str();
    }

    // Extract device name
    std::string device = "";
    std::smatch deviceMatch;
    if (std::regex_search(s, deviceMatch, deviceRegex)) {
        device = deviceMatch[1].str();
    }

    // Extract physical channel name or channel name
    std::string cn = "";
    std::smatch physicalChannelMatch;
    if (std::regex_search(s, physicalChannelMatch, physicalChannelRegex)) {
        cn = physicalChannelMatch[1].str();
        if (!device.empty()) {
            cn = device + "/" + cn; // Combine device and physical channel name
        }
    } else {
        std::smatch channelMatch;
        if (std::regex_search(s, channelMatch, channelRegex)) {
            cn = channelMatch[1].str();
        }
    }

    // Extract the first property
    std::string p = "";
    std::smatch propertyMatch;
    if (std::regex_search(s, propertyMatch, propertyRegex)) {
        p = propertyMatch[1].str();
    }
    if(sc == "-200170"){
        p = "port";
    }

    // Extract possible values
    std::string possibleValues = "";
    std::smatch possibleValuesMatch;
    if (std::regex_search(s, possibleValuesMatch, possibleValuesRegex)) {
        possibleValues = possibleValuesMatch[1].str();

        // Remove "Channel Name" from possible values if it exists
        size_t pos = possibleValues.find("Channel Name");
        if (pos != std::string::npos) {
            possibleValues.erase(pos, std::string("Channel Name").length());
        }
    }

    // Extract maximum value
    std::string maxValue = "";
    std::smatch maxValueMatch;
    if (std::regex_search(s, maxValueMatch, maxValueRegex)) {
        maxValue = maxValueMatch[1].str();
    }

    // Extract minimum value
    std::string minValue = "";
    std::smatch minValueMatch;
    if (std::regex_search(s, minValueMatch, minValueRegex)) {
        minValue = minValueMatch[1].str();
    }

    // Check if the channel name is in the channel map
    if (channel_map.count(cn) != 0) {
        this->err_info["path"] = channel_map[cn] + ".";
    } else if (!cn.empty()) {
        this->err_info["path"] = cn + ".";
    } else {
        this->err_info["path"] = "";
    }

    // Check if the property is in the field map
    if (FIELD_MAP.count(p) == 0) {
        this->err_info["path"] = this->err_info["path"].get<std::string>() + p;
        this->err_info["message"] = "NI Error " + sc + ": " + message + " Path: " + this->err_info["path"].get<std::string>() + " Channel: " + cn;
        return;
    }

    this->err_info["type"] = "field error";
    this->err_info["path"] = this->err_info["path"].get<std::string>() + FIELD_MAP.at(p);

    // Update the message with possible values, max value, and min value if they exist
    std::string errorMessage = "NI Error " + sc + ": " + message + " Path: " + this->err_info["path"].get<std::string>();
    if (!possibleValues.empty()) {
        errorMessage += " Possible Values: " + possibleValues;
    }
    if (!maxValue.empty()) {
        errorMessage += " Maximum Value: " + maxValue;
    }
    if (!minValue.empty()) {
        errorMessage += " Minimum Value: " + minValue;
    }
    this->err_info["message"] = errorMessage;

    json j = json::array();
    j.push_back(this->err_info);
    this->err_info["errors"] = j;

    LOG(INFO) << this->err_info.dump(4);
}


