// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import {
  convertPortTypeToReadChannelType,
  convertPortTypeToWriteChannelType,
  convertReadChannelTypeToPortType,
  convertWriteChannelTypeToPortType,
} from "@/feature/labjack/task/convertChannelTypeToPortType";

describe("convertReadChannelTypeToPortType", () => {
  it(`should convert "analog" to ${LabJack.Device.AI_PORT_TYPE}`, () => {
    expect(convertReadChannelTypeToPortType("analog")).toBe(
      LabJack.Device.AI_PORT_TYPE,
    );
  });

  it(`should convert "digital" to ${LabJack.Device.DI_PORT_TYPE}`, () => {
    expect(convertReadChannelTypeToPortType("digital")).toBe(
      LabJack.Device.DI_PORT_TYPE,
    );
  });

  it(`should convert "thermocouple" to ${LabJack.Device.AI_PORT_TYPE}`, () => {
    expect(convertReadChannelTypeToPortType("thermocouple")).toBe(
      LabJack.Device.AI_PORT_TYPE,
    );
  });
});

describe("convertWriteChannelTypeToPortType", () => {
  it(`should convert "analog" to ${LabJack.Device.AO_PORT_TYPE}`, () => {
    expect(convertWriteChannelTypeToPortType("analog")).toBe(
      LabJack.Device.AO_PORT_TYPE,
    );
  });

  it(`should convert "digital" to ${LabJack.Device.DO_PORT_TYPE}`, () => {
    expect(convertWriteChannelTypeToPortType("digital")).toBe(
      LabJack.Device.DO_PORT_TYPE,
    );
  });
});

describe("convertPortTypeToReadChannelType", () => {
  it(`should convert ${LabJack.Device.AI_PORT_TYPE} to "analog"`, () => {
    expect(convertPortTypeToReadChannelType(LabJack.Device.AI_PORT_TYPE)).toBe(
      "analog",
    );
  });

  it(`should convert ${LabJack.Device.DI_PORT_TYPE} to "digital"`, () => {
    expect(convertPortTypeToReadChannelType(LabJack.Device.DI_PORT_TYPE)).toBe(
      "digital",
    );
  });
});

describe("convertPortTypeToWriteChannelType", () => {
  it(`should convert ${LabJack.Device.AO_PORT_TYPE} to "analog"`, () => {
    expect(convertPortTypeToWriteChannelType(LabJack.Device.AO_PORT_TYPE)).toBe(
      "analog",
    );
  });

  it(`should convert ${LabJack.Device.DO_PORT_TYPE} to "digital"`, () => {
    expect(convertPortTypeToWriteChannelType(LabJack.Device.DO_PORT_TYPE)).toBe(
      "digital",
    );
  });
});
