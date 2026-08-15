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

describe("readStatusDataZ", () => {
  it("should accept null", () => {
    expect(LabJack.Task.READ_SCHEMAS.statusData.safeParse(null).success).toBe(true);
  });
  it("should accept undefined", () => {
    expect(LabJack.Task.READ_SCHEMAS.statusData.safeParse(undefined).success).toBe(
      true,
    );
  });
  it("should accept a valid errors object", () => {
    const result = LabJack.Task.READ_SCHEMAS.statusData.safeParse({
      errors: [{ message: "bad", path: "/dev/ai0" }],
    });
    expect(result.success).toBe(true);
  });
});

describe("readConfigZ", () => {
  const readConfigZ = LabJack.Task.READ_SCHEMAS.config;
  const deployReadConfigZ = LabJack.Task.deployReadConfigZ;
  it("should validate a valid read configuration", () => {
    const validConfig = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: "none" },
          range: 10,
        },
        {
          key: "2",
          channel: 2,
          disabled: false,
          type: "DI",
          name: "Test_DI_Channel",
          port: "DIO0",
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(validConfig);
    expect(result.success).toBe(true);
    expect(result.data?.channels).toHaveLength(2);
  });

  it("should mint a UUID key when the configuration lacks one", () => {
    const result = readConfigZ.safeParse({ device: "labjack" });
    expect(result.success).toBe(true);
    expect(result.data?.key).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/,
    );
  });

  it("should reject a configuration with duplicate ports", () => {
    const configWithDuplicatePorts = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel_1",
          port: "AIN0",
          scale: { type: "none" },
          range: 10,
        },
        {
          key: "2",
          channel: 2,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel_2",
          port: "AIN0", // Duplicate port
          scale: { type: "none" },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = deployReadConfigZ.safeParse(configWithDuplicatePorts);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("has already been used");
    }
  });

  it("should reject a configuration with sampleRate exceeding the maximum", () => {
    const configWithInvalidSampleRate = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: "none" },
          range: 10,
        },
      ],
      sampleRate: 60000, // Exceeds the max of 50000
      streamRate: 500,
    };

    const result = deployReadConfigZ.safeParse(configWithInvalidSampleRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("sampleRate");
    }
  });

  it("should reject a configuration with streamRate exceeding the maximum", () => {
    const configWithInvalidStreamRate = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: "none" },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 60000, // Exceeds the max of 50000
    };

    const result = deployReadConfigZ.safeParse(configWithInvalidStreamRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("streamRate");
    }
  });

  it("should reject a configuration with invalid stream rate refinement", () => {
    const configWithInvalidStreamRateRefinement = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "Test_AI_Channel",
          port: "AIN0",
          scale: { type: "none" },
          range: 10,
        },
      ],
      sampleRate: 100,
      streamRate: 200, // streamRate > sampleRate will violate the refinement
    };

    const result = deployReadConfigZ.safeParse(configWithInvalidStreamRateRefinement);
    expect(result.success).toBe(false);
  });

  it("should validate a configuration with linear scale", () => {
    const configWithLinearScale = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "ai_with_scale",
          port: "AIN0",
          scale: { type: "linear", slope: 2.5, offset: 0.5 },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(configWithLinearScale);
    expect(result.success).toBe(true);
  });

  it("should validate a configuration with map scale", () => {
    const configWithMapScale = {
      device: "labjack",
      channels: [
        {
          key: "1",
          channel: 1,
          disabled: false,
          type: "AI",
          name: "ai_with_map_scale",
          port: "AIN0",
          scale: {
            type: "map",
            preScaledMin: 0,
            preScaledMax: 10,
            scaledMin: 0,
            scaledMax: 100,
          },
          range: 10,
        },
      ],
      sampleRate: 1000,
      streamRate: 500,
    };

    const result = readConfigZ.safeParse(configWithMapScale);
    expect(result.success).toBe(true);
  });
});

describe("writeConfigZ", () => {
  const writeConfigZ = LabJack.Task.WRITE_SCHEMAS.config;
  const deployWriteConfigZ = LabJack.Task.deployWriteConfigZ;
  // Deploy validates form values, which are always shape-parsed first.
  const deployParse = (config: unknown) =>
    deployWriteConfigZ.safeParse(writeConfigZ.parse(config));
  it("should validate a valid write configuration", () => {
    const validConfig = {
      device: "labjack",
      channels: [
        {
          key: "1",
          disabled: false,
          type: "AO",
          cmdChannelName: "Test_AO_Channel",
          stateChannelName: "",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
        {
          key: "2",
          disabled: false,
          type: "DO",
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
      device: "labjack",
      channels: [
        {
          key: "1",
          disabled: false,
          type: "AO",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
        {
          key: "2",
          disabled: false,
          type: "AO",
          port: "DAC0", // Duplicate port
          cmdChannel: 3,
          stateChannel: 4,
        },
      ],
      stateRate: 1000,
    };

    const result = deployParse(configWithDuplicatePorts);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("has already been used");
    }
  });

  it("should reject a configuration with duplicate command channels", () => {
    const configWithDuplicateCmdChannels = {
      device: "labjack",
      channels: [
        {
          key: "1",
          disabled: false,
          type: "AO",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
        {
          key: "2",
          disabled: false,
          type: "DO",
          port: "DIO0",
          cmdChannel: 1, // Duplicate command channel
          stateChannel: 3,
        },
      ],
      stateRate: 1000,
    };

    const result = deployParse(configWithDuplicateCmdChannels);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("used on multiple channels");
    }
  });

  it("should reject a configuration with duplicate state channels", () => {
    const configWithDuplicateStateChannels = {
      device: "labjack",
      channels: [
        {
          key: "1",
          disabled: false,
          type: "AO",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
        {
          key: "2",
          disabled: false,
          type: "DO",
          port: "DIO0",
          cmdChannel: 3,
          stateChannel: 2, // Duplicate state channel
        },
      ],
      stateRate: 1000,
    };

    const result = deployParse(configWithDuplicateStateChannels);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].message).toContain("used for multiple channels");
    }
  });

  it("should reject a configuration with stateRate exceeding the maximum", () => {
    const configWithInvalidStateRate = {
      device: "labjack",
      channels: [
        {
          key: "1",
          disabled: false,
          type: "AO",
          port: "DAC0",
          cmdChannel: 1,
          stateChannel: 2,
        },
      ],
      stateRate: 60000, // Exceeds the max of 50000
    };

    const result = deployParse(configWithInvalidStateRate);
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(0);
      expect(result.error.issues[0].path).toContain("stateRate");
    }
  });

  it("should not migrate legacy v0 cmdKey shapes", () => {
    // The Core migrates stored configs; the console only speaks generated shapes, so
    // legacy cmdKey/stateKey fields are stripped rather than translated.
    const v0Config = {
      device: "labjack",
      stateRate: 1000,
      channels: [
        {
          key: "1",
          enabled: true,
          type: "AO",
          port: "DAC0",
          cmdKey: 1,
          stateKey: 2,
        },
      ],
    };

    const result = writeConfigZ.safeParse(v0Config);
    expect(result.success).toBe(true);
    const [ch] = result.data?.channels as LabJack.Task.OutputChannel[];
    expect(ch.cmdChannel).toBe(0);
    expect(ch.stateChannel).toBe(0);
    expect(ch).not.toHaveProperty("cmdKey");
    expect(ch).not.toHaveProperty("stateKey");
    expect(ch).not.toHaveProperty("enabled");
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every zero config; retrieve parses with it.
  it("should accept the zero read config", () => {
    expect(
      LabJack.Task.READ_SCHEMAS.config.safeParse(
        LabJack.Task.READ_SCHEMAS.config.parse({}),
      ).success,
    ).toBe(true);
  });
  it("should accept the zero write config", () => {
    expect(
      LabJack.Task.WRITE_SCHEMAS.config.safeParse(
        LabJack.Task.WRITE_SCHEMAS.config.parse({}),
      ).success,
    ).toBe(true);
  });
});
