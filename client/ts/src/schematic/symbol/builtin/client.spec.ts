// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("builtin", () => {
  describe("retrieve", () => {
    it("should return the symbols that the Core ships", async () => {
      const builtins = await client.schematics.symbols.builtins.retrieve();
      expect(builtins).toHaveLength(85);
      const valve = builtins.find((b) => b.name === "valve");
      expect(valve?.states.map((s) => s.key)).toEqual(["active"]);
      expect(valve?.commands.map((c) => [c.key, c.name])).toEqual([
        ["deactivate", "Close"],
        ["activate", "Open"],
      ]);
      const light = builtins.find((b) => b.name === "light");
      expect(light?.states).toHaveLength(1);
      expect(light?.commands).toEqual([]);
    });
  });
});
