// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";

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

export const LINEAR_SCALE_TYPE = v0.LINEAR_SCALE_TYPE;
export const MAP_SCALE_TYPE = v0.MAP_SCALE_TYPE;
export const TABLE_SCALE_TYPE = v0.TABLE_SCALE_TYPE;
export const NO_SCALE_TYPE = v0.NO_SCALE_TYPE;
export type Scale = v0.Scale;
export type ScaleType = v0.ScaleType;
export const ZERO_SCALES = v0.ZERO_SCALES;
export const SCALE_SCHEMAS = v0.SCALE_SCHEMAS;

export const SINE_WAVE_TYPE = v0.SINE_WAVE_TYPE;
export const TRIANGLE_WAVE_TYPE = v0.TRIANGLE_WAVE_TYPE;
export const SQUARE_WAVE_TYPE = v0.SQUARE_WAVE_TYPE;
export const SAWTOOTH_WAVE_TYPE = v0.SAWTOOTH_WAVE_TYPE;
export const WAVE_TYPES = v0.WAVE_TYPES;
export type WaveType = v0.WaveType;

export type AIChannel = v1.AIChannel;
export type AIChannelType = v0.AIChannelType;
export const AI_CHANNEL_SCHEMAS = v1.AI_CHANNEL_SCHEMAS;
export const ZERO_AI_CHANNELS = v1.ZERO_AI_CHANNELS;
export const ZERO_AI_CHANNEL = v1.ZERO_AI_CHANNEL;
export const AI_CHANNEL_TYPE_NAMES = v0.AI_CHANNEL_TYPE_NAMES;
export const AI_CHANNEL_TYPE_ICONS = v0.AI_CHANNEL_TYPE_ICONS;

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

export const analogReadTypeZ = v0.analogReadTypeZ;
export const analogReadConfigZ = v1.analogReadConfigZ;
export const analogReadStatusDataZ = v0.analogReadStatusDataZ;
export interface AnalogReadConfig extends v1.AnalogReadConfig {}
export const ANALOG_READ_TYPE = v0.ANALOG_READ_TYPE;
export type AnalogReadType = v0.AnalogReadType;
export const ZERO_ANALOG_READ_PAYLOAD = v1.ZERO_ANALOG_READ_PAYLOAD;
export interface AnalogReadPayload extends v1.AnalogReadPayload {}
export interface AnalogReadTask extends v1.AnalogReadTask {}
export interface NewAnalogReadTask extends v1.NewAnalogReadTask {}
export const ANALOG_READ_SCHEMAS: task.Schemas<
  typeof analogReadTypeZ,
  typeof analogReadConfigZ,
  typeof analogReadStatusDataZ
> = {
  typeSchema: analogReadTypeZ,
  configSchema: analogReadConfigZ,
  statusDataSchema: analogReadStatusDataZ,
};

export const analogWriteTypeZ = v0.analogWriteTypeZ;
export const analogWriteConfigZ = v0.analogWriteConfigZ;
export const analogWriteStatusDataZ = v0.analogWriteStatusDataZ;
export interface AnalogWriteConfig extends v0.AnalogWriteConfig {}
export const ANALOG_WRITE_TYPE = v0.ANALOG_WRITE_TYPE;
export type AnalogWriteType = v0.AnalogWriteType;
export const ZERO_ANALOG_WRITE_PAYLOAD = v0.ZERO_ANALOG_WRITE_PAYLOAD;
export interface AnalogWritePayload extends v0.AnalogWritePayload {}
export interface AnalogWriteTask extends v0.AnalogWriteTask {}
export interface NewAnalogWriteTask extends v0.NewAnalogWriteTask {}
export const ANALOG_WRITE_SCHEMAS: task.Schemas<
  typeof analogWriteTypeZ,
  typeof analogWriteConfigZ,
  typeof analogWriteStatusDataZ
> = {
  typeSchema: analogWriteTypeZ,
  configSchema: analogWriteConfigZ,
  statusDataSchema: analogWriteStatusDataZ,
};

export const digitalReadTypeZ = v0.digitalReadTypeZ;
export const digitalReadConfigZ = v0.digitalReadConfigZ;
export const digitalReadStatusDataZ = v0.digitalReadStatusDataZ;
export interface DigitalReadConfig extends v0.DigitalReadConfig {}
export const DIGITAL_READ_TYPE = v0.DIGITAL_READ_TYPE;
export type DigitalReadType = v0.DigitalReadType;
export const ZERO_DIGITAL_READ_PAYLOAD = v0.ZERO_DIGITAL_READ_PAYLOAD;
export interface DigitalReadPayload extends v0.DigitalReadPayload {}
export interface DigitalReadTask extends v0.DigitalReadTask {}
export interface NewDigitalReadTask extends v0.NewDigitalReadTask {}
export const DIGITAL_READ_SCHEMAS: task.Schemas<
  typeof digitalReadTypeZ,
  typeof digitalReadConfigZ,
  typeof digitalReadStatusDataZ
> = {
  typeSchema: digitalReadTypeZ,
  configSchema: digitalReadConfigZ,
  statusDataSchema: digitalReadStatusDataZ,
};

export const digitalWriteTypeZ = v0.digitalWriteTypeZ;
export const digitalWriteConfigZ = v0.digitalWriteConfigZ;
export const digitalWriteStatusDataZ = v0.digitalWriteStatusDataZ;
export interface DigitalWriteConfig extends v0.DigitalWriteConfig {}
export const DIGITAL_WRITE_TYPE = v0.DIGITAL_WRITE_TYPE;
export type DigitalWriteType = v0.DigitalWriteType;
export const ZERO_DIGITAL_WRITE_PAYLOAD = v0.ZERO_DIGITAL_WRITE_PAYLOAD;
export interface DigitalWritePayload extends v0.DigitalWritePayload {}
export interface DigitalWriteTask extends v0.DigitalWriteTask {}
export interface NewDigitalWriteTask extends v0.NewDigitalWriteTask {}
export const DIGITAL_WRITE_SCHEMAS: task.Schemas<
  typeof digitalWriteTypeZ,
  typeof digitalWriteConfigZ,
  typeof digitalWriteStatusDataZ
> = {
  typeSchema: digitalWriteTypeZ,
  configSchema: digitalWriteConfigZ,
  statusDataSchema: digitalWriteStatusDataZ,
};

export const scanTypeZ = v0.scanTypeZ;
export const scanConfigZ = v0.scanConfigZ;
export const scanStatusDataZ = v0.scanStatusDataZ;
export interface ScanConfig extends v0.ScanConfig {}
export const SCAN_TYPE = v0.SCAN_TYPE;
export type ScanType = v0.ScanType;
export interface ScanPayload extends v0.ScanPayload {}
export interface ScanTask extends v0.ScanTask {}
export interface NewScanTask extends v0.NewScanTask {}
export const SCAN_SCHEMAS: task.Schemas<
  typeof scanTypeZ,
  typeof scanConfigZ,
  typeof scanStatusDataZ
> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: scanStatusDataZ,
};
