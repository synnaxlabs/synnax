// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Ranger } from "@/ranger";
import { mockBoundingClientRect } from "@/testutil/dom";

const NOW = TimeStamp.now().nanoseconds;
const HOUR = Number(TimeSpan.HOUR.valueOf());
const UNSET = TimeStamp.MAX.nanoseconds;

/** Opens the `index`th time cell of a rendered timeline. */
const openCell = (container: HTMLElement, index: number): void => {
  const triggers = container.querySelectorAll(".pluto-time-cell__trigger");
  fireEvent.click(triggers[index]);
};

describe("Ranger.TimelineEffect", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should name the stage an action would land the range in", () => {
    const { container } = render(
      <Ranger.Timeline
        value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
        onChange={() => {}}
      />,
    );
    openCell(container, 0);
    // Now falls inside the range, so starting it early runs it.
    expect(screen.getByText("In progress")).toBeTruthy();
  });

  it("should say nothing when a commit leaves the stage and the other end alone", () => {
    const { container } = render(
      <Ranger.Timeline value={{ start: NOW - HOUR, end: UNSET }} onChange={() => {}} />,
    );
    openCell(container, 0);
    const slots = document.querySelectorAll(".pluto-datetime__effect");
    expect(slots.length).toBeGreaterThan(0);
    slots.forEach((slot) => expect(slot.textContent).toBe(""));
  });

  it("should warn where a crossing edit drags the other end", () => {
    const { container } = render(
      <Ranger.Timeline
        value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
        onChange={() => {}}
      />,
    );
    openCell(container, 0);
    expect(screen.getByText(/^Moves end to /)).toBeTruthy();
  });

  it("should report a cleared end rather than an instant", () => {
    const { container } = render(
      <Ranger.Timeline
        value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
        onChange={() => {}}
      />,
    );
    openCell(container, 0);
    expect(screen.getByText("Clears end")).toBeTruthy();
  });
});
