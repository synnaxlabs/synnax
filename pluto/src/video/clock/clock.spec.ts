// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { clock } from "@/video/clock";

const START = TimeStamp.seconds(100);
const WALL = 1_000_000;

describe("clock", () => {
  it("should show the time it stopped at whatever the wall time", () => {
    const c = clock.stopped(START, WALL);
    expect(clock.timeAt(c, WALL + 5_000)).toEqual(START);
  });

  it("should advance with the wall time at its speed while it plays", () => {
    const c = clock.play(clock.stopped(START, WALL), WALL);
    expect(clock.timeAt(c, WALL + 2_000)).toEqual(START.add(TimeSpan.seconds(2)));
    const fast = clock.setSpeed(c, 4, WALL + 2_000);
    expect(clock.timeAt(fast, WALL + 3_000)).toEqual(START.add(TimeSpan.seconds(6)));
  });

  it("should stop at the time it shows when it pauses", () => {
    const c = clock.pause(clock.play(clock.stopped(START, WALL), WALL), WALL + 1_500);
    expect(c.playing).toBe(false);
    expect(clock.timeAt(c, WALL + 9_000)).toEqual(
      START.add(TimeSpan.milliseconds(1_500)),
    );
  });

  it("should play on from a seek", () => {
    const c = clock.play(clock.stopped(START, WALL), WALL);
    const target = START.add(TimeSpan.seconds(30));
    const moved = clock.seek(c, target, WALL + 1_000);
    expect(clock.timeAt(moved, WALL + 2_000)).toEqual(target.add(TimeSpan.seconds(1)));
  });

  it("should parse a clock that crossed a thread", () => {
    const c = clock.play(clock.stopped(START, WALL), WALL);
    const cloned = structuredClone(c);
    expect(clock.clockZ.parse(cloned).at).toEqual(START);
  });
});
