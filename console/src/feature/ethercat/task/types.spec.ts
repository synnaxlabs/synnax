// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { EtherCAT } from "@/feature/ethercat";
import {
  createEntryReadChannel,
  createEntryWriteChannel,
  createRepeatedNamePDOs,
  createUnresolvedReadChannel,
  createUnresolvedWriteChannel,
  STATUS_ENTRY,
} from "@/feature/ethercat/testutil";

const roundTrip = (key: string): string =>
  caseconv.snakeToCamel(caseconv.camelToSnake(key));

describe("addressKey", () => {
  it("should key an entry by its decimal index and subindex", () => {
    expect(EtherCAT.Task.addressKey({ index: 0x6010, subIndex: 1 })).toBe("24592_1");
  });

  it("should key the write state of an entry with a state prefix", () => {
    expect(EtherCAT.Task.stateKey({ index: 0x6010, subIndex: 1 })).toBe(
      "state_24592_1",
    );
  });

  it("should survive the case conversion of the wire unchanged", () => {
    const address = { index: 0x6010, subIndex: 1 };
    const key = EtherCAT.Task.addressKey(address);
    const state = EtherCAT.Task.stateKey(address);
    expect(roundTrip(key)).toBe(key);
    expect(roundTrip(state)).toBe(state);
  });
});

describe("uniqueEntries", () => {
  it("should list an address that two PDOs map only once", () => {
    const value = { ...STATUS_ENTRY, name: "Value", pdoName: "Channel 1" };
    const entries = EtherCAT.Task.uniqueEntries([
      value,
      { ...value, pdoName: "Other" },
    ]);
    expect(entries).toEqual([value]);
  });

  it("should keep entries that share a name but not an address", () => {
    const { inputs } = createRepeatedNamePDOs();
    expect(EtherCAT.Task.uniqueEntries(inputs)).toHaveLength(8);
  });
});

describe("getEntryLabel", () => {
  it("should prefix the entry name with its PDO name", () => {
    const { inputs } = createRepeatedNamePDOs();
    expect(EtherCAT.Task.getEntryLabel(inputs[1])).toBe("Channel 2 › Input");
  });

  it("should use the entry name alone when the PDO name is unknown", () => {
    expect(EtherCAT.Task.getEntryLabel(STATUS_ENTRY)).toBe("Status");
  });

  it("should use the entry name alone when the PDO name is empty", () => {
    expect(EtherCAT.Task.getEntryLabel({ ...STATUS_ENTRY, pdoName: "" })).toBe(
      "Status",
    );
  });
});

describe("entryAt", () => {
  it("should return the entry at the address", () => {
    const { inputs } = createRepeatedNamePDOs();
    expect(EtherCAT.Task.entryAt(inputs, { index: 0x6020, subIndex: 1 })).toBe(
      inputs[2],
    );
  });

  it("should return undefined when the slave has no entry at the address", () => {
    const { inputs } = createRepeatedNamePDOs();
    expect(
      EtherCAT.Task.entryAt(inputs, { index: 0x6020, subIndex: 2 }),
    ).toBeUndefined();
  });
});

describe("getUnresolvedMessage", () => {
  const slave = {
    name: "EL1008",
    properties: {
      ...EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
      pdos: createRepeatedNamePDOs(),
    },
  };

  it("should say when no entry on the slave carries the name", () => {
    expect(EtherCAT.Task.getUnresolvedMessage("Velocity", slave, "inputs")).toBe(
      "'Velocity' matches no input entry on slave EL1008. Pick an entry.",
    );
  });

  it("should name the one entry that carries the name", () => {
    const value = { ...STATUS_ENTRY, name: "Value", pdoName: "Channel 1" };
    const single = {
      ...slave,
      properties: {
        ...slave.properties,
        pdos: { inputs: [value, { ...value, pdoName: "Compact" }], outputs: [] },
      },
    };
    expect(EtherCAT.Task.getUnresolvedMessage("Value", single, "inputs")).toBe(
      "'Value' matches input entry 0x6000:01 (Channel 1) on slave EL1008. " +
        "Pick an entry.",
    );
  });

  it("should list every address that carries the name", () => {
    const outputs = createRepeatedNamePDOs().outputs.slice(0, 2);
    const two = {
      ...slave,
      properties: { ...slave.properties, pdos: { inputs: [], outputs } },
    };
    expect(EtherCAT.Task.getUnresolvedMessage("Output", two, "outputs")).toBe(
      "'Output' matches more than one output entry on slave EL1008: " +
        "0x7000:01 (Channel 1), 0x7010:01 (Channel 2). Pick an entry.",
    );
  });
});

describe("getEntryChannelName", () => {
  it("should use the entry name when it is unique on the slave", () => {
    expect(EtherCAT.Task.getEntryChannelName("dev", [STATUS_ENTRY], STATUS_ENTRY)).toBe(
      "dev_Status",
    );
  });

  it("should add the PDO name when the entry name repeats", () => {
    const { inputs } = createRepeatedNamePDOs();
    expect(EtherCAT.Task.getEntryChannelName("dev", inputs, inputs[2])).toBe(
      "dev_Channel_3_Input",
    );
  });

  it("should give each of eight repeated entries a different name", () => {
    const { inputs } = createRepeatedNamePDOs();
    const names = inputs.map((e) =>
      EtherCAT.Task.getEntryChannelName("dev", inputs, e),
    );
    expect(new Set(names).size).toBe(8);
  });

  it("should add the address when the entry name repeats and no PDO name is known", () => {
    const inputs = createRepeatedNamePDOs().inputs.map(({ pdoName: _, ...e }) => e);
    expect(EtherCAT.Task.getEntryChannelName("dev", inputs, inputs[1])).toBe(
      "dev_Input_6010_01",
    );
  });

  it("should throw when the slave has no entry at the address", () => {
    expect(() =>
      EtherCAT.Task.getEntryChannelName("dev", [STATUS_ENTRY], {
        index: 0x6000,
        subIndex: 2,
      }),
    ).toThrow("The slave has no entry at 0x6000:02");
  });
});

describe("createReadChannel", () => {
  it("should start with no entry picked when the list is empty", () => {
    const ch = EtherCAT.Task.createReadChannel([]);
    expect(ch.type).toBe("entry");
    expect(EtherCAT.Task.hasEntry(ch)).toBe(false);
    expect(ch.key).not.toBe("");
  });

  it("should copy the slave of the last channel but reset its entry and binding", () => {
    const last = createEntryReadChannel("dev", STATUS_ENTRY, {
      channel: 42,
      name: "copied",
    });
    const ch = EtherCAT.Task.createReadChannel([last]);
    expect(ch).toMatchObject({
      type: "entry",
      device: "dev",
      channel: 0,
      name: "",
      index: 0,
      subIndex: 0,
      dataType: "",
    });
    expect(ch.key).not.toBe(last.key);
  });

  it("should copy the slave of an unresolved channel without its name", () => {
    const ch = EtherCAT.Task.createReadChannel([
      createUnresolvedReadChannel("dev", "Input"),
    ]);
    expect(ch.type).toBe("entry");
    expect(ch.device).toBe("dev");
    expect("pdo" in ch).toBe(false);
  });
});

describe("createWriteChannel", () => {
  it("should start with no entry picked when the list is empty", () => {
    const ch = EtherCAT.Task.createWriteChannel([]);
    expect(ch.type).toBe("entry");
    expect(EtherCAT.Task.hasEntry(ch)).toBe(false);
  });

  it("should copy the last channel but reset names, bindings, the entry, and the stop value", () => {
    const last = createEntryWriteChannel("dev", STATUS_ENTRY, {
      cmdChannel: 5,
      stateChannel: 6,
      cmdChannelName: "cmd",
      stateChannelName: "state",
      name: "copied",
      stopValue: 1,
    });
    const ch = EtherCAT.Task.createWriteChannel([last]);
    expect(ch).toMatchObject({
      device: "dev",
      cmdChannel: 0,
      stateChannel: 0,
      cmdChannelName: "",
      stateChannelName: "",
      name: "",
      index: 0,
      stopValue: 0,
    });
    expect(ch.key).not.toBe(last.key);
  });

  it("should copy the slave of an unresolved channel without its name", () => {
    const ch = EtherCAT.Task.createWriteChannel([
      createUnresolvedWriteChannel("dev", "Output", { cmdChannel: 5 }),
    ]);
    expect(ch).toMatchObject({ type: "entry", device: "dev", cmdChannel: 0, index: 0 });
    expect("pdo" in ch).toBe(false);
  });
});

describe("getPortLabel", () => {
  it("should label a channel with the address of its entry", () => {
    const ch = createEntryReadChannel("dev", { ...STATUS_ENTRY, index: 0x6010 });
    expect(EtherCAT.Task.getPortLabel(ch)).toBe("0x6010:01");
  });

  it("should label a channel with no entry as No entry", () => {
    expect(EtherCAT.Task.getPortLabel(EtherCAT.Task.createReadChannel([]))).toBe(
      "No entry",
    );
  });

  it("should label an unresolved channel with its entry name", () => {
    expect(
      EtherCAT.Task.getPortLabel(createUnresolvedReadChannel("dev", "Input")),
    ).toBe("Input");
  });
});

describe("EtherCAT Task statusData", () => {
  describe("readStatusDataZ", () => {
    it("should accept null", () => {
      expect(z.validate(EtherCAT.Task.READ_SCHEMAS.statusData, null)).toBe(true);
    });
    it("should accept undefined", () => {
      expect(z.validate(EtherCAT.Task.READ_SCHEMAS.statusData, undefined)).toBe(true);
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
      expect(z.validate(EtherCAT.Task.WRITE_SCHEMAS.statusData, null)).toBe(true);
    });
    it("should accept undefined", () => {
      expect(z.validate(EtherCAT.Task.WRITE_SCHEMAS.statusData, undefined)).toBe(true);
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
      z.validate(
        EtherCAT.Task.READ_SCHEMAS.config,
        EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
  it("should accept the zero write config", () => {
    expect(
      z.validate(
        EtherCAT.Task.WRITE_SCHEMAS.config,
        EtherCAT.Task.WRITE_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
});

describe("legacy shapes", () => {
  it("should reject a legacy manual channel, which the Core migrates first", () => {
    const v0Config = {
      sampleRate: 1000,
      streamRate: 25,
      channels: [
        {
          key: "1",
          type: "manual",
          device: "dev",
          index: 0x6000,
          subindex: 5,
          bitLength: 16,
          dataType: "uint16",
        },
      ],
    };
    expect(EtherCAT.Task.READ_SCHEMAS.config.safeParse(v0Config).success).toBe(false);
  });
});

describe("deploy configs", () => {
  it.each([
    ["read", EtherCAT.Task.READ_SCHEMAS.config, EtherCAT.Task.deployReadConfigZ],
    ["write", EtherCAT.Task.WRITE_SCHEMAS.config, EtherCAT.Task.deployWriteConfigZ],
  ])("should require an enabled channel to deploy a %s task", (_, draftZ, deployZ) => {
    const issues = deployZ.safeParse(draftZ.parse({})).error?.issues;
    expect(issues).toContainEqual(
      expect.objectContaining({
        path: ["channels"],
        message: "At least one enabled channel is required",
      }),
    );
  });
});
