// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";
import { join } from "node:path";

import { icd } from "@synnaxlabs/client";
import { caseconv, json } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { ICD } from "@/platform/icd";

const TABLE_PATH = join(
  import.meta.dirname,
  "../../../../core/pkg/service/icd/testdata/routes.json",
);

const tableZ = z.object({
  groups: z
    .object({
      description: z.string(),
      messages: z.object({ name: z.string(), payload: icd.jsonPayloadZ }).array(),
      cases: z
        .object({
          document: z.unknown(),
          fields: z.string().array(),
          message: z.string().optional(),
          matches: z.string().array().optional(),
        })
        .array(),
    })
    .array(),
});

const table = tableZ.parse(
  caseconv.snakeToCamel(JSON.parse(readFileSync(TABLE_PATH, "utf8"))),
);

interface Message {
  match?: icd.JSONMatch;
  pointers?: string[];
}

/** @returns the value of doc at pointer, or undefined when it holds none. */
const valueAt = (doc: unknown, pointer: string): unknown => {
  let value = doc;
  for (const token of json.parsePointer(pointer)) {
    if (value == null || typeof value !== "object") return undefined;
    value = (value as Record<string, unknown>)[token];
  }
  return value;
};

/** @returns a field of a shape whose last document holds value at pointer. */
const sampled = (pointer: string, value: unknown, enumLabels: string[] = []) => ({
  pointer,
  sample: JSON.stringify(value),
  enumLabels,
});

describe("route", () => {
  const kind = (value: string, ...pointers: string[]): Message => ({
    match: { type: "string", pointer: "/kind", value },
    pointers,
  });
  const rpm: Message = {
    match: { type: "present", pointer: "/rpm" },
    pointers: ["/rpm"],
  };
  const route = (fields: ReturnType<typeof sampled>[], messages: Message[]) =>
    ICD.route(fields, messages, (m) => m.pointers ?? []);

  describe("with a sample cut short", () => {
    // A sample holds at most 64 bytes, so a longer string is no JSON.
    const cut = (...pointers: string[]) =>
      pointers.map((pointer) => ({
        pointer,
        sample: `"${"a".repeat(63)}`,
        enumLabels: [],
      }));

    it("should give the message of a value rule that reads the most of the fields", () => {
      const status = kind("status", "/kind", "/code");
      const reading = kind("reading", "/kind", "/temp", "/rpm");
      expect(route(cut("/kind", "/temp", "/rpm"), [status, reading])).toEqual({
        message: reading,
      });
    });

    it("should give the message with no rule when it reads the most of the fields", () => {
      const rest = { pointers: ["/kind", "/temp"] };
      expect(route(cut("/kind", "/temp"), [kind("status", "/kind"), rest])).toEqual({
        message: rest,
      });
    });

    it("should give the message of a met present rule", () => {
      const status = kind("status", "/kind", "/temp");
      expect(route(cut("/kind", "/temp", "/rpm"), [status, rpm])).toEqual({
        message: rpm,
      });
    });

    it("should split when two messages read the most of the fields", () => {
      const messages = [
        kind("status", "/kind", "/code"),
        kind("fault", "/kind", "/rpm"),
      ];
      expect(route(cut("/kind", "/temp"), messages)).toEqual({ split: "/kind" });
    });
  });

  it("should give the message of the value rule that the sample meets over one that reads more of the fields", () => {
    const status = kind("status", "/kind", "/code");
    const reading = kind("reading", "/kind", "/temp", "/rpm");
    const fields = [
      sampled("/kind", "status"),
      sampled("/temp", 1),
      sampled("/rpm", 2),
    ];
    expect(route(fields, [status, reading])).toEqual({ message: status });
  });

  it("should give the message with no rule when the sample meets no value rule", () => {
    const rest = { pointers: ["/kind"] };
    const fields = [sampled("/kind", "fault"), sampled("/temp", 1)];
    expect(route(fields, [kind("status", "/kind", "/temp"), rest])).toEqual({
      message: rest,
    });
  });

  it("should give the matches of a met present rule and the value rule that the sample meets", () => {
    const status = kind("status", "/kind");
    const fields = [sampled("/kind", "status"), sampled("/rpm", 3000)];
    expect(route(fields, [status, rpm])).toEqual({ matches: [status, rpm] });
  });

  it("should give the message of a met present rule when the sample meets no value rule", () => {
    const fields = [sampled("/kind", "reading"), sampled("/rpm", 3000)];
    expect(route(fields, [kind("status", "/kind"), rpm])).toEqual({ message: rpm });
  });

  it("should split when the enum labels meet the rules of two messages", () => {
    const fields = [sampled("/kind", "fault", ["fault", "status"])];
    const messages = [kind("status", "/kind"), kind("fault", "/kind")];
    expect(route(fields, messages)).toEqual({ split: "/kind" });
  });

  it("should give the message with no rule when no enum label meets a value rule", () => {
    const rest = { pointers: ["/kind"] };
    const fields = [sampled("/kind", "open", ["closed", "open"])];
    expect(route(fields, [kind("status", "/kind"), rest])).toEqual({ message: rest });
  });

  it("should give the message of an integer rule that a sampled number meets", () => {
    const three: Message = { match: { type: "integer", pointer: "/code", value: "3" } };
    const rest = { pointers: ["/code"] };
    expect(route([sampled("/code", 3)], [three, rest])).toEqual({ message: three });
  });

  it("should give the message with no rule when the shape holds an object at the pointer of a value rule", () => {
    const rest = { pointers: ["/kind/code"] };
    expect(route([sampled("/kind/code", 1)], [kind("status", "/kind"), rest])).toEqual({
      message: rest,
    });
  });
});

describe.each(table.groups)("route: $description", ({ messages, cases }) => {
  const routed = messages.map(({ name, payload }) => ({
    name,
    match: payload.match,
    pointers: payload.fields.map(({ pointer }) => pointer),
  }));
  const named = (name: string) => routed.find((m) => m.name === name);
  it.each(cases)(
    "should give a shape of the document $document the message of the Core",
    ({ document, fields, message, matches }) => {
      let expected = {};
      if (matches != null) expected = { matches: matches.map(named) };
      else if (message != null) expected = { message: named(message) };
      expect(
        ICD.route(
          fields.map((pointer) => sampled(pointer, valueAt(document, pointer))),
          routed,
          ({ pointers }) => pointers,
        ),
      ).toEqual(expected);
    },
  );
});
