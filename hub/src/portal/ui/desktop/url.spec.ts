// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { activateURL, SCHEME, STATE } from "@/portal/ui/desktop/url";

describe("desktop link", () => {
  describe("STATE", () => {
    it("should accept a URL-safe value of at least sixteen characters", () => {
      expect(STATE.test("a".repeat(16))).toBe(true);
      expect(STATE.test("a".repeat(15))).toBe(false);
      expect(STATE.test("a b".padEnd(16, "a"))).toBe(false);
    });
  });
  describe("activateURL", () => {
    it("should carry every field on the app's scheme", () => {
      const url = new URL(
        activateURL("s".repeat(16), {
          token: "t.o.k",
          secret: "sec ret",
          activation: "act",
          email: "a@b.c",
        }),
      );
      expect(url.protocol).toBe(`${SCHEME}:`);
      expect(url.host).toBe("activate");
      expect(url.searchParams.get("state")).toBe("s".repeat(16));
      expect(url.searchParams.get("token")).toBe("t.o.k");
      expect(url.searchParams.get("secret")).toBe("sec ret");
      expect(url.searchParams.get("activation")).toBe("act");
      expect(url.searchParams.get("email")).toBe("a@b.c");
    });
  });
});
