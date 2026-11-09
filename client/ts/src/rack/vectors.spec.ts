// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { caseconv } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { actionZ } from "@/rack/types.gen";

// The Core and every client decode these action envelopes alike.
const VECTORS_URL = new URL(
  "../../../../core/testdata/actions/vectors.json",
  import.meta.url,
);

interface Vector {
  name: string;
  action: { type: string };
  value?: unknown;
}

const vectors = (
  JSON.parse(readFileSync(VECTORS_URL, "utf-8")) as Record<string, Vector[]>
).rack;

describe("rack action vectors", () => {
  it.each(vectors.map((v) => [v.name, v] as const))(
    "should decode the action: %s",
    (_, v) => {
      const action = actionZ.parse(caseconv.snakeToCamel(v.action));
      expect(action.type).toBe(v.action.type);
      if (!("value" in v)) return;
      const value =
        action.type === "set"
          ? action.set.value
          : action.type === "insert"
            ? action.insert.value
            : expect.fail(`${v.name}: only a set or insert holds a value`);
      expect(value ?? null).toEqual(v.value);
    },
  );
});
