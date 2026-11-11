// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { memory, sync } from "@/server/audience";
import { memory as directory } from "@/server/directory";

const ADA = { name: "Ada Lovelace", email: "Ada@Example.com" };
const GRACE = { name: "Grace Hopper", email: "grace@example.com" };

describe("audience", () => {
  describe("sync", () => {
    it("should add every account to the list in lowercase", async () => {
      const audience = memory();
      expect(await sync(directory({ people: { a: ADA, g: GRACE } }), audience)).toEqual(
        { added: 2, removed: 0 },
      );
      expect([...audience.list].sort()).toEqual([
        "ada@example.com",
        "grace@example.com",
      ]);
      expect(audience.synced).toEqual(audience.list);
    });

    it("should leave an address added by hand as it is", async () => {
      const audience = memory(["ada@example.com"]);
      expect(await sync(directory({ people: { a: ADA } }), audience)).toEqual({
        added: 0,
        removed: 0,
      });
      expect(audience.synced.size).toBe(0);
    });

    it("should keep a hand-added address after its account is deleted", async () => {
      const audience = memory(["ada@example.com"]);
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(directory(), audience);
      expect([...audience.list]).toEqual(["ada@example.com"]);
    });

    it("should remove the address of a deleted account", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA, g: GRACE } }), audience);
      expect(await sync(directory({ people: { g: GRACE } }), audience)).toEqual({
        added: 0,
        removed: 1,
      });
      expect([...audience.list]).toEqual(["grace@example.com"]);
    });

    it("should move an account to its new address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: ADA } }), audience);
      await sync(
        directory({ people: { a: { ...ADA, email: "ada@acme.co" } } }),
        audience,
      );
      expect([...audience.list]).toEqual(["ada@acme.co"]);
    });

    it("should skip an account with no address", async () => {
      const audience = memory();
      await sync(directory({ people: { a: { ...ADA, email: "" } } }), audience);
      expect(audience.list.size).toBe(0);
    });
  });
});
