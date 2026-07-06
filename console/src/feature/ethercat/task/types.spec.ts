// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createAutoInputChannel,
  createAutoOutputChannel,
  createManualInputChannel,
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
    const ch = createAutoInputChannel("dev", "Status");
    expect(EtherCAT.Task.channelMapKey(ch)).toBe("auto_Status");
  });

  it("should key manual channels by index and subindex", () => {
    const ch = createManualInputChannel("dev", 0x6000, 3);
    expect(EtherCAT.Task.channelMapKey(ch)).toBe(`manual_${0x6000}_3`);
  });
});

describe("createInputChannel", () => {
  it("should start from the automatic zero channel when the list is empty", () => {
    const ch = EtherCAT.Task.createInputChannel([]);
    expect(ch.type).toBe("automatic");
    expect(ch.key).not.toBe("");
  });

  it("should copy the last channel but reset its synnax channel binding", () => {
    const last = createAutoInputChannel("dev", "Status", {
      channel: 42,
      name: "copied",
    });
    const ch = EtherCAT.Task.createInputChannel([last]);
    expect(ch.device).toBe("dev");
    expect(ch.channel).toBe(0);
    expect(ch.key).not.toBe(last.key);
  });
});

describe("createOutputChannel", () => {
  it("should start from the automatic zero channel when the list is empty", () => {
    const ch = EtherCAT.Task.createOutputChannel([]);
    expect(ch.type).toBe("automatic");
    expect(ch.key).not.toBe("");
  });

  it("should copy the last channel but reset names and channel bindings", () => {
    const last = createAutoOutputChannel("dev", "Control", {
      cmdChannel: 5,
      stateChannel: 6,
      cmdChannelName: "cmd",
      stateChannelName: "state",
      name: "copied",
    });
    const ch = EtherCAT.Task.createOutputChannel([last]);
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
    expect(EtherCAT.Task.getPortLabel(createAutoInputChannel("dev", "Status"))).toBe(
      "Status",
    );
  });

  it("should label automatic channels without a PDO as No PDO", () => {
    expect(EtherCAT.Task.getPortLabel(createAutoInputChannel("dev", ""))).toBe(
      "No PDO",
    );
  });

  it("should label manual channels with a padded hex address", () => {
    expect(EtherCAT.Task.getPortLabel(createManualInputChannel("dev", 0x60, 2))).toBe(
      "0x0060:2",
    );
  });
});

describe("getPDOName", () => {
  it("should use the PDO name for automatic channels", () => {
    expect(EtherCAT.Task.getPDOName(createAutoInputChannel("dev", "Status"))).toBe(
      "Status",
    );
  });

  it("should build an escaped hex address name for manual channels", () => {
    expect(EtherCAT.Task.getPDOName(createManualInputChannel("dev", 0x6000, 2))).toBe(
      "_0x6000_2",
    );
  });

  it("should escape characters that are invalid in channel names", () => {
    const ch = createAutoInputChannel("dev", "Status Word 1");
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
