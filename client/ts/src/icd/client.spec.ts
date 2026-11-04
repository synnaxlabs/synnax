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

import { NotFoundError, PathError, StaleICDError, ValidationError } from "@/errors";
import { icd } from "@/icd";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createNew = (name: string = "ICD"): icd.New => ({
  name,
  messages: [
    {
      name: "Status",
      payload: {
        format: "binary",
        identifier: { type: "can", id: 0x101, extended: false, fd: false },
        length: 8,
        fields: [
          { name: "state", startBit: 0, bitLength: 8 },
          {
            name: "temperature",
            startBit: 8,
            bitLength: 16,
            signed: true,
            scale: 0.1,
            units: "degC",
          },
        ],
      },
    },
  ],
});

describe("icd", () => {
  describe("payload", () => {
    it("should reject a text identifier on a binary payload", () => {
      expect(
        icd.payloadZ.safeParse({
          format: "binary",
          identifier: { type: "token", prefix: "$" },
        }).success,
      ).toBe(false);
    });

    it("should reject a binary field in a text payload", () => {
      expect(
        icd.payloadZ.safeParse({
          format: "text",
          fields: [{ encoding: "binary", name: "rpm", startBit: 0, bitLength: 8 }],
        }).success,
      ).toBe(false);
    });
  });

  describe("create", () => {
    it("should create an ICD with a CAN message", async () => {
      const ic = await client.icds.create(createNew());
      expect(ic.key).not.toEqual("");
      expect(ic.name).toEqual("ICD");
      expect(ic.messages).toHaveLength(1);
      const [msg] = ic.messages;
      expect(msg).toMatchObject({
        name: "Status",
        payload: {
          format: "binary",
          identifier: { type: "can", id: 0x101, extended: false, fd: false },
          length: 8,
        },
      });
      const { fields } = msg.payload;
      expect(fields).toHaveLength(2);
      expect(fields[1]).toMatchObject({
        name: "temperature",
        startBit: 8,
        bitLength: 16,
        signed: true,
        scale: 0.1,
        units: "degC",
      });
      const keys = [msg.key, ...fields.map((f) => f.key)];
      expect(new Set(keys).size).toEqual(keys.length);
    });

    it("should create a text message with the default delimiter", async () => {
      const ic = await client.icds.create({
        name: "Text",
        messages: [
          {
            name: "Line",
            payload: {
              format: "text",
              prefix: "$T",
              fields: [
                { encoding: "delimited", name: "temp", position: 1 },
                { encoding: "tagged", name: "rh", tag: "RH=" },
              ],
            },
          },
        ],
      });
      const [msg] = ic.messages;
      expect(msg).toMatchObject({
        payload: {
          format: "text",
          delimiter: ",",
          prefix: "$T",
          fields: [
            { encoding: "delimited", name: "temp", position: 1, scale: 1 },
            { encoding: "tagged", name: "rh", tag: "RH=" },
          ],
        },
      });
    });

    it("should create a JSON message with each field type", async () => {
      const ic = await client.icds.create({
        name: "JSON",
        messages: [
          {
            name: "Reading",
            payload: {
              format: "json",
              fields: [
                { type: "number", name: "temp", pointer: "/temp", scale: 0.1 },
                { type: "string", name: "state", pointer: "/state" },
                { type: "boolean", name: "armed", pointer: "/armed" },
                { type: "timestamp", name: "time", pointer: "/ts", format: "unix_ms" },
              ],
            },
          },
        ],
      });
      expect(await client.icds.retrieve(ic.key)).toEqual(ic);
      expect(ic.messages[0].payload).toMatchObject({
        format: "json",
        fields: [
          { type: "number", pointer: "/temp", scale: 0.1 },
          { type: "string", pointer: "/state", scale: 1 },
          { type: "boolean", pointer: "/armed" },
          { type: "timestamp", pointer: "/ts", format: "unix_ms" },
        ],
      });
    });

    it("should create multiple ICDs", async () => {
      const ics = await client.icds.create([createNew("A"), createNew("B")]);
      expect(ics.map((i) => i.name)).toEqual(["A", "B"]);
      expect(ics[0].key).not.toEqual(ics[1].key);
    });

    it("should replace an ICD with an existing key", async () => {
      const ic = await client.icds.create(createNew("Engine"));
      const [status] = ic.messages;
      await client.icds.create({ ...ic, messages: [{ ...status, name: "Renamed" }] });
      const retrieved = await client.icds.retrieve(ic.key);
      expect(retrieved.messages).toHaveLength(1);
      expect(retrieved.messages[0]).toMatchObject({ key: status.key, name: "Renamed" });
    });

    it("should reject a replacement that carries a stale hash", async () => {
      const ic = await client.icds.create(createNew("Engine"));
      await client.icds.create({ ...ic, messages: [] });
      await expect(client.icds.create({ ...ic, messages: [] })).rejects.toEqual(
        new StaleICDError("ICD Engine: changed since it was read"),
      );
    });

    it("should accept a replacement read before a rename", async () => {
      const ic = await client.icds.create(createNew());
      await client.icds.rename(ic.key, "Renamed");
      const saved = await client.icds.create({ ...ic, name: "Renamed", messages: [] });
      expect(saved.messages).toEqual([]);
    });

    it("should reject duplicate message names", async () => {
      const ic = createNew();
      ic.messages = [
        { name: "Mode", payload: { format: "binary" } },
        { name: "Mode", payload: { format: "binary" } },
      ];
      await expect(client.icds.create(ic)).rejects.toThrow(
        new PathError(
          ["messages", "1", "name"],
          new ValidationError('duplicate message name "Mode": validation error'),
        ),
      );
    });
  });

  describe("check", () => {
    it("should return no problems for a valid ICD", async () => {
      expect(await client.icds.check(createNew())).toEqual([]);
    });

    it("should return every problem with its path", async () => {
      const problems = await client.icds.check({
        name: "ICD",
        messages: [
          { name: "Mode", payload: { format: "binary" } },
          { name: "Mode", payload: { format: "binary" } },
          {
            name: "Status",
            payload: {
              format: "binary",
              length: 8,
              fields: [{ name: "rpm", startBit: 0, bitLength: 8, scale: 0 }],
            },
          },
        ],
      });
      expect(problems).toEqual([
        { path: "messages.1.name", message: 'duplicate message name "Mode"' },
        {
          path: "messages.2.payload.fields.0.scale",
          message: "scale must be finite and non-zero",
        },
      ]);
    });

    it("should return the problems with the pointers of a JSON message", async () => {
      const problems = await client.icds.check({
        name: "ICD",
        messages: [
          {
            name: "Reading",
            payload: {
              format: "json",
              fields: [
                { type: "number", name: "a", pointer: "/v" },
                { type: "string", name: "b", pointer: "/v/x", scale: 2 },
              ],
            },
          },
        ],
      });
      expect(problems).toEqual([
        {
          path: "messages.0.payload.fields.1.pointer",
          message: 'pointer "/v/x" is inside the value of field "a"',
        },
        {
          path: "messages.0.payload.fields.1.scale",
          message: "only a number field can scale",
        },
      ]);
    });

    it("should return the problems with a rule of the schema", async () => {
      const problems = await client.icds.check({
        name: "ICD",
        messages: [{ name: "", payload: { format: "binary" } }],
      });
      expect(problems).toEqual([
        {
          path: "messages.0.name",
          message: "Too small: expected string to have >=1 characters",
        },
      ]);
    });

    it("should not save the ICD", async () => {
      const ic = icd.icdZ.parse(createNew());
      await client.icds.check(ic);
      await expect(client.icds.retrieve(ic.key)).rejects.toThrow(NotFoundError);
    });
  });

  describe("retrieve", () => {
    it("should retrieve an ICD by key", async () => {
      const ic = await client.icds.create(createNew());
      const retrieved = await client.icds.retrieve(ic.key);
      expect(retrieved).toEqual(ic);
    });

    it("should retrieve multiple ICDs by key", async () => {
      const ics = await client.icds.create([createNew("A"), createNew("B")]);
      const retrieved = await client.icds.retrieve({
        keys: ics.map((i) => i.key),
      });
      expect(retrieved.map((i) => i.name).sort()).toEqual(["A", "B"]);
    });

    it("should search for ICDs by name", async () => {
      const name = `${id.create()} ICD`;
      const ic = await client.icds.create(createNew(name));
      await expect
        .poll(async () => {
          const res = await client.icds.retrieve({ searchTerm: name });
          return res.map((i) => i.key);
        })
        .toContain(ic.key);
    });
  });

  describe("rename", () => {
    it("should rename an ICD", async () => {
      const ic = await client.icds.create(createNew());
      await client.icds.rename(ic.key, "Renamed");
      const retrieved = await client.icds.retrieve(ic.key);
      expect(retrieved.name).toEqual("Renamed");
      expect(retrieved.messages).toEqual(ic.messages);
    });
  });

  describe("import", () => {
    const encode = (text: string): Uint8Array => new TextEncoder().encode(text);

    it("should import messages from a CSV field table", async () => {
      const ic = await client.icds.create({ name: "Import" });
      const csv = [
        "message,id,length,field,start_bit,bit_length,scale,units",
        "Engine,256,8,Rpm,0,16,0.25,rpm",
        "Engine,256,8,Temp,16,8,1,degC",
      ].join("\n");
      const imported = await client.icds.import(ic.key, encode(csv), "csv");
      expect(imported.key).toEqual(ic.key);
      expect(imported.messages).toHaveLength(1);
      const [msg] = imported.messages;
      expect(msg.name).toEqual("Engine");
      expect(msg.payload).toMatchObject({
        format: "binary",
        fields: [
          { name: "Rpm", startBit: 0, bitLength: 16, scale: 0.25 },
          { name: "Temp" },
        ],
      });
      expect(await client.icds.retrieve(ic.key)).toEqual(imported);
    });

    it("should import messages from a DBC file with value tables", async () => {
      const ic = await client.icds.create({ name: "Import" });
      const dbc = [
        'VERSION ""',
        "BS_:",
        "BU_: Ecu",
        'VAL_TABLE_ GearTable 0 "Park" 1 "Drive" ;',
        "BO_ 256 Engine: 8 Ecu",
        ' SG_ Rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Ecu',
        "",
      ].join("\n");
      const imported = await client.icds.import(ic.key, encode(dbc), "dbc");
      expect(imported.messages.map((m) => m.name)).toEqual(["Engine"]);
    });

    it("should keep message keys across a second import", async () => {
      const ic = await client.icds.create({ name: "Import" });
      const csv = encode("message,field,start_bit,bit_length\nEngine,Rpm,0,16");
      const first = await client.icds.import(ic.key, csv, "csv");
      const second = await client.icds.import(ic.key, csv, "csv");
      expect(second.messages[0].key).toEqual(first.messages[0].key);
    });

    it("should reject a document that does not parse", async () => {
      const ic = await client.icds.create({ name: "Import" });
      await expect(client.icds.import(ic.key, encode(""), "csv")).rejects.toThrow(
        PathError,
      );
    });
  });

  describe("delete", () => {
    it("should delete an ICD", async () => {
      const ic = await client.icds.create(createNew());
      await client.icds.delete(ic.key);
      await expect(client.icds.retrieve(ic.key)).rejects.toThrow(NotFoundError);
    });

    it("should delete multiple ICDs", async () => {
      const ics = await client.icds.create([createNew("A"), createNew("B")]);
      const keys = ics.map((i) => i.key);
      await client.icds.delete(keys);
      await expect(client.icds.retrieve({ keys })).rejects.toThrow(NotFoundError);
    });
  });
});
