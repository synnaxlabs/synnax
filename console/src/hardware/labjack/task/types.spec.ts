// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Common } from "@/hardware/common";
import {
  AI_CHANNEL_TYPE,
  AO_CHANNEL_TYPE,
  DI_CHANNEL_TYPE,
  DO_CHANNEL_TYPE,
  LINEAR_SCALE_TYPE,
  NO_SCALE_TYPE,
  type OutputChannel,
  type ReadConfig,
  readConfigZ,
  type WriteConfig,
  writeConfigZ,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/labjack/task/types";

describe("readConfigZ", () => {
  it("should validate a valid read configuration", () => {
    const validConfig = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
        {
          key: "2",
          channel: 2,
          enabled: true,
          type: DI_CHANNEL_TYPE,
          name: "Test_DI_Channel",
          port: "DIO0",
          scale: { type: NO_SCALE_TYPE },
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(validConfig);
    expect(result.success).toBe(true);
  });

  it("should reject a configuration with duplicate ports", () => {
    const configWithDuplicatePorts = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel_1",
          port: "AIN0",
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
        {
          key: "2",
          channel: 2,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel_2",
          port: "AIN0", // Duplicate port
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(configWithDuplicatePorts);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("has already been used");
    }
  });

  it("should reject a configuration with sampleRate exceeding the maximum", () => {
    const configWithInvalidSampleRate = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
      ],
      sampleRate: 60000, // Exceeds the max of 50000
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(configWithInvalidSampleRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("sampleRate");
    }
  });

  it("should reject a configuration with streamRate exceeding the maximum", () => {
    const configWithInvalidStreamRate = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 60000, // Exceeds the max of 50000
    };

    const result = readConfigZ.safeParse(configWithInvalidStreamRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("streamRate");
    }
  });

  it("should reject a configuration with invalid stream rate refinement", () => {
    const configWithInvalidStreamRateRefinement: ReadConfig = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      dataSaving: true,
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: NO_SCALE_TYPE },
          range: 10,
        },
      ],
      sampleRate: 100,
      streamRate: 200, // streamRate > sampleRate will violate the refinement
    };

    const result = readConfigZ.safeParse(configWithInvalidStreamRateRefinement);
    expect(result.success).toBe(false);
  });

  it("should validate a configuration with linear scale", () => {
    const configWithLinearScale: ReadConfig = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          enabled: true,
          type: AI_CHANNEL_TYPE,
          name: "ai_with_scale",
          port: "AIN0",
          scale: {
            type: LINEAR_SCALE_TYPE,
            slope: 2.5,
            offset: 0.5,
          },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
      dataSaving: true,
    };

    const result = readConfigZ.safeParse(configWithLinearScale);
    expect(result.success).toBe(true);
  });
});

describe("writeConfigZ", () => {
  it("should validate a valid write configuration", () => {
    const validConfig: WriteConfig = {
      ...ZERO_WRITE_PAYLOAD.config,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          cmdChannelName: "Test_AO_Channel",
          stateChannelName: "",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
        {
          key: "2",
          enabled: true,
          type: DO_CHANNEL_TYPE,
          cmdChannelName: "Test_DO_Channel",
          stateChannelName: "",
          port: "DIO0",
          cmdChannel: 3,
          stateChannel: 4,
        },
      ],
      stateRate: 1000,
    };

    const result = writeConfigZ.safeParse(validConfig);
    expect(result.success).toBe(true);
  });

  it("should reject a configuration with duplicate ports", () => {
    const configWithDuplicatePorts = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          name: "Test_AO_Channel_1",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
          scale: { type: NO_SCALE_TYPE },
        },
        {
          key: "2",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          name: "Test_AO_Channel_2",
          port: "DAC0", // Duplicate port
          cmdKey: 3,
          stateKey: 4,
          scale: { type: NO_SCALE_TYPE },
        },
      ],
      stateRate: 1000,
    };

    const result = writeConfigZ.safeParse(configWithDuplicatePorts);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("has already been used");
    }
  });

  it("should reject a configuration with duplicate cmdKeys", () => {
    const configWithDuplicateCmdKeys = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          cmdName: "Test_AO_Channel_1",
          stateName: "",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
          scale: { type: NO_SCALE_TYPE },
        },
        {
          key: "2",
          enabled: true,
          type: DO_CHANNEL_TYPE,
          cmdName: "Test_DO_Channel",
          stateName: "",
          port: "DIO0",
          cmdKey: 1, // Duplicate cmdKey
          stateKey: 3,
          scale: { type: NO_SCALE_TYPE },
        },
      ],
      stateRate: 1000,
    };

    const result = writeConfigZ.safeParse(configWithDuplicateCmdKeys);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("used on multiple channels");
    }
  });

  it("should reject a configuration with duplicate stateKeys", () => {
    const configWithDuplicateStateKeys = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          name: "Test AO Channel",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
          scale: { type: NO_SCALE_TYPE },
        },
        {
          key: "2",
          enabled: true,
          type: DO_CHANNEL_TYPE,
          name: "Test DO Channel",
          port: "DIO0",
          cmdKey: 3,
          stateKey: 2, // Duplicate stateKey
          scale: { type: NO_SCALE_TYPE },
        },
      ],
      stateRate: 1000,
    };

    const result = writeConfigZ.safeParse(configWithDuplicateStateKeys);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("used for multiple channels");
    }
  });

  it("should reject a configuration with stateRate exceeding the maximum", () => {
    const configWithInvalidStateRate = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          name: "Test AO Channel",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
          scale: { type: NO_SCALE_TYPE },
        },
      ],
      stateRate: 60000, // Exceeds the max of 50000
    };

    const result = writeConfigZ.safeParse(configWithInvalidStateRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("stateRate");
    }
  });

  it("should validate a configuration with linear scale", () => {
    const configWithLinearScale = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      channels: [
        {
          key: "1",
          enabled: true,
          type: AO_CHANNEL_TYPE,
          name: "Test AO Channel",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
          scale: {
            type: LINEAR_SCALE_TYPE,
            slope: 2.5,
            intercept: 0.5,
            unit: "V",
          },
        },
      ],
      stateRate: 1000,
    };

    const result = writeConfigZ.safeParse(configWithLinearScale);
    expect(result.success).toBe(true);
  });

  it("should move a v0 configuration to the new format", () => {
    const inputChannels = [
      {
        key: "1",
        enabled: true,
        type: "AO",
        port: "DAC0",
        name: "dac0_no_scale",
        cmdKey: 1,
        stateKey: 2,
        scale: { type: "NO_SCALE" },
      },
      {
        key: "2",
        enabled: true,
        type: "DO",
        port: "DIO0",
        name: "dio0_noscale",
        cmdKey: 3,
        stateKey: 4,
        scale: { type: "NO_SCALE" },
      },
    ];
    const v0Config = {
      ...Common.Task.ZERO_BASE_CONFIG,
      device: "labjack",
      stateRate: 1000,
      channels: inputChannels,
    };

    const result = writeConfigZ.safeParse(v0Config);
    expect(result.success).toBe(true);
    expect(result.data?.channels.length).toBe(2);
    const channels = result.data?.channels as OutputChannel[];
    channels.forEach((ch, i) => {
      expect(ch.cmdChannel).toBe(inputChannels[i].cmdKey);
      expect(ch.stateChannel).toBe(inputChannels[i].stateKey);
      expect(ch).not.toHaveProperty("cmdKey");
      expect(ch).not.toHaveProperty("stateKey");
    });
  });
});
