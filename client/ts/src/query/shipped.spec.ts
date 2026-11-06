// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { query } from "@/query";

const lineZ = z.union([z.string(), z.string().array()]);

const inputZ = z.strictObject({
  name: lineZ.optional(),
  type: lineZ.optional(),
  not: z.strictObject({ type: lineZ }).optional(),
  and: z.record(z.string(), z.unknown()).array().optional(),
});

const shippedZ = query.shippedZ(inputZ, {
  names: "name",
  notTypes: {
    hint: "Use `not: { type }`.",
    read: (types: string[]) => ({ not: { type: types } }),
  },
  ignoreNotFoundError: query.ignoreNotFoundError,
});

const spyOnWarn = () => vi.spyOn(console, "warn").mockImplementation(() => {});

describe("shippedZ", () => {
  it("should read a shipped field under the name that replaces it", () => {
    const warn = spyOnWarn();
    expect(shippedZ.parse({ names: ["a"] })).toEqual({ name: ["a"] });
    expect(warn).toHaveBeenCalledWith(
      "names is deprecated and will be removed in v0.62. Use name.",
    );
  });

  it("should read a shipped field through its read function", () => {
    const warn = spyOnWarn();
    expect(shippedZ.parse({ notTypes: ["b"] })).toEqual({ not: { type: ["b"] } });
    expect(warn).toHaveBeenCalledWith(
      "notTypes is deprecated and will be removed in v0.62. Use `not: { type }`.",
    );
  });

  it("should read an empty list as unset", () => {
    spyOnWarn();
    expect(shippedZ.parse({ names: [] })).toEqual({});
  });

  it("should read ignoreNotFoundError as nothing", () => {
    const warn = spyOnWarn();
    expect(shippedZ.parse({ ignoreNotFoundError: true })).toEqual({});
    expect(warn).toHaveBeenCalledWith(
      "ignoreNotFoundError is deprecated and will be removed in v0.62. A list " +
        "retrieve drops missing keys.",
    );
  });

  it("should AND a shipped field with the field that replaces it", () => {
    spyOnWarn();
    expect(shippedZ.parse({ names: ["a"], name: "b" })).toEqual({
      name: "b",
      and: [{ name: ["a"] }],
    });
  });

  it("should pass input without shipped fields through without a warning", () => {
    const warn = spyOnWarn();
    expect(shippedZ.parse({ name: ["a"] })).toEqual({ name: ["a"] });
    expect(warn).not.toHaveBeenCalled();
  });
});
