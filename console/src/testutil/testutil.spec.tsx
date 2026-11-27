// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/lyra/status";
import { TimeSpan } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { CaptureStatuses } from "@/testutil";

const quiet: Status.Log = () => {};

describe("CaptureStatuses", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should keep a status after its toast expires", () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date", "Temporal"] });
    const onStatuses = vi.fn();
    const { result } = renderHook(() => Status.useAdder(), {
      wrapper: ({ children }: PropsWithChildren) => (
        <Status.Aggregator log={quiet}>
          {children}
          <CaptureStatuses onStatuses={onStatuses} />
        </Status.Aggregator>
      ),
    });
    act(() => result.current({ variant: "error", message: "Failed to save change" }));
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(8).milliseconds));
    expect(onStatuses).toHaveBeenLastCalledWith([
      expect.objectContaining({ message: "Failed to save change" }),
    ]);
  });
});
