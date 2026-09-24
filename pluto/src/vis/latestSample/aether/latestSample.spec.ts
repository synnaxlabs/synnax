// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { latestSample } from "@/vis/latestSample/aether";

const START = TimeStamp.seconds(1000);

// The test source has no last write of its own, so specs stamp it by hand.
const setup = (initial: TimeStamp | null = START) => {
  const source = telemTest.source<number>(1);
  let stamp = initial;
  (source as telem.Source<number>).lastWrite = () => stamp;
  const h = renderAether(latestSample.LatestSample, {
    state: { source: telemTest.numberSourceSpec(source) },
  });
  const arrive = (at: TimeStamp | null): void => {
    stamp = at;
    source.setValue(source.value() + 1);
  };
  return { h, source, arrive };
};

describe("latestSample/aether/LatestSample", () => {
  it("should report the source's last write on mount", () => {
    const { h } = setup();
    expect(h.state.time).toEqual(START);
  });

  it("should report null until the source has a last write", () => {
    const { h, arrive } = setup(null);
    expect(h.state.time).toBeNull();
    arrive(START);
    expect(h.state.time).toEqual(START);
  });

  // A remote source opens its stream on the first value read, so the leaf must read.
  it("should read the source's value on mount", () => {
    const source = telemTest.source<number>(1);
    const value = vi.spyOn(source, "value");
    renderAether(latestSample.LatestSample, {
      state: { source: telemTest.numberSourceSpec(source) },
    });
    expect(value).toHaveBeenCalled();
  });

  it("should follow a write one second newer", () => {
    const { h, arrive } = setup();
    arrive(START.add(TimeSpan.SECOND));
    expect(h.state.time).toEqual(START.add(TimeSpan.SECOND));
  });

  it("should skip a write less than one second newer", () => {
    const { h, arrive } = setup();
    arrive(START.add(TimeSpan.milliseconds(999)));
    expect(h.state.time).toEqual(START);
  });

  it("should skip a write older than the last report", () => {
    const { h, arrive } = setup();
    arrive(START.sub(TimeSpan.seconds(5)));
    expect(h.state.time).toEqual(START);
  });

  it("should keep the last report when the source loses its last write", () => {
    const { h, arrive } = setup();
    arrive(null);
    expect(h.state.time).toEqual(START);
  });

  it("should push at most once per second of sample time", () => {
    const { h, arrive } = setup();
    const setState = vi.spyOn(h.component, "setState");
    for (let i = 1; i <= 20; i++) arrive(START.add(TimeSpan.milliseconds(100 * i)));
    expect(setState).toHaveBeenCalledTimes(2);
    expect(h.state.time).toEqual(START.add(TimeSpan.seconds(2)));
  });

  it("should stop listening and clean up the source on delete", () => {
    const { h, source, arrive } = setup();
    const cleanup = vi.spyOn(source, "cleanup");
    h.unmount();
    expect(cleanup).toHaveBeenCalledTimes(1);
    arrive(START.add(TimeSpan.minutes(1)));
    expect(h.state.time).toEqual(START);
  });
});
