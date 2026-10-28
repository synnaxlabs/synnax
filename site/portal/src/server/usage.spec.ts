// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NOW } from "@/server/license/testutil";
import { dayOf, isActive, NONE, since } from "@/server/usage";

const at = (iso: string): Date => new Date(iso);

describe("usage", () => {
  describe("dayOf", () => {
    it("should return the start of the UTC day", () => {
      expect(dayOf(at("2026-09-17T23:59:59.999Z"))).toEqual(at("2026-09-17T00:00:00Z"));
      expect(dayOf(at("2026-09-17T00:00:00Z"))).toEqual(at("2026-09-17T00:00:00Z"));
    });
  });

  describe("since", () => {
    it("should start each period on a UTC day that ends with today", () => {
      expect(since("day", NOW)).toEqual(at("2026-09-17T00:00:00Z"));
      expect(since("week", NOW)).toEqual(at("2026-09-11T00:00:00Z"));
      expect(since("month", NOW)).toEqual(at("2026-08-19T00:00:00Z"));
    });
  });

  describe("isActive", () => {
    const usage = (lastActive: string) => ({ ...NONE, lastActive: at(lastActive) });

    it("should count a user active since midnight UTC as a daily active user", () => {
      expect(isActive(usage("2026-09-17T00:00:00Z"), "day", NOW)).toBe(true);
      expect(isActive(usage("2026-09-16T23:59:59Z"), "day", NOW)).toBe(false);
    });

    it("should count a user active within the period", () => {
      expect(isActive(usage("2026-09-11T00:00:00Z"), "week", NOW)).toBe(true);
      expect(isActive(usage("2026-09-10T23:59:59Z"), "week", NOW)).toBe(false);
      expect(isActive(usage("2026-08-19T00:00:00Z"), "month", NOW)).toBe(true);
      expect(isActive(usage("2026-08-18T23:59:59Z"), "month", NOW)).toBe(false);
    });

    it("should not count a user who was never active", () => {
      expect(isActive(NONE, "month", NOW)).toBe(false);
    });
  });
});
