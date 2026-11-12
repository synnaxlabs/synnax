// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError, PathError } from "@/errors";
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

    it("should store an ICD that has problems", async () => {
      const ic = await client.icds.create({
        name: "ICD",
        messages: [
          { name: "Mode", payload: { format: "binary" } },
          { name: "Mode", payload: { format: "binary" } },
        ],
      });
      expect((await client.icds.retrieve(ic.key)).messages).toHaveLength(2);
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

    it("should return a problem from the Core with a camel case path", async () => {
      const problems = await client.icds.check({
        name: "ICD",
        messages: [
          {
            name: "Frame",
            payload: {
              format: "binary",
              identifier: {
                type: "can",
                id: 1,
                extended: false,
                fd: false,
                bitrateSwitched: true,
              },
            },
          },
        ],
      });
      expect(problems).toEqual([
        {
          path: "messages.0.payload.identifier.bitrateSwitched",
          message: "only a CAN FD frame can switch bitrates",
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
          message: "name is required",
        },
      ]);
    });

    it("should name the field of a broken range rule", async () => {
      const problems = await client.icds.check({
        name: "ICD",
        messages: [
          {
            name: "M",
            payload: { format: "binary", fields: [{ name: "a", bitLength: 65 }] },
          },
        ],
      });
      expect(problems).toEqual([
        {
          path: "messages.0.payload.fields.0.bitLength",
          message: "bit length must be less than or equal to 64",
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
      const retrieved = await client.icds.retrieve(ics.map((i) => i.key));
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

  describe("dispatch", () => {
    it("should set a value inside a message payload", async () => {
      const ic = await client.icds.create(createNew());
      const [msg] = ic.messages;
      const [field] = msg.payload.fields;
      const path = ["messages", msg.key, "payload", "fields", field.key, "bitLength"];
      expect(await client.icds.dispatch(ic.key, icd.set({ path, value: 12 }))).toBe(
        true,
      );
      const stored = await createTestClient().icds.retrieve(ic.key);
      expect(stored.messages[0].payload.fields[0]).toMatchObject({ bitLength: 12 });
    });

    it("should insert and move messages", async () => {
      const ic = await client.icds.create(createNew());
      const line: icd.Message = {
        key: uuid.create(),
        name: "Line",
        payload: { format: "text", delimiter: ",", prefix: "", fields: [] },
      };
      await client.icds.dispatch(ic.key, [
        icd.insert({ path: ["messages"], value: line, after: ic.messages[0].key }),
        icd.move({ path: ["messages"], key: line.key }),
      ]);
      const stored = await createTestClient().icds.retrieve(ic.key);
      expect(stored.messages.map(({ name }) => name)).toEqual(["Line", "Status"]);
    });

    it("should remove a message", async () => {
      const ic = await client.icds.create(createNew());
      await client.icds.dispatch(
        ic.key,
        icd.remove({ path: ["messages"], key: ic.messages[0].key }),
      );
      expect((await createTestClient().icds.retrieve(ic.key)).messages).toEqual([]);
    });

    it("should store a dispatch that leaves problems", async () => {
      const ic = await client.icds.create(createNew());
      const [msg] = ic.messages;
      const [state] = msg.payload.fields;
      const path = ["messages", msg.key, "payload", "fields", state.key, "name"];
      await client.icds.dispatch(ic.key, icd.set({ path, value: "temperature" }));
      const stored = await createTestClient().icds.retrieve(ic.key);
      expect(await client.icds.check(stored)).toEqual([
        {
          path: "messages.0.payload.fields.1.name",
          message: 'duplicate field name "temperature"',
        },
      ]);
    });

    it("should drop a dispatch the Core refuses from the cached ICD", async () => {
      const ic = await client.icds.create(createNew());
      await expect(
        client.icds.dispatch(ic.key, icd.set({ path: ["key"], value: uuid.create() })),
      ).rejects.toThrow('unknown object member name "key"');
      expect((await client.icds.retrieve(ic.key)).key).toEqual(ic.key);
    });

    it("should return false without a send when the ICD is not cached", async () => {
      const ic = await client.icds.create(createNew());
      expect(
        await createTestClient().icds.dispatch(
          ic.key,
          icd.set({ path: ["name"], value: "Renamed" }),
        ),
      ).toBe(false);
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
      await expect(client.icds.retrieve(keys)).rejects.toThrow(NotFoundError);
    });
  });

  describe("resources", () => {
    it("should find an ICD by its name in a resource search", async () => {
      const name = id.create();
      const ic = await client.icds.create(createNew(name));
      await expect
        .poll(async () =>
          (await client.resources.search({ searchTerm: name, types: ["icd"] })).map(
            (r) => r.id,
          ),
        )
        .toContainEqual(icd.resourceID(ic.key));
    });
  });
});
