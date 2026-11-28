// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { zod } from "@/zod";

const messages = (schema: z.ZodType, value: unknown): string[] => {
  const result = schema.safeParse(value, { error: zod.issueMessage });
  if (result.success) throw new Error("the value parsed");
  return result.error.issues.map((issue) => issue.message);
};

describe("issueMessage", () => {
  it("should call a string that is too short required", () => {
    expect(messages(z.object({ port: z.string().min(1) }), { port: "" })).toEqual([
      "port is required",
    ]);
  });

  it("should give the length of a string minimum above 1", () => {
    expect(
      messages(z.object({ password: z.string().min(8) }), { password: "abc" }),
    ).toEqual(["password must be at least 8 characters long"]);
  });

  it("should give the length of a string maximum", () => {
    expect(messages(z.object({ name: z.string().max(3) }), { name: "John" })).toEqual([
      "name must be at most 3 characters long",
    ]);
  });

  it("should call the field value when the path has no string key", () => {
    expect(messages(z.string().min(1), "")).toEqual(["value is required"]);
  });

  it("should word an inclusive lower bound", () => {
    expect(
      messages(z.object({ dataBits: z.number().min(5) }), { dataBits: 4 }),
    ).toEqual(["data bits must be greater than or equal to 5"]);
  });

  it("should word an exclusive lower bound", () => {
    expect(messages(z.object({ baudRate: z.number().gt(0) }), { baudRate: 0 })).toEqual(
      ["baud rate must be greater than 0"],
    );
  });

  it("should word an inclusive upper bound", () => {
    expect(
      messages(z.object({ dataBits: z.number().max(8) }), { dataBits: 9 }),
    ).toEqual(["data bits must be less than or equal to 8"]);
  });

  it("should word an exclusive upper bound", () => {
    expect(messages(z.object({ ratio: z.number().lt(1) }), { ratio: 1 })).toEqual([
      "ratio must be less than 1",
    ]);
  });

  it("should call any other issue not valid", () => {
    expect(
      messages(z.object({ parity: z.enum(["none", "even"]) }), { parity: "odd" }),
    ).toEqual(["parity is not valid"]);
  });

  it("should name the field by the last string key of the path", () => {
    const schema = z.object({
      fields: z.array(z.object({ bitLength: z.number().min(1) })),
    });
    expect(messages(schema, { fields: [{ bitLength: 1 }, { bitLength: 0 }] })).toEqual([
      "bit length must be greater than or equal to 1",
    ]);
  });

  it("should leave a message that the schema gives", () => {
    expect(
      messages(z.object({ uri: z.string().min(1, "URI is required") }), { uri: "" }),
    ).toEqual(["URI is required"]);
  });
});
