// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as v0 from "@/hardware/ni/task/types/v0";
import * as v1 from "@/hardware/ni/task/types/v1";

export const PREFIX = v0.PREFIX;

export type AccelSensitivityUnits = v0.AccelSensitivityUnits;
export type ForceUnits = v0.ForceUnits;
export type ElectricalUnits = v0.ElectricalUnits;
export type ShuntResistorLoc = v0.ShuntResistorLoc;
export type PressureUnits = v0.PressureUnits;
export type TemperatureUnits = v0.TemperatureUnits;
export type TorqueUnits = v0.TorqueUnits;
export type VelocityUnits = v0.VelocityUnits;
export type VelocitySensitivityUnits = v0.VelocitySensitivityUnits;
export type Units = v0.Units;

export type Scale = v0.Scale;
export type ScaleType = v0.ScaleType;
export const ZERO_SCALES = v0.ZERO_SCALES;
export const SCALE_SCHEMAS = v0.SCALE_SCHEMAS;

export const WAVE_TYPES = v0.WAVE_TYPES;
export type WaveType = v0.WaveType;

export type AIChannel = v1.AIChannel;
export type AIChannelType = v0.AIChannelType;
export const AI_CHANNEL_SCHEMAS = v1.AI_CHANNEL_SCHEMAS;
export const ZERO_AI_CHANNELS = v1.ZERO_AI_CHANNELS;
export const ZERO_AI_CHANNEL = v1.ZERO_AI_CHANNEL;
export const AI_CHANNEL_TYPE_NAMES = v0.AI_CHANNEL_TYPE_NAMES;
export const AI_CHANNEL_TYPE_ICONS = v0.AI_CHANNEL_TYPE_ICONS;

export type CIChannel = v0.CIChannel;
export type CIChannelType = v0.CIChannelType;
export type CIFreqUnits = v0.CIFreqUnits;
export type CIPeriodUnits = v0.CIPeriodUnits;
export type CIPulseWidthUnits = v0.CIPulseWidthUnits;
export type CISemiPeriodUnits = v0.CISemiPeriodUnits;
export type CITwoEdgeSepUnits = v0.CITwoEdgeSepUnits;
export type CILinearVelocityUnits = v0.CILinearVelocityUnits;
export type CIAngularVelocityUnits = v0.CIAngularVelocityUnits;
export type CILinearPositionUnits = v0.CILinearPositionUnits;
export type CIAngularPositionUnits = v0.CIAngularPositionUnits;
export type CIZIndexPhase = v0.CIZIndexPhase;
export type CIEdge = v0.CIEdge;
export type CIMeasMethod = v0.CIMeasMethod;
export type CICountDirection = v0.CICountDirection;
export type CIDecodingType = v0.CIDecodingType;
export const CI_CHANNEL_SCHEMAS = v0.CI_CHANNEL_SCHEMAS;
export const ZERO_CI_CHANNELS = v0.ZERO_CI_CHANNELS;
export const ZERO_CI_CHANNEL = v0.ZERO_CI_CHANNEL;
export const CI_CHANNEL_TYPE_NAMES = v0.CI_CHANNEL_TYPE_NAMES;
export const CI_CHANNEL_TYPE_ICONS = v0.CI_CHANNEL_TYPE_ICONS;

export const AO_CURRENT_CHAN_TYPE = v0.AO_CURRENT_CHAN_TYPE;
export const AO_FUNC_GEN_CHAN_TYPE = v0.AO_FUNC_GEN_CHAN_TYPE;
export const AO_VOLTAGE_CHAN_TYPE = v0.AO_VOLTAGE_CHAN_TYPE;
export type AOChannel = v0.AOChannel;
export type AOChannelType = v0.AOChannelType;
export const AO_CHANNEL_SCHEMAS = v0.AO_CHANNEL_SCHEMAS;
export const ZERO_AO_CHANNELS = v0.ZERO_AO_CHANNELS;
export const ZERO_AO_CHANNEL = v0.ZERO_AO_CHANNEL;
export const AO_CHANNEL_TYPES = v0.AO_CHANNEL_TYPES;
export const AO_CHANNEL_TYPE_NAMES = v0.AO_CHANNEL_TYPE_NAMES;
export const AO_CHANNEL_TYPE_ICONS = v0.AO_CHANNEL_TYPE_ICONS;

export type AnalogChannel = v1.AnalogChannel;

export interface DIChannel extends v0.DIChannel {}
export const ZERO_DI_CHANNEL = v0.ZERO_DI_CHANNEL;

export interface DOChannel extends v0.DOChannel {}
export const ZERO_DO_CHANNEL = v0.ZERO_DO_CHANNEL;

export type DigitalChannel = v0.DigitalChannel;

export type Channel = v1.Channel;

export const analogReadConfigZ = v1.analogReadConfigZ;
export const ANALOG_READ_TYPE = v0.ANALOG_READ_TYPE;
export const ZERO_ANALOG_READ_PAYLOAD = v1.ZERO_ANALOG_READ_PAYLOAD;
export const ANALOG_READ_SCHEMAS = v0.ANALOG_READ_SCHEMAS;
export type AnalogReadSchemas = v1.AnalogReadSchemas;

export const analogWriteConfigZ = v0.analogWriteConfigZ;
export const ANALOG_WRITE_TYPE = v0.ANALOG_WRITE_TYPE;
export const ZERO_ANALOG_WRITE_PAYLOAD = v0.ZERO_ANALOG_WRITE_PAYLOAD;
export const ANALOG_WRITE_SCHEMAS = v0.ANALOG_WRITE_SCHEMAS;
export type AnalogWriteSchemas = v0.AnalogWriteSchemas;

export const counterReadConfigZ = v0.counterReadConfigZ;
export const COUNTER_READ_TYPE = v0.COUNTER_READ_TYPE;
export const ZERO_COUNTER_READ_PAYLOAD = v0.ZERO_COUNTER_READ_PAYLOAD;
export const COUNTER_READ_SCHEMAS = v0.COUNTER_READ_SCHEMAS;
export type CounterReadSchemas = v0.CounterReadSchemas;

export const digitalReadConfigZ = v0.digitalReadConfigZ;
export const DIGITAL_READ_TYPE = v0.DIGITAL_READ_TYPE;
export const ZERO_DIGITAL_READ_PAYLOAD = v0.ZERO_DIGITAL_READ_PAYLOAD;
export const DIGITAL_READ_SCHEMAS = v0.DIGITAL_READ_SCHEMAS;
export type DigitalReadSchemas = v0.DigitalReadSchemas;

export const digitalWriteConfigZ = v0.digitalWriteConfigZ;
export const DIGITAL_WRITE_TYPE = v0.DIGITAL_WRITE_TYPE;
export const ZERO_DIGITAL_WRITE_PAYLOAD = v0.ZERO_DIGITAL_WRITE_PAYLOAD;
export const DIGITAL_WRITE_SCHEMAS = v0.DIGITAL_WRITE_SCHEMAS;
export type DigitalWriteSchemas = v0.DigitalWriteSchemas;

export const SCAN_TYPE = v0.SCAN_TYPE;

export const SCAN_SCHEMAS = v0.SCAN_SCHEMAS;
