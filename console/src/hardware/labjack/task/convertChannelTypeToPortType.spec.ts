// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Device } from "@/hardware/labjack/device";
import { convertChannelTypeToPortType } from "@/hardware/labjack/task/convertChannelTypeToPortType";

describe("convertChannelTypeToPortType", () => {
  it(`should convert "AI" to ${Device.AI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType("AI");
    expect(result).toBe(Device.AI_PORT_TYPE);
  });

  it(`should convert "AO" to ${Device.AO_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType("AO");
    expect(result).toBe(Device.AO_PORT_TYPE);
  });

  it(`should convert "DI" to ${Device.DI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType("DI");
    expect(result).toBe(Device.DI_PORT_TYPE);
  });

  it(`should convert "DO" to ${Device.DO_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType("DO");
    expect(result).toBe(Device.DO_PORT_TYPE);
  });

  it(`should convert "TC" to ${Device.AI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType("TC");
    expect(result).toBe(Device.AI_PORT_TYPE);
  });
});
