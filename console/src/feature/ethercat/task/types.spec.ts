// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { assert, describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createAutoReadChannel,
  createAutoWriteChannel,
  createManualReadChannel,
  createPDOs,
} from "@/feature/ethercat/testutil";

const createLocalSlave = (): EtherCAT.Device.SlaveDevice => ({
  key: "slave_key",
  rack: 1,
  location: "loc",
  make: EtherCAT.Device.MAKE,
  model: EtherCAT.Device.SLAVE_MODEL,
  name: "Slave",
  configured: true,
  properties: {
    ...EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
    name: "slave_props_name",
    pdos: createPDOs(),
  },
});

describe("channelMapKey", () => {
  it("should key automatic channels by their PDO", () => {
    const ch = createAutoReadChannel("dev", "Status");
    expect(EtherCAT.Task.channelMapKey(ch)).toBe("auto_Status");
  });

  it("should key manual channels by index and subindex", () => {
    const ch = createManualReadChannel("dev", 0x6000, 3);
    expect(EtherCAT.Task.channelMapKey(ch)).toBe(`manual_${0x6000}_3`);
  });
});

describe("createReadChannel", () => {
  it("should start from the automatic zero channel when the list is empty", () => {
    const ch = EtherCAT.Task.createReadChannel([]);
    expect(ch.type).toBe("automatic");
    expect(ch.key).not.toBe("");
  });

  it("should copy the last channel but reset its synnax channel binding", () => {
    const last = createAutoReadChannel("dev", "Status", {
      channel: 42,
      name: "copied",
    });
    const ch = EtherCAT.Task.createReadChannel([last]);
    expect(ch.device).toBe("dev");
    expect(ch.channel).toBe(0);
    expect(ch.key).not.toBe(last.key);
  });
});

describe("createWriteChannel", () => {
  it("should start from the automatic zero channel when the list is empty", () => {
    const ch = EtherCAT.Task.createWriteChannel([]);
    expect(ch.type).toBe("automatic");
    expect(ch.key).not.toBe("");
  });

  it("should copy the last channel but reset names and channel bindings", () => {
    const last = createAutoWriteChannel("dev", "Control", {
      cmdChannel: 5,
      stateChannel: 6,
      cmdChannelName: "cmd",
      stateChannelName: "state",
      name: "copied",
    });
    const ch = EtherCAT.Task.createWriteChannel([last]);
    expect(ch.device).toBe("dev");
    expect(ch.cmdChannel).toBe(0);
    expect(ch.stateChannel).toBe(0);
    expect(ch.cmdChannelName).toBe("");
    expect(ch.stateChannelName).toBe("");
    expect(ch.name).toBe("");
    expect(ch.key).not.toBe(last.key);
  });
});

describe("getChannelByMapKey", () => {
  it("should return the channel stored under the exact key", () => {
    expect(EtherCAT.Task.getChannelByMapKey({ auto_Status: 7 }, "auto_Status")).toBe(7);
  });

  it("should fall back to the camelized key the wire codec produces", () => {
    expect(
      EtherCAT.Task.getChannelByMapKey({ auto_ControlState: 9 }, "auto_Control_state"),
    ).toBe(9);
  });

  it("should return zero when neither key is present", () => {
    expect(EtherCAT.Task.getChannelByMapKey({}, "auto_Status")).toBe(0);
  });
});

describe("resolvePDODataType", () => {
  it("should resolve the data type of a known input PDO", () => {
    expect(
      EtherCAT.Task.resolvePDODataType(createLocalSlave(), "Status", "inputs"),
    ).toBe("uint16");
  });

  it("should throw an UnexpectedError for an unknown PDO", () => {
    let caught: unknown;
    try {
      EtherCAT.Task.resolvePDODataType(createLocalSlave(), "Missing", "outputs");
    } catch (e) {
      caught = e;
    }
    expect(UnexpectedError.matches(caught)).toBe(true);
  });
});

describe("getPortLabel", () => {
  it("should label automatic channels with their PDO name", () => {
    expect(EtherCAT.Task.getPortLabel(createAutoReadChannel("dev", "Status"))).toBe(
      "Status",
    );
  });

  it("should label automatic channels without a PDO as No PDO", () => {
    expect(EtherCAT.Task.getPortLabel(createAutoReadChannel("dev", ""))).toBe(
      "No PDO",
    );
  });

  it("should label manual channels with a padded hex address", () => {
    expect(EtherCAT.Task.getPortLabel(createManualReadChannel("dev", 0x60, 2))).toBe(
      "0x0060:2",
    );
  });
});

describe("getPDOName", () => {
  it("should use the PDO name for automatic channels", () => {
    expect(EtherCAT.Task.getPDOName(createAutoReadChannel("dev", "Status"))).toBe(
      "Status",
    );
  });

  it("should build an escaped hex address name for manual channels", () => {
    expect(EtherCAT.Task.getPDOName(createManualReadChannel("dev", 0x6000, 2))).toBe(
      "_0x6000_2",
    );
  });

  it("should escape characters that are invalid in channel names", () => {
    const ch = createAutoReadChannel("dev", "Status Word 1");
    expect(EtherCAT.Task.getPDOName(ch)).not.toContain(" ");
  });
});

describe("EtherCAT Task statusData", () => {
  describe("readStatusDataZ", () => {
    it("should accept null", () => {
      expect(EtherCAT.Task.READ_SCHEMAS.statusData.safeParse(null).success).toBe(true);
    });
    it("should accept undefined", () => {
      expect(EtherCAT.Task.READ_SCHEMAS.statusData.safeParse(undefined).success).toBe(
        true,
      );
    });
    it("should accept a valid status object", () => {
      const result = EtherCAT.Task.READ_SCHEMAS.statusData.safeParse({
        running: true,
        message: "ok",
        errors: [{ message: "err", path: "/dev" }],
      });
      expect(result.success).toBe(true);
    });
  });

  describe("writeStatusDataZ", () => {
    it("should accept null", () => {
      expect(EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse(null).success).toBe(true);
    });
    it("should accept undefined", () => {
      expect(EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse(undefined).success).toBe(
        true,
      );
    });
    it("should accept a valid status object", () => {
      const result = EtherCAT.Task.WRITE_SCHEMAS.statusData.safeParse({
        running: false,
        message: "ok",
        errors: [{ message: "err", path: "/dev" }],
      });
      expect(result.success).toBe(true);
    });
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every zero config; retrieve parses with it.
  it("should accept the zero read config", () => {
    expect(
      EtherCAT.Task.READ_SCHEMAS.config.safeParse(
        EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      ).success,
    ).toBe(true);
  });
  it("should accept the zero write config", () => {
    expect(
      EtherCAT.Task.WRITE_SCHEMAS.config.safeParse(
        EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      ).success,
    ).toBe(true);
  });
});

describe("legacy shapes", () => {
  it("should not migrate legacy v0 enabled/subindex shapes", () => {
    // The Core migrates stored configs; the console only speaks generated shapes, so
    // legacy enabled, subindex, and dataSaving fields are stripped rather than
    // translated.
    const v0Config = {
      sampleRate: 1000,
      streamRate: 25,
      dataSaving: true,
      channels: [
        {
          key: "1",
          type: "manual",
          device: "dev",
          enabled: true,
          index: 0x6000,
          subindex: 5,
          bitLength: 16,
          dataType: "uint16",
          channel: 0,
          name: "",
        },
      ],
    };
    const result = EtherCAT.Task.READ_SCHEMAS.config.safeParse(v0Config);
    expect(result.success).toBe(true);
    const [ch] = result.data?.channels as EtherCAT.Task.ReadChannel[];
    assert(ch.type === "manual");
    expect(ch.disabled).toBe(false);
    expect(ch.subIndex).toBe(0);
    expect(ch).not.toHaveProperty("enabled");
    expect(ch).not.toHaveProperty("subindex");
    expect(result.data).not.toHaveProperty("dataSaving");
    expect(result.data?.dataSavingDisabled).toBe(false);
  });
});
