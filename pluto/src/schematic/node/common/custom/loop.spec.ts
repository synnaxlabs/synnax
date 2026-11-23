// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Custom } from "@/schematic/node/common/custom";

const createLoop = () => {
  const callbacks: FrameRequestCallback[] = [];
  const cancelFrame = vi.fn();
  const loop = new Custom.Loop({
    requestFrame: (cb) => callbacks.push(cb),
    cancelFrame,
  });
  const frame = (now: number) => callbacks.shift()?.(now);
  return { loop, callbacks, cancelFrame, frame };
};

describe("Custom.Loop", () => {
  it("should ask for one frame while parts move", () => {
    const { loop, callbacks } = createLoop();
    loop.add({ step: () => true });
    loop.add({ step: () => true });
    expect(callbacks).toHaveLength(1);
  });

  it("should step every part on each frame at the frame time", () => {
    const { loop, frame } = createLoop();
    const step = vi.fn(() => true);
    loop.add({ step });
    frame(16);
    frame(32);
    expect(step.mock.calls).toEqual([[16], [32]]);
  });

  it("should stop asking for frames when the last part stops", () => {
    const { loop, callbacks, frame } = createLoop();
    loop.add({ step: () => false });
    frame(16);
    expect(loop.size).toBe(0);
    expect(callbacks).toHaveLength(0);
  });

  it("should cancel its frame when the last part leaves", () => {
    const { loop, cancelFrame } = createLoop();
    const mover = { step: () => true };
    loop.add(mover);
    loop.delete(mover);
    expect(cancelFrame).toHaveBeenCalledOnce();
  });
});
