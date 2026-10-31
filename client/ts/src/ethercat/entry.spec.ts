// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ethercat } from "@/ethercat";
import { MultipleFoundError, NotFoundError } from "@/index";

const EL1008_INPUTS: ethercat.PDOEntry[] = Array.from({ length: 8 }, (_, i) => ({
  name: "Input",
  pdoName: `Channel ${i + 1}`,
  pdoIndex: 0x1a00 + i,
  index: 0x6000 + 0x10 * i,
  subIndex: 1,
  bitLength: 1,
  dataType: "uint8",
}));

describe("pdoEntryZ", () => {
  it("should parse an entry that a scan wrote", () => {
    expect(ethercat.pdoEntryZ.parse(EL1008_INPUTS[0])).toEqual(EL1008_INPUTS[0]);
  });

  it("should parse an entry with no PDO name", () => {
    const { pdoName: _, ...entry } = EL1008_INPUTS[0];
    expect(ethercat.pdoEntryZ.parse(entry)).toEqual(entry);
  });
});

describe("formatAddress", () => {
  it("should format the index and subindex as padded uppercase hex", () => {
    expect(ethercat.formatAddress({ index: 0x1a0, subIndex: 0x1b })).toBe("0x01A0:1B");
  });
});

describe("formatEntry", () => {
  it("should follow the address with the PDO name", () => {
    expect(ethercat.formatEntry(EL1008_INPUTS[2])).toBe("0x6020:01 (Channel 3)");
  });

  it("should format the address alone when the PDO name is unknown", () => {
    const { pdoName: _, ...entry } = EL1008_INPUTS[2];
    expect(ethercat.formatEntry(entry)).toBe("0x6020:01");
  });
});

describe("findEntry", () => {
  it("should resolve a name that only one entry has", () => {
    const entries: ethercat.PDOEntry[] = [
      { name: "Output", index: 0x7000, subIndex: 1, bitLength: 8, dataType: "uint8" },
    ];
    expect(ethercat.findEntry(entries, "Output")).toEqual(entries[0]);
  });

  it("should pick the entry inside the named PDO group", () => {
    const entry = ethercat.findEntry(EL1008_INPUTS, "Input", "Channel 3");
    expect([entry.index, entry.subIndex]).toEqual([0x6020, 1]);
  });

  it("should throw and list every address when the name repeats", () => {
    expect(() => ethercat.findEntry(EL1008_INPUTS, "Input")).toThrow(
      MultipleFoundError,
    );
    expect(() => ethercat.findEntry(EL1008_INPUTS, "Input")).toThrow(
      /^entry "Input" matches 0x6000:01 \(Channel 1\).*0x6070:01 \(Channel 8\)\. Pass pdoName to pick one$/,
    );
  });

  it("should throw when no entry has the name", () => {
    expect(() => ethercat.findEntry(EL1008_INPUTS, "Velocity")).toThrow(NotFoundError);
    expect(() => ethercat.findEntry(EL1008_INPUTS, "Velocity")).toThrow(
      /^no entry "Velocity"$/,
    );
  });

  it("should name the PDO when no entry in it has the name", () => {
    expect(() => ethercat.findEntry(EL1008_INPUTS, "Input", "Channel 9")).toThrow(
      /^no entry "Input" in PDO "Channel 9"$/,
    );
  });

  it("should resolve once when PDO groups repeat the same object", () => {
    const entries: ethercat.PDOEntry[] = Array.from({ length: 4 }, (_, i) => ({
      name: "SubIndex 032",
      pdoName: `DCM TxPDO-Map ${i}`,
      index: 0x1c32,
      subIndex: 0x20,
      bitLength: 8,
      dataType: "uint8",
    }));
    expect(ethercat.findEntry(entries, "SubIndex 032")).toEqual(entries[0]);
  });
});
