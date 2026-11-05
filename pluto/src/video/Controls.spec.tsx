// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Video } from "@/video";

const START = TimeStamp.seconds(1_000);
const RANGE = new TimeRange(START, START.add(TimeSpan.seconds(10)));

// Renders controls that own their clock, and returns a reader of the newest clock.
const renderControls = (initial: Video.clock.Clock): (() => Video.clock.Clock) => {
  let current = initial;
  const Harness = (): ReactElement => {
    const [c, setC] = useState(initial);
    current = c;
    return <Video.Controls range={RANGE} clock={c} onChange={setC} />;
  };
  render(<Harness />);
  return () => current;
};

describe("Controls", () => {
  it("should play from the time the clock shows", () => {
    const read = renderControls(Video.clock.stopped(START.add(TimeSpan.seconds(2))));
    fireEvent.click(screen.getByLabelText("Play"));
    expect(read().playing).toBe(true);
    expect(read().at).toEqual(START.add(TimeSpan.seconds(2)));
  });

  it("should pause at the time the clock shows", () => {
    const read = renderControls(Video.clock.play(Video.clock.stopped(START)));
    fireEvent.click(screen.getByLabelText("Pause"));
    expect(read().playing).toBe(false);
    expect(read().at.after(START)).toBe(true);
  });

  it("should seek to the time of the scrub", () => {
    const read = renderControls(Video.clock.stopped(START));
    fireEvent.change(screen.getByLabelText("Time"), { target: { value: "4000" } });
    expect(read().at).toEqual(START.add(TimeSpan.seconds(4)));
  });

  it("should pause at the end of the range", async () => {
    const late = Video.clock.play(
      Video.clock.stopped(RANGE.end.sub(TimeSpan.milliseconds(1)), Date.now() - 1_000),
      Date.now() - 1_000,
    );
    const read = renderControls(late);
    await act(async () => {});
    expect(read().playing).toBe(false);
    expect(read().at).toEqual(RANGE.end);
  });

  it("should play from the start of a range it played to the end", () => {
    const read = renderControls(Video.clock.stopped(RANGE.end));
    fireEvent.click(screen.getByLabelText("Play"));
    expect(read().playing).toBe(true);
    expect(read().at).toEqual(START);
  });
});
