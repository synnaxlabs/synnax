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
import { caseconv } from "@synnaxlabs/x";
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

describe("route", () => {
  const kind = (value: string, ...pointers: string[]): Message => ({
    match: { type: "string", pointer: "/kind", value },
    pointers,
  });
  const fields = (...pointers: string[]) => pointers.map((pointer) => ({ pointer }));
  const route = (pointers: string[], messages: Message[]) =>
    ICD.route(fields(...pointers), messages, (m) => m.pointers ?? []);

  it("should give the message of a value rule that reads the most of the fields", () => {
    const status = kind("status", "/kind", "/code");
    const reading = kind("reading", "/kind", "/temp", "/rpm");
    expect(route(["/kind", "/temp", "/rpm"], [status, reading])).toEqual({
      message: reading,
    });
  });

  it("should give the message with no rule when it reads the most of the fields", () => {
    const rest = { pointers: ["/kind", "/temp"] };
    expect(route(["/kind", "/temp"], [kind("status", "/kind"), rest])).toEqual({
      message: rest,
    });
  });

  it("should split when two messages read the most of the fields", () => {
    const messages = [kind("status", "/kind", "/code"), kind("fault", "/kind", "/rpm")];
    expect(route(["/kind", "/temp"], messages)).toEqual({ split: "/kind" });
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
    "should give a shape with fields $fields the message of the Core",
    ({ fields, message, matches }) => {
      let expected = {};
      if (matches != null) expected = { matches: matches.map(named) };
      else if (message != null) expected = { message: named(message) };
      expect(
        ICD.route(
          fields.map((pointer) => ({ pointer })),
          routed,
          ({ pointers }) => pointers,
        ),
      ).toEqual(expected);
    },
  );
});
