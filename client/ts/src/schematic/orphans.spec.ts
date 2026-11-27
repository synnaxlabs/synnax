// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { schematic } from "@/schematic";

// The vectors pin this copy of the rule to the one that a symbol save runs in the Core.
const VECTORS_URL = new URL(
  "../../../../core/testdata/orphans/vectors.json",
  import.meta.url,
);

const specZ = z.object({
  states: schematic.symbol.stateZ.array().optional(),
  commands: schematic.symbol.commandZ.array().optional(),
});

const vectorsZ = z.object({
  before: specZ.required(),
  bindings: z.object({
    states: schematic.stateBindingZ.array(),
    commands: schematic.commandBindingZ.array(),
  }),
  cases: z
    .object({
      name: z.string(),
      after: specZ,
      orphans: z.object({
        states: z.string().array(),
        overrides: z
          .object({
            key: z.string(),
            fields: z.enum(["bounds", "enumeration", "match", "default"]).array(),
          })
          .array(),
        commands: z.string().array(),
      }),
    })
    .array(),
});

const { before, bindings, cases } = vectorsZ.parse(
  JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
);

describe("orphans", () => {
  it.each(cases)("should return $name", ({ after, orphans }) => {
    // A case that leaves out states or commands keeps those of before.
    expect(
      schematic.orphans(
        bindings,
        after.states ?? before.states,
        after.commands ?? before.commands,
      ),
    ).toEqual(orphans);
  });
});

const OVERRIDES = ["bounds", "enumeration", "match", "default"] as const;

// removed returns what dropOrphans took from the bindings to leave config.
const removed = (config: schematic.SymbolNodeConfig): schematic.Orphans => ({
  states: bindings.states
    .filter((b) => !config.states.some((k) => k.key === b.key))
    .map((b) => b.key),
  overrides: bindings.states.flatMap((b) => {
    const kept = config.states.find((k) => k.key === b.key);
    if (kept == null) return [];
    const fields = OVERRIDES.filter((f) => b[f] != null && kept[f] == null);
    return fields.length === 0 ? [] : [{ key: b.key, fields }];
  }),
  commands: bindings.commands
    .filter((b) => !config.commands.some((k) => k.key === b.key))
    .map((b) => b.key),
});

describe("dropOrphans", () => {
  it.each(cases)("should drop $name", ({ after, orphans }) => {
    const config = schematic.symbolNodeConfigZ.parse({
      variant: "symbol",
      spec: { type: "builtin", name: "valve" },
      ...bindings,
    });
    const dropped = schematic.dropOrphans(
      config,
      after.states ?? before.states,
      after.commands ?? before.commands,
    );
    expect(removed(dropped)).toEqual(orphans);
    expect(dropped.spec).toEqual(config.spec);
  });
});
