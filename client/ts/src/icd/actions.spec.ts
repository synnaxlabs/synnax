// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { icd } from "@/icd";

const MESSAGE_KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";
const FIELD_KEY = "6b8e4f2d-3a5c-4d7e-9fa0-1b2c3d4e5f60";

const createICD = (): icd.ICD =>
  icd.icdZ.parse({
    name: "Engine",
    messages: [
      {
        key: MESSAGE_KEY,
        name: "Status",
        payload: {
          format: "binary",
          fields: [{ key: FIELD_KEY, name: "rpm", startBit: 0, bitLength: 8 }],
        },
      },
    ],
  });

const FIELD_PATH = ["messages", MESSAGE_KEY, "payload", "fields", FIELD_KEY];

describe("icd.reduceAll", () => {
  it("should set a snake_case path in the camelCase ICD", () => {
    const { next } = icd.reduceAll(createICD(), [
      icd.set({ path: [...FIELD_PATH, "bit_length"], value: 16 }),
    ]);
    expect(next.messages[0].payload.fields[0]).toMatchObject({ bitLength: 16 });
  });

  it("should convert the keys of an inserted field to camelCase", () => {
    const { next } = icd.reduceAll(createICD(), [
      icd.insert({
        path: ["messages", MESSAGE_KEY, "payload", "fields"],
        value: { key: "b", name: "temp", start_bit: 8, bit_length: 8 },
        after: FIELD_KEY,
      }),
    ]);
    expect(next.messages[0].payload.fields[1]).toMatchObject({
      name: "temp",
      startBit: 8,
    });
  });

  it("should return the inverse that restores the ICD", () => {
    const ic = createICD();
    const { next, inverse } = icd.reduceAll(ic, [
      icd.remove({ path: ["messages"], key: MESSAGE_KEY }),
    ]);
    expect(next.messages).toEqual([]);
    expect(icd.reduceAll(next, inverse).next).toEqual(ic);
  });

  it("should replace the ICD on a create", () => {
    const created = { ...createICD(), name: "Created", messages: [] };
    const { next } = icd.reduceAll(createICD(), [icd.create({ icd: created })]);
    expect(next).toEqual(created);
  });
});

describe("icd.encode", () => {
  it("should convert a camelCase path to snake_case", () => {
    const action = icd.set({ path: [...FIELD_PATH, "bitLength"], value: 16 });
    expect(icd.encode(createICD(), action)).toEqual(
      icd.set({ path: [...FIELD_PATH, "bit_length"], value: 16 }),
    );
  });
});
