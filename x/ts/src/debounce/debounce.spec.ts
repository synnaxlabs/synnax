// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { debounce, throttle } from "@/debounce/debounce";

describe("debounce", () => {
  describe("debounce", () => {
    it("should debounce the execution of the given function", () => {
      vi.useFakeTimers();
      const fn = vi.fn();
      const debounced = debounce(fn, 100);
      debounced(10);
      debounced(20);
      debounced(30);
      debounced(40);
      expect(fn).toHaveBeenCalledTimes(0);
      vi.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenCalledWith(40);
    });
    it("should not debounce the execution of the given function if the time is 0", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, 0);
      // assert that debounced is the same
      expect(debounced).toBe(fn);
    });
  });
  describe("throttle", () => {
    it("should throttle the execution of the given function", () => {
      vi.useFakeTimers();
      const fn = vi.fn();
      const debounced = throttle(fn, 100);
      debounced(10);
      debounced(20);
      debounced(30);
      debounced(40);
      vi.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenCalledWith(10);
      vi.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledTimes(1);
    });
  });
});
