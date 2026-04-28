// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { debounce } from "@/debounce/debounce";
import { TimeSpan } from "@/telem/telem";

describe("debounce", () => {
  it("should debounce the execution of the given function", () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));
    debounced(10);
    debounced(20);
    debounced(30);
    debounced(40);
    expect(fn).toHaveBeenCalledTimes(0);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(40);
  });
  it("should accept any CrudeTimeSpan equivalently", () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.MILLISECOND.valueOf() * 100n);
    debounced(1);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith(1);
  });
  it("should not debounce the execution of the given function if the time is 0", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.ZERO);
    expect(debounced).toBe(fn);
  });
});
