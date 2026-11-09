// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { parse } from "@/feature/embedded/log";

describe("parse", () => {
  it("should quote a field that a reader could not tell apart, and write others as JSON", () => {
    const [entry] = parse(
      JSON.stringify({
        level: "info",
        ts: 0,
        msg: "opened",
        key: "simple",
        text: "has space",
        empty: "",
        count: 3,
        nested: { a: 1 },
      }),
    );
    expect(entry.text).toBe(
      'opened key=simple text="has space" empty="" count=3 nested={"a":1}',
    );
  });
});
