// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Clock } from "@/telem/aether/static";

describe("Clock", () => {
  const PERIOD = TimeSpan.milliseconds(500);

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(1200);
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should count the periods elapsed since the epoch", () => {
    const c = new Clock({ period: PERIOD });
    expect(c.value()).toBe(2);
    vi.advanceTimersByTime(300);
    expect(c.value()).toBe(3);
    c.cleanup();
  });

  it("should notify at each period boundary", () => {
    const c = new Clock({ period: PERIOD });
    const handler = vi.fn();
    c.onChange(handler);
    vi.advanceTimersByTime(299);
    expect(handler).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(handler).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1000);
    expect(handler).toHaveBeenCalledTimes(3);
    c.cleanup();
  });

  it("should tick in phase with a clock created earlier", () => {
    const first = new Clock({ period: PERIOD });
    vi.advanceTimersByTime(150);
    const second = new Clock({ period: PERIOD });
    const firstHandler = vi.fn();
    const secondHandler = vi.fn();
    first.onChange(firstHandler);
    second.onChange(secondHandler);
    vi.advanceTimersByTime(149);
    expect(firstHandler).not.toHaveBeenCalled();
    expect(secondHandler).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(firstHandler).toHaveBeenCalledTimes(1);
    expect(secondHandler).toHaveBeenCalledTimes(1);
    expect(first.value()).toBe(second.value());
    first.cleanup();
    second.cleanup();
  });

  it("should stop notifying once cleaned up", () => {
    const c = new Clock({ period: PERIOD });
    const handler = vi.fn();
    c.onChange(handler);
    c.cleanup();
    vi.advanceTimersByTime(2000);
    expect(handler).not.toHaveBeenCalled();
  });
});
