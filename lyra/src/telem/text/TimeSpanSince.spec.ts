// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Telem } from "@/telem";

describe("useTimeSpanSince", () => {
  it("should advance the elapsed span every second", () => {
    vi.useFakeTimers();
    try {
      const stamp = TimeStamp.now().sub(TimeSpan.seconds(5));
      const { result } = renderHook(() => Telem.Text.useTimeSpanSince(stamp));
      expect(result.current.equals(TimeSpan.seconds(5))).toBe(true);
      act(() => {
        vi.advanceTimersByTime(TimeSpan.seconds(3).milliseconds);
      });
      expect(result.current.equals(TimeSpan.seconds(8))).toBe(true);
    } finally {
      vi.useRealTimers();
    }
  });
});
