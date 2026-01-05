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
import {
  AI_CHANNEL_TYPE,
  AO_CHANNEL_TYPE,
  DI_CHANNEL_TYPE,
  DO_CHANNEL_TYPE,
  TC_CHANNEL_TYPE,
} from "@/hardware/labjack/task/types";

describe("convertChannelTypeToPortType", () => {
  it(`should convert ${AI_CHANNEL_TYPE} to ${Device.AI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType(AI_CHANNEL_TYPE);
    expect(result).toBe(Device.AI_PORT_TYPE);
  });

  it(`should convert ${AO_CHANNEL_TYPE} to ${Device.AO_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType(AO_CHANNEL_TYPE);
    expect(result).toBe(Device.AO_PORT_TYPE);
  });

  it(`should convert ${DI_CHANNEL_TYPE} to ${Device.DI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType(DI_CHANNEL_TYPE);
    expect(result).toBe(Device.DI_PORT_TYPE);
  });

  it(`should convert ${DO_CHANNEL_TYPE} to ${Device.DO_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType(DO_CHANNEL_TYPE);
    expect(result).toBe(Device.DO_PORT_TYPE);
  });

  it(`should convert ${TC_CHANNEL_TYPE} to ${Device.AI_PORT_TYPE}`, () => {
    const result = convertChannelTypeToPortType(TC_CHANNEL_TYPE);
    expect(result).toBe(Device.AI_PORT_TYPE);
  });
});
