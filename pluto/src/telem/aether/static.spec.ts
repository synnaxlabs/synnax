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
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should count up and notify once every period", () => {
    const c = new Clock({ period: TimeSpan.milliseconds(500) });
    const handler = vi.fn();
    c.onChange(handler);
    expect(c.value()).toBe(0);
    vi.advanceTimersByTime(1250);
    expect(c.value()).toBe(2);
    expect(handler).toHaveBeenCalledTimes(2);
    c.cleanup();
  });

  it("should stop ticking once cleaned up", () => {
    const c = new Clock({ period: TimeSpan.milliseconds(500) });
    const handler = vi.fn();
    c.onChange(handler);
    c.cleanup();
    vi.advanceTimersByTime(2000);
    expect(c.value()).toBe(0);
    expect(handler).not.toHaveBeenCalled();
  });
});
