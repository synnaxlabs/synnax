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

import { filter } from "@/filter";
import { zod } from "@/zod";

describe("filter input", () => {
  describe("setInputZ", () => {
    const keyZ = filter.setInputZ(z.number());

    it("should read a value as eq", () => {
      expect(keyZ.parse(12)).toEqual({ eq: 12 });
    });

    it("should read a list as in", () => {
      expect(keyZ.parse([1, 2])).toEqual({ in: [1, 2] });
    });

    it("should keep an operator", () => {
      expect(keyZ.parse({ in: [1], eq: 1 })).toEqual({ in: [1], eq: 1 });
    });

    it("should report the type of the line for a value of the wrong type", () => {
      expect(() => zod.parse(keyZ, "12")).toThrow('× expected number, received "12"');
    });

    it("should reject an unknown operator", () => {
      expect(() => keyZ.parse({ gt: 1 })).toThrow();
    });
  });

  describe("orderedInputZ", () => {
    const createdZ = filter.orderedInputZ(z.number());

    it("should read a value as eq", () => {
      expect(createdZ.parse(3)).toEqual({ eq: 3 });
    });

    it("should read a list as in", () => {
      expect(createdZ.parse([3, 4])).toEqual({ in: [3, 4] });
    });

    it("should keep an operator", () => {
      expect(createdZ.parse({ gt: 1, lte: 9 })).toEqual({ gt: 1, lte: 9 });
    });
  });

  describe("stringInputZ", () => {
    it("should read a value as eq", () => {
      expect(filter.stringInputZ.parse("a")).toEqual({ eq: "a" });
    });

    it("should read a list as in", () => {
      expect(filter.stringInputZ.parse(["a", "b"])).toEqual({ in: ["a", "b"] });
    });

    it("should keep an operator", () => {
      expect(
        filter.stringInputZ.parse({ startsWith: "Hot", caseInsensitive: true }),
      ).toEqual({ startsWith: "Hot", caseInsensitive: true });
    });
  });

  describe("boolInputZ", () => {
    it("should read a value as eq", () => {
      expect(filter.boolInputZ.parse(false)).toEqual({ eq: false });
    });

    it("should keep an operator", () => {
      expect(filter.boolInputZ.parse({ eq: true })).toEqual({ eq: true });
    });

    it("should reject a list", () => {
      expect(() => filter.boolInputZ.parse([true])).toThrow();
    });

    it("should report a value that is not a boolean", () => {
      expect(() => zod.parse(filter.boolInputZ, "yes")).toThrow(
        '× expected boolean, received "yes"',
      );
    });
  });

  describe("containsInputZ", () => {
    const integrationsZ = filter.containsInputZ(z.string());

    it("should read a value as any of one", () => {
      expect(integrationsZ.parse("ni")).toEqual({ any: ["ni"] });
    });

    it("should read a list as any", () => {
      expect(integrationsZ.parse(["ni", "labjack"])).toEqual({
        any: ["ni", "labjack"],
      });
    });

    it("should keep an operator", () => {
      expect(integrationsZ.parse({ any: ["ni"] })).toEqual({ any: ["ni"] });
    });

    it("should report a value of the wrong type", () => {
      expect(() => zod.parse(integrationsZ, 5)).toThrow(
        "× expected string, received 5",
      );
    });
  });

  describe("listInputZ", () => {
    const labelsZ = filter.listInputZ(z.string());

    it("should read a value as a list of one", () => {
      expect(labelsZ.parse("a")).toEqual(["a"]);
    });

    it("should keep a list", () => {
      expect(labelsZ.parse(["a", "b"])).toEqual(["a", "b"]);
    });
  });

  describe("requestInputZ", () => {
    const requestZ = z.strictObject({
      where: z
        .strictObject({ name: filter.stringZ.optional(), on: filter.boolZ.optional() })
        .optional(),
      limit: z.number().optional(),
    });
    const fieldsZ = z
      .strictObject({
        name: filter.stringInputZ.optional(),
        on: filter.boolInputZ.optional(),
        limit: z.number().optional(),
      })
      .transform((input) => filter.split(input, ["name", "on"]));
    const inputZ = filter.requestInputZ(requestZ, fieldsZ);

    it("should read filter fields beside the options", () => {
      expect(inputZ.parse({ name: "a", limit: 2 })).toEqual({
        where: { name: { eq: "a" } },
        limit: 2,
      });
    });

    it("should read a where tree", () => {
      expect(inputZ.parse({ where: { name: { eq: "a" } } })).toEqual({
        where: { name: { eq: "a" } },
      });
    });

    it("should name the operator at fault in a where tree", () => {
      expect(() => zod.parse(inputZ, { where: { name: { eq: 5 } } })).toThrow(
        /at where\.name\n.*\n\s*✗ "eq": 5 {2}× expected string/,
      );
    });

    it("should reject a filter field beside a where tree", () => {
      expect(() => zod.parse(inputZ, { where: {}, name: "a" })).toThrow(
        '✗ "name": "a"  × unexpected key',
      );
    });

    it("should name the filter field at fault", () => {
      expect(() => zod.parse(inputZ, { on: "yes" })).toThrow(
        '✗ "on": "yes"  × expected boolean',
      );
    });

    it("should report a value that is not an object", () => {
      expect(() => zod.parse(inputZ, "x")).toThrow('× expected object, received "x"');
    });
  });

  describe("split", () => {
    it("should move the named fields into where", () => {
      expect(filter.split({ key: { eq: 1 }, limit: 2 }, ["key"])).toEqual({
        where: { key: { eq: 1 } },
        limit: 2,
      });
    });

    it("should leave where undefined when no named field is set", () => {
      expect(filter.split({ key: undefined, limit: 2 }, ["key"]).where).toBeUndefined();
    });
  });
});
