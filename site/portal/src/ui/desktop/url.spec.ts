// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { activateURL, read, SCHEME, STATE } from "@/ui/desktop/url";

const STATE_VALUE = "s".repeat(16);
const HASH = "a".repeat(64);
const NOT_FROM_APP = "This link did not come from the Synnax Desktop app.";

const query = (fields: Record<string, string>): URLSearchParams =>
  new URLSearchParams({ state: STATE_VALUE, name: "bench", fp: HASH, ...fields });

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
          key: "t.o.k",
          secret: "sec ret",
          activation: "act",
          email: "a@b.c",
        }),
      );
      expect(url.protocol).toBe(`${SCHEME}:`);
      expect(url.host).toBe("activate");
      expect(url.searchParams.get("state")).toBe("s".repeat(16));
      expect(url.searchParams.get("key")).toBe("t.o.k");
      expect(url.searchParams.get("secret")).toBe("sec ret");
      expect(url.searchParams.get("activation")).toBe("act");
      expect(url.searchParams.get("email")).toBe("a@b.c");
    });
  });
  describe("read", () => {
    it("should read a query the app sent", () => {
      expect(read(query({ name: "  bench  ", fp: `${HASH}, ${HASH}` }))).toEqual({
        state: STATE_VALUE,
        name: "bench",
        fingerprint: [HASH],
        problem: null,
      });
    });
    it("should trim the name to its bound", () => {
      const { name } = read(query({ name: "n".repeat(MAX_NAME_LENGTH + 10) }));
      expect(name).toHaveLength(MAX_NAME_LENGTH);
    });
    it("should refuse a malformed state", () => {
      expect(read(query({ state: "short" })).problem).toBe(NOT_FROM_APP);
    });
    it("should refuse a blank name", () => {
      expect(read(query({ name: "  " })).problem).toBe(NOT_FROM_APP);
    });
    it("should refuse a fingerprint that is not host hashes", () => {
      const { fingerprint, problem } = read(query({ fp: "nope" }));
      expect(fingerprint).toEqual([]);
      expect(problem).toBe(NOT_FROM_APP);
    });
    it("should refuse an empty query", () => {
      expect(read(new URLSearchParams()).problem).toBe(NOT_FROM_APP);
    });
  });
});
