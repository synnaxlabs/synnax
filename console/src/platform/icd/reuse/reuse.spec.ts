// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import type { payload } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const typeOf = (kind: payload.Kind): ICD.Reuse.Field["type"] => {
  if (kind === "boolean") return "boolean";
  return ["float", "integer", "unsigned"].includes(kind) ? "number" : "string";
};

const sampled = (
  pointer: string,
  kind: payload.Kind,
  extra: Partial<ICD.Reuse.Field> = {},
): ICD.Reuse.Field => ({ pointer, kind, type: typeOf(kind), ...extra });

const shape = (fields: ICD.Reuse.Field[], present?: string): ICD.Reuse.Shape => ({
  fields,
  present,
});

const message = (
  fields: icd.JSONField[],
  match?: icd.JSONMatch,
  name = "reading",
): icd.Message =>
  icd.messageZ.parse({ name, payload: { format: "json", match, fields } });

const field = (
  pointer: string,
  type: string,
  extra: Record<string, unknown> = {},
): icd.JSONField =>
  icd.jsonFieldZ.parse({ name: pointer.slice(1), pointer, type, ...extra });

describe("Reuse", () => {
  describe("fit", () => {
    const temp = sampled("/temp", "float");

    it("should fit a message with no match rule to the only shape of a source", () => {
      const m = message([field("/temp", "float")]);
      expect(ICD.Reuse.fit(shape([temp]), m)).toEqual({ message: m, missing: [] });
    });

    it("should not fit a message with no match rule to one of many shapes", () => {
      const m = message([field("/temp", "float")]);
      expect(ICD.Reuse.fit(shape([temp], "/temp"), m)).toBeUndefined();
    });

    it("should fit a present rule at a field that every document holds", () => {
      const m = message([field("/temp", "float")], {
        type: "present",
        pointer: "/temp",
      });
      expect(ICD.Reuse.fit(shape([temp], "/temp"), m)?.message).toBe(m);
    });

    it("should fit a present rule at an object that holds a field", () => {
      const m = message([field("/motor/rpm", "float")], {
        type: "present",
        pointer: "/motor",
      });
      const rpm = sampled("/motor/rpm", "float");
      expect(ICD.Reuse.fit(shape([rpm], "/motor/rpm"), m)?.message).toBe(m);
    });

    it("should not fit a present rule at a field that some documents lack", () => {
      const m = message([field("/temp", "float")], {
        type: "present",
        pointer: "/fault",
      });
      const fault = sampled("/fault", "string", { optional: true });
      expect(ICD.Reuse.fit(shape([temp, fault], "/temp"), m)).toBeUndefined();
    });

    it("should fit a string rule to the one value of a field", () => {
      const rule: icd.JSONMatch = {
        type: "string",
        pointer: "/kind",
        value: "reading",
      };
      const m = message([field("/temp", "float")], rule);
      const kind = (value?: string) => sampled("/kind", "string", { value });
      expect(ICD.Reuse.fit(shape([kind('"reading"'), temp]), m)).toEqual({
        message: m,
        missing: ["/kind"],
      });
      expect(ICD.Reuse.fit(shape([kind('"status"'), temp]), m)).toBeUndefined();
      expect(ICD.Reuse.fit(shape([kind(), temp]), m)).toBeUndefined();
    });

    it("should not fit a field with no default that the documents lack", () => {
      const m = message([field("/temp", "float"), field("/rpm", "float")]);
      expect(ICD.Reuse.fit(shape([temp]), m)).toBeUndefined();
    });

    it("should fit a message with a field the documents lack and a default", () => {
      const m = message([
        field("/temp", "float"),
        field("/rpm", "float", { default: 0 }),
      ]);
      expect(ICD.Reuse.fit(shape([temp]), m)?.message).toBe(m);
    });

    it("should not fit a field with no default that some documents lack", () => {
      const m = message([field("/temp", "float")]);
      const optional = sampled("/temp", "float", { optional: true });
      expect(
        ICD.Reuse.fit(shape([optional, sampled("/t", "string")]), m),
      ).toBeUndefined();
    });

    it("should read integers and unsigned integers into a float field", () => {
      const m = message([field("/temp", "float")]);
      expect(ICD.Reuse.fit(shape([sampled("/temp", "integer")]), m)).toBeDefined();
      expect(ICD.Reuse.fit(shape([sampled("/temp", "unsigned")]), m)).toBeDefined();
    });

    it("should read unsigned integers into an integer field", () => {
      const m = message([field("/count", "integer")]);
      expect(ICD.Reuse.fit(shape([sampled("/count", "unsigned")]), m)).toBeDefined();
      expect(ICD.Reuse.fit(shape([sampled("/count", "float")]), m)).toBeUndefined();
    });

    it("should not read integers into an unsigned field", () => {
      const m = message([field("/count", "unsigned")]);
      expect(ICD.Reuse.fit(shape([sampled("/count", "integer")]), m)).toBeUndefined();
    });

    it("should read any string into a string field", () => {
      const m = message([field("/at", "string")]);
      const at = sampled("/at", "timestamp", { timeFormat: "iso8601" });
      expect(ICD.Reuse.fit(shape([at]), m)).toBeDefined();
    });

    it("should not read a timestamp of another format", () => {
      const m = message([field("/t", "timestamp", { timeFormat: "unix_ms" })]);
      const t = (timeFormat: ICD.Reuse.Field["timeFormat"]) =>
        shape([sampled("/t", "timestamp", { type: "number", timeFormat })]);
      expect(ICD.Reuse.fit(t("unix_ms"), m)).toBeDefined();
      expect(ICD.Reuse.fit(t("unix_ns"), m)).toBeUndefined();
    });

    it("should hold the values under a JSON field", () => {
      const m = message([field("/status", "json")]);
      const code = sampled("/status/code", "integer");
      expect(ICD.Reuse.fit(shape([code, temp]), m)).toEqual({
        message: m,
        missing: ["/temp"],
      });
    });

    it("should list the fields of the shape that the message lacks", () => {
      const m = message([field("/temp", "float")]);
      const s = shape([sampled("/t", "string"), temp, sampled("/rpm", "float")]);
      expect(ICD.Reuse.fit(s, m)?.missing).toEqual(["/t", "/rpm"]);
    });

    it("should not fit a message that holds no field of the shape", () => {
      const m = message([field("/rpm", "float", { default: 0 })]);
      expect(ICD.Reuse.fit(shape([temp]), m)).toBeUndefined();
    });

    it("should not fit a message that is not JSON", () => {
      const m = icd.messageZ.parse({ name: "line", payload: { format: "text" } });
      expect(ICD.Reuse.fit(shape([temp]), m)).toBeUndefined();
    });
  });

  describe("find", () => {
    const temp = sampled("/temp", "float");
    const rpm = sampled("/rpm", "float");
    const plant = shape([temp, rpm]);
    const source = (
      shapes: ICD.Reuse.Shape[],
      stored: icd.MessageKey[] = [],
    ): ICD.Reuse.Source => ({ shapes, stored: new Set(stored) });
    const create = (...messages: icd.Message[]) =>
      icd.icdZ.parse({ name: "ICD", messages });
    const one = message([field("/temp", "float")]);
    const both = message([field("/temp", "float"), field("/rpm", "float")]);

    it("should pick the ICD whose message holds the most fields", () => {
      const a = create(one);
      const b = create(both);
      expect(ICD.Reuse.find(source([plant]), [a, b])).toEqual({
        pick: { icd: b.key, fits: [{ message: both, missing: [] }] },
        candidates: [
          { icd: b.key, fits: [[{ message: both, missing: [] }]] },
          { icd: a.key, fits: [[{ message: one, missing: ["/rpm"] }]] },
        ],
      });
    });

    it("should pick a stored message over one that holds more fields", () => {
      const a = create(one);
      const b = create(both);
      const { pick } = ICD.Reuse.find(source([plant], [one.key]), [a, b]);
      expect(pick).toEqual({ icd: a.key, fits: [{ message: one, missing: ["/rpm"] }] });
    });

    it("should pick the home ICD when two ICDs fit equally", () => {
      const a = create(both);
      const b = create(message([field("/temp", "float"), field("/rpm", "float")]));
      expect(ICD.Reuse.find(source([plant]), [a, b], b.key).pick?.icd).toBe(b.key);
    });

    it("should pick nothing when two ICDs fit equally and neither is home", () => {
      const a = create(both);
      const b = create(message([field("/temp", "float"), field("/rpm", "float")]));
      const { pick, candidates } = ICD.Reuse.find(source([plant]), [a, b]);
      expect(pick).toBeUndefined();
      expect(candidates.map(({ icd }) => icd)).toEqual([a.key, b.key]);
    });

    it("should pick nothing when two messages of one ICD fit a shape equally", () => {
      const twin = message(
        [field("/temp", "float"), field("/rpm", "float")],
        undefined,
        "twin",
      );
      const a = create(both, twin);
      const result = ICD.Reuse.find(source([plant]), [a, create(one)], a.key);
      expect(result.pick).toBeUndefined();
      expect(result.candidates[0].fits).toEqual([
        [
          { message: both, missing: [] },
          { message: twin, missing: [] },
        ],
      ]);
    });

    it("should give the fit of each shape of a source by position", () => {
      const reading = message([field("/temp", "float")], {
        type: "present",
        pointer: "/temp",
      });
      const a = create(reading);
      const fault = shape([sampled("/fault", "string")], "/fault");
      const { pick } = ICD.Reuse.find(source([fault, shape([temp], "/temp")]), [a]);
      expect(pick).toEqual({
        icd: a.key,
        fits: [undefined, { message: reading, missing: [] }],
      });
    });

    it("should give no candidates when no message fits", () => {
      const a = create(message([field("/pressure", "float")]));
      expect(ICD.Reuse.find(source([plant]), [a])).toEqual({ candidates: [] });
    });
  });
});
