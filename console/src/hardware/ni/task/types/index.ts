// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/hardware/ni/task/types/v0";
import * as v1 from "@/hardware/ni/task/types/v1";

export const PREFIX = v0.PREFIX;

// Units

export type AccelSensitivityUnits = v0.AccelSensitivityUnits;
export type AccelerationUnits = v0.AccelUnits;
export type ForceUnits = v0.ForceUnits;
export type ElectricalUnits = v0.ElectricalUnits;
export type ShuntResistorLoc = v0.ShuntResistorLoc;
export type PressureUnits = v0.PressureUnits;
export type TemperatureUnits = v0.TemperatureUnits;
export type TorqueUnits = v0.TorqueUnits;
export type VelocityUnits = v0.VelocityUnits;
export type VelocitySensitivityUnits = v0.VelocitySensitivityUnits;
export type Units = v0.Units;

// Scales

export type Scale = v0.Scale;
export type ScaleType = v0.ScaleType;
export const ZERO_SCALES = v0.ZERO_SCALES;
export const SCALE_SCHEMAS = v0.SCALE_SCHEMAS;

// Wave Types

export type WaveType = v0.WaveType;

// Channels

export type AIChannel = v1.AIChannel;
export type AIChannelType = v0.AIChannelType;
export const AI_CHANNEL_SCHEMAS = v1.AI_CHANNEL_SCHEMAS;
export const ZERO_AI_CHANNELS = v1.ZERO_AI_CHANNELS;
export const AI_CHANNEL_TYPE_NAMES = v0.AI_CHANNEL_TYPE_NAMES;
export type AOChannel = v0.AOChannel;
export type AOChannelType = v0.AOChannelType;
export const AO_CHANNEL_SCHEMAS = v0.AO_CHANNEL_SCHEMAS;
export const ZERO_AO_CHANNELS = v0.ZERO_AO_CHANNELS;
export const AO_CHANNEL_TYPE_NAMES = v0.AO_CHANNEL_TYPE_NAMES;
export const ZERO_DO_CHANNEL = v0.ZERO_DO_CHANNEL;
export interface DOChannel extends v0.DOChannel {}
export const ZERO_DI_CHANNEL = v0.ZERO_DI_CHANNEL;
export interface DIChannel extends v0.DIChannel {}

// Tasks

export const analogReadConfigZ = v1.analogReadConfigZ;
export interface AnalogReadConfig extends v1.AnalogReadConfig {}
export type AnalogReadDetails = v0.AnalogReadDetails;
export const ANALOG_READ_TYPE = v0.ANALOG_READ_TYPE;
export type AnalogReadType = v0.AnalogReadType;
export interface AnalogRead extends v1.AnalogRead {}
export interface AnalogReadPayload extends v1.AnalogReadPayload {}
export const ZERO_ANALOG_READ_PAYLOAD = v1.ZERO_ANALOG_READ_PAYLOAD;

export const analogWriteConfigZ = v0.analogWriteConfigZ;
export interface AnalogWriteConfig extends v0.AnalogWriteConfig {}
export interface AnalogWriteDetails extends v0.AnalogWriteDetails {}
export const ANALOG_WRITE_TYPE = v0.ANALOG_WRITE_TYPE;
export type AnalogWriteType = v0.AnalogWriteType;
export interface AnalogWrite extends v0.AnalogWrite {}
export interface AnalogWritePayload extends v0.AnalogWritePayload {}
export const ZERO_ANALOG_WRITE_PAYLOAD = v0.ZERO_ANALOG_WRITE_PAYLOAD;

export const digitalWriteConfigZ = v0.digitalWriteConfigZ;
export interface DigitalWriteConfig extends v0.DigitalWriteConfig {}
export interface DigitalWriteDetails extends v0.DigitalWriteDetails {}
export const DIGITAL_WRITE_TYPE = v0.DIGITAL_WRITE_TYPE;
export type DigitalWriteType = v0.DigitalWriteType;
export interface DigitalWrite extends v0.DigitalWrite {}
export interface DigitalWritePayload extends v0.DigitalWritePayload {}
export const ZERO_DIGITAL_WRITE_PAYLOAD = v0.ZERO_DIGITAL_WRITE_PAYLOAD;

export const digitalReadConfigZ = v0.digitalReadConfigZ;
export interface DigitalReadConfig extends v0.DigitalReadConfig {}
export interface DigitalReadDetails extends v0.DigitalReadDetails {}
export const DIGITAL_READ_TYPE = v0.DIGITAL_READ_TYPE;
export type DigitalReadType = v0.DigitalReadType;
export interface DigitalRead extends v0.DigitalRead {}
export interface DigitalReadPayload extends v0.DigitalReadPayload {}
export const ZERO_DIGITAL_READ_PAYLOAD = v0.ZERO_DIGITAL_READ_PAYLOAD;

export interface ScanConfig extends v0.ScanConfig {}

// Migrations

type AnyAnalogReadConfig = v0.AnalogReadConfig | v1.AnalogReadConfig;
const ANALOG_READ_CONFIG_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.analogReadConfigMigration,
};
export const migrateAnalogReadConfig = migrate.migrator<
  AnyAnalogReadConfig,
  v1.AnalogReadConfig
>({
  name: v1.ANALOG_READ_CONFIG_MIGRATION_NAME,
  migrations: ANALOG_READ_CONFIG_MIGRATIONS,
  def: v1.ZERO_ANALOG_READ_CONFIG,
  defaultVersion: v0.VERSION,
});
