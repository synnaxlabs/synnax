// Copyright 2025 Synnax Labs, Inc.
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
export type WaveType = v0.WaveType;

export type AIChannel = v1.AIChannel;
export type AIChannelType = v0.AIChannelType;
export const AI_CHANNEL_SCHEMAS = v1.AI_CHANNEL_SCHEMAS;
export const ZERO_AI_CHANNELS = v1.ZERO_AI_CHANNELS;
export const ZERO_AI_CHANNEL = v1.ZERO_AI_CHANNEL;
export const AI_CHANNEL_TYPE_NAMES = v0.AI_CHANNEL_TYPE_NAMES;

export const AO_CURRENT_CHAN_TYPE = v0.AO_CURRENT_CHAN_TYPE;
export const AO_FUNC_GEN_CHAN_TYPE = v0.AO_FUNC_GEN_CHAN_TYPE;
export const AO_VOLTAGE_CHAN_TYPE = v0.AO_VOLTAGE_CHAN_TYPE;
export type AOChannel = v0.AOChannel;
export type AOChannelType = v0.AOChannelType;
export const AO_CHANNEL_SCHEMAS = v0.AO_CHANNEL_SCHEMAS;
export const ZERO_AO_CHANNELS = v0.ZERO_AO_CHANNELS;
export const ZERO_AO_CHANNEL = v0.ZERO_AO_CHANNEL;
export const AO_CHANNEL_TYPE_NAMES = v0.AO_CHANNEL_TYPE_NAMES;

export type AnalogChannel = v1.AnalogChannel;

export interface DIChannel extends v0.DIChannel {}
export const ZERO_DI_CHANNEL = v0.ZERO_DI_CHANNEL;

export interface DOChannel extends v0.DOChannel {}
export const ZERO_DO_CHANNEL = v0.ZERO_DO_CHANNEL;

export type DigitalChannel = v0.DigitalChannel;

export type Channel = v1.Channel;

export const analogReadConfigZ = v1.analogReadConfigZ;
export interface AnalogReadConfig extends v1.AnalogReadConfig {}
export interface AnalogReadStateDetails extends v0.AnalogReadStateDetails {}
export interface AnalogReadState extends v0.AnalogReadState {}
export const ANALOG_READ_TYPE = v0.ANALOG_READ_TYPE;
export type AnalogReadType = v0.AnalogReadType;
export const ZERO_ANALOG_READ_PAYLOAD = v1.ZERO_ANALOG_READ_PAYLOAD;
export interface AnalogReadPayload extends v1.AnalogReadPayload {}
export interface AnalogReadTask extends v1.AnalogReadTask {}
export interface NewAnalogReadTask extends v1.NewAnalogReadTask {}

export const analogWriteConfigZ = v0.analogWriteConfigZ;
export interface AnalogWriteConfig extends v0.AnalogWriteConfig {}
export interface AnalogWriteStateDetails extends v0.AnalogWriteStateDetails {}
export interface AnalogWriteState extends v0.AnalogWriteState {}
export const ANALOG_WRITE_TYPE = v0.ANALOG_WRITE_TYPE;
export type AnalogWriteType = v0.AnalogWriteType;
export const ZERO_ANALOG_WRITE_PAYLOAD = v0.ZERO_ANALOG_WRITE_PAYLOAD;
export interface AnalogWritePayload extends v0.AnalogWritePayload {}
export interface AnalogWriteTask extends v0.AnalogWriteTask {}
export interface NewAnalogWriteTask extends v0.NewAnalogWriteTask {}

export const digitalReadConfigZ = v0.digitalReadConfigZ;
export interface DigitalReadConfig extends v0.DigitalReadConfig {}
export interface DigitalReadStateDetails extends v0.DigitalReadStateDetails {}
export interface DigitalReadState extends v0.DigitalReadState {}
export const DIGITAL_READ_TYPE = v0.DIGITAL_READ_TYPE;
export type DigitalReadType = v0.DigitalReadType;
export const ZERO_DIGITAL_READ_PAYLOAD = v0.ZERO_DIGITAL_READ_PAYLOAD;
export interface DigitalReadPayload extends v0.DigitalReadPayload {}
export interface DigitalReadTask extends v0.DigitalReadTask {}
export interface NewDigitalReadTask extends v0.NewDigitalReadTask {}

export const digitalWriteConfigZ = v0.digitalWriteConfigZ;
export interface DigitalWriteConfig extends v0.DigitalWriteConfig {}
export interface DigitalWriteStateDetails extends v0.DigitalWriteStateDetails {}
export interface DigitalWriteState extends v0.DigitalWriteState {}
export const DIGITAL_WRITE_TYPE = v0.DIGITAL_WRITE_TYPE;
export type DigitalWriteType = v0.DigitalWriteType;
export const ZERO_DIGITAL_WRITE_PAYLOAD = v0.ZERO_DIGITAL_WRITE_PAYLOAD;
export interface DigitalWritePayload extends v0.DigitalWritePayload {}
export interface DigitalWriteTask extends v0.DigitalWriteTask {}
export interface NewDigitalWriteTask extends v0.NewDigitalWriteTask {}

export interface ScanConfig extends v0.ScanConfig {}
export interface ScanStateDetails extends v0.ScanStateDetails {}
export interface ScanState extends v0.ScanState {}
export const SCAN_TYPE = v0.SCAN_TYPE;
export type ScanType = v0.ScanType;
export interface ScanPayload extends v0.ScanPayload {}
export interface ScanTask extends v0.ScanTask {}
export interface NewScanTask extends v0.NewScanTask {}
