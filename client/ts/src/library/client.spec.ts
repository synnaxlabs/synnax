// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError, PathError, ValidationError } from "@/errors";
import { type library } from "@/library";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createNew = (name: string = "Library"): library.New => ({
  name,
  entries: [
    {
      kind: "enum",
      name: "State",
      values: [
        { value: 0, name: "Off" },
        { value: 1, name: "On" },
      ],
    },
    {
      kind: "message",
      name: "Status",
      identifier: { type: "can", id: 0x101, extended: false, fd: false },
      length: 8,
      fields: [
        { encoding: "binary", name: "state", startBit: 0, bitLength: 8 },
        {
          encoding: "binary",
          name: "temperature",
          startBit: 8,
          bitLength: 16,
          signed: true,
          scale: 0.1,
          units: "degC",
        },
      ],
    },
  ],
});

describe("library", () => {
  describe("create", () => {
    it("should create a library with an enum and a CAN message", async () => {
      const lib = await client.libraries.create(createNew());
      expect(lib.key).not.toEqual("");
      expect(lib.name).toEqual("Library");
      expect(lib.entries).toHaveLength(2);
      const [en, msg] = lib.entries;
      expect(en).toMatchObject({
        kind: "enum",
        name: "State",
        values: [
          { value: 0, name: "Off" },
          { value: 1, name: "On" },
        ],
      });
      expect(msg).toMatchObject({
        kind: "message",
        name: "Status",
        identifier: { type: "can", id: 0x101, extended: false, fd: false },
        format: "binary",
        length: 8,
      });
      if (msg.kind !== "message") throw new Error("expected a message entry");
      expect(msg.fields).toHaveLength(2);
      expect(msg.fields[1]).toMatchObject({
        encoding: "binary",
        name: "temperature",
        startBit: 8,
        bitLength: 16,
        signed: true,
        scale: 0.1,
        units: "degC",
      });
      const keys = [en.key, msg.key, ...msg.fields.map((f) => f.key)];
      expect(new Set(keys).size).toEqual(keys.length);
    });

    it("should create multiple libraries", async () => {
      const libs = await client.libraries.create([createNew("A"), createNew("B")]);
      expect(libs.map((l) => l.name)).toEqual(["A", "B"]);
      expect(libs[0].key).not.toEqual(libs[1].key);
    });

    it("should replace a library with an existing key", async () => {
      const lib = await client.libraries.create(createNew());
      await client.libraries.create({ ...lib, entries: [lib.entries[0]] });
      const retrieved = await client.libraries.retrieve(lib.key);
      expect(retrieved.entries).toHaveLength(1);
      expect(retrieved.entries[0].key).toEqual(lib.entries[0].key);
    });

    it("should reject duplicate entry names", async () => {
      const lib = createNew();
      lib.entries = [
        { kind: "enum", name: "Mode" },
        { kind: "enum", name: "Mode" },
      ];
      await expect(client.libraries.create(lib)).rejects.toThrow(
        new PathError(
          ["entries", "1", "name"],
          new ValidationError('duplicate entry name "Mode": validation error'),
        ),
      );
    });
  });

  describe("retrieve", () => {
    it("should retrieve a library by key", async () => {
      const lib = await client.libraries.create(createNew());
      const retrieved = await client.libraries.retrieve(lib.key);
      expect(retrieved).toEqual(lib);
    });

    it("should retrieve multiple libraries by key", async () => {
      const libs = await client.libraries.create([createNew("A"), createNew("B")]);
      const retrieved = await client.libraries.retrieve({
        keys: libs.map((l) => l.key),
      });
      expect(retrieved.map((l) => l.name).sort()).toEqual(["A", "B"]);
    });

    it("should search for libraries by name", async () => {
      const name = `${id.create()} Library`;
      const lib = await client.libraries.create(createNew(name));
      await expect
        .poll(async () => {
          const res = await client.libraries.retrieve({ searchTerm: name });
          return res.map((l) => l.key);
        })
        .toContain(lib.key);
    });
  });

  describe("rename", () => {
    it("should rename a library", async () => {
      const lib = await client.libraries.create(createNew());
      await client.libraries.rename(lib.key, "Renamed");
      const retrieved = await client.libraries.retrieve(lib.key);
      expect(retrieved.name).toEqual("Renamed");
      expect(retrieved.entries).toEqual(lib.entries);
    });
  });

  describe("import", () => {
    const encode = (text: string): Uint8Array => new TextEncoder().encode(text);

    it("should import messages from a CSV field table", async () => {
      const lib = await client.libraries.create({ name: "Import" });
      const csv = [
        "message,id,length,field,start_bit,bit_length,scale,units",
        "Engine,256,8,Rpm,0,16,0.25,rpm",
        "Engine,256,8,Temp,16,8,1,degC",
      ].join("\n");
      const imported = await client.libraries.import(lib.key, "csv", encode(csv));
      expect(imported.key).toEqual(lib.key);
      expect(imported.entries).toHaveLength(1);
      const [msg] = imported.entries;
      if (msg.kind !== "message") throw new Error("expected a message entry");
      expect(msg.name).toEqual("Engine");
      expect(msg.fields.map((f) => f.name)).toEqual(["Rpm", "Temp"]);
      expect(msg.fields[0]).toMatchObject({ startBit: 0, bitLength: 16, scale: 0.25 });
      expect(await client.libraries.retrieve(lib.key)).toEqual(imported);
    });

    it("should import messages and value tables from a DBC file", async () => {
      const lib = await client.libraries.create({ name: "Import" });
      const dbc = [
        'VERSION ""',
        "BS_:",
        "BU_: Ecu",
        'VAL_TABLE_ GearTable 0 "Park" 1 "Drive" ;',
        "BO_ 256 Engine: 8 Ecu",
        ' SG_ Rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Ecu',
        "",
      ].join("\n");
      const imported = await client.libraries.import(lib.key, "dbc", encode(dbc));
      const names = imported.entries.map((e) => `${e.kind}:${e.name}`).sort();
      expect(names).toEqual(["enum:GearTable", "message:Engine"]);
    });

    it("should keep entry keys across a second import", async () => {
      const lib = await client.libraries.create({ name: "Import" });
      const csv = encode("message,field,start_bit,bit_length\nEngine,Rpm,0,16");
      const first = await client.libraries.import(lib.key, "csv", csv);
      const second = await client.libraries.import(lib.key, "csv", csv);
      expect(second.entries[0].key).toEqual(first.entries[0].key);
    });

    it("should reject a document that does not parse", async () => {
      const lib = await client.libraries.create({ name: "Import" });
      await expect(client.libraries.import(lib.key, "csv", encode(""))).rejects.toThrow(
        PathError,
      );
    });
  });

  describe("delete", () => {
    it("should delete a library", async () => {
      const lib = await client.libraries.create(createNew());
      await client.libraries.delete(lib.key);
      await expect(client.libraries.retrieve(lib.key)).rejects.toThrow(NotFoundError);
    });

    it("should delete multiple libraries", async () => {
      const libs = await client.libraries.create([createNew("A"), createNew("B")]);
      const keys = libs.map((l) => l.key);
      await client.libraries.delete(keys);
      await expect(client.libraries.retrieve({ keys })).rejects.toThrow(NotFoundError);
    });
  });
});
