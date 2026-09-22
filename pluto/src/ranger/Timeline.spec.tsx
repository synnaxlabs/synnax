// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";
import { resolutionFor } from "@/ranger/Timeline";
import { mockBoundingClientRect } from "@/testutil/dom";

const HOUR = Number(TimeSpan.HOUR.valueOf());
const MILLISECOND = Number(TimeSpan.MILLISECOND.valueOf());
const SECOND = Number(TimeSpan.SECOND.valueOf());

const triggers = (container: HTMLElement): NodeListOf<HTMLElement> =>
  container.querySelectorAll(".pluto-time-editor__trigger");

describe("resolutionFor", () => {
  it("should coarsen with the span", () => {
    expect(resolutionFor(TimeSpan.days(2)).equals(TimeSpan.MINUTE)).toBe(true);
    expect(resolutionFor(TimeSpan.hours(2)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.minutes(5)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.seconds(15)).equals(TimeSpan.MILLISECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.milliseconds(3)).equals(TimeSpan.MICROSECOND)).toBe(
      true,
    );
  });
});

describe("Ranger.Timeline", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  describe("preview", () => {
    it("should not open a timestamp editor", () => {
      const now = TimeStamp.now().nanoseconds;
      const { container } = render(
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
          preview
        />,
      );
      fireEvent.click(triggers(container)[0]);
      expect(screen.queryByRole("textbox")).toBeNull();
    });

    it("should not open the stage menu", () => {
      const now = TimeStamp.now().nanoseconds;
      render(
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
          preview
        />,
      );
      fireEvent.click(screen.getByText("Completed"));
      expect(screen.queryByText("Reopen")).toBeNull();
    });
  });

  it("should move to in progress when the planned start passes", async () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now + 50 * MILLISECOND, end: now + HOUR }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText("To do")).toBeTruthy();
    await waitFor(() => expect(screen.getByText("In progress")).toBeTruthy());
  });

  it("should move to completed when the planned end passes", async () => {
    const now = TimeStamp.now().nanoseconds;
    render(
      <Ranger.Timeline
        value={{ start: now - HOUR, end: now + 50 * MILLISECOND }}
        onChange={() => {}}
      />,
    );
    expect(screen.getByText("In progress")).toBeTruthy();
    await waitFor(() => expect(screen.getByText("Completed")).toBeTruthy());
  });

  it("should never read an open range finer than the second", () => {
    const start = TimeStamp.now().nanoseconds - 5 * SECOND + 123 * MILLISECOND;
    const { container } = render(
      <Ranger.Timeline
        value={{ start, end: TimeStamp.MAX.nanoseconds }}
        onChange={() => {}}
      />,
    );
    expect(triggers(container)[0].textContent).toMatch(/\d{2}:\d{2}(:\d{2})?$/);
  });

  it("should offer a planned end once a range is scheduled", () => {
    const now = TimeStamp.now().nanoseconds;
    const onChange = vi.fn();
    render(
      <Ranger.Timeline
        value={{ start: now + HOUR, end: TimeStamp.MAX.nanoseconds }}
        onChange={onChange}
      />,
    );
    fireEvent.click(screen.getByText("Set an end time"));
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "start + 2h" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith({ start: now + HOUR, end: now + 3 * HOUR });
  });

  it("should not offer an end before a range is scheduled", () => {
    const unset = TimeStamp.MAX.nanoseconds;
    render(
      <Ranger.Timeline value={{ start: unset, end: unset }} onChange={() => {}} />,
    );
    expect(screen.queryByText("Set an end time")).toBeNull();
  });

  it("should end a running range at its start on a zero elapsed time", () => {
    const now = TimeStamp.now().nanoseconds;
    const start = now - HOUR;
    const onChange = vi.fn();
    const { container } = render(
      <Ranger.Timeline
        value={{ start, end: TimeStamp.MAX.nanoseconds }}
        onChange={onChange}
      />,
    );
    fireEvent.click(triggers(container)[1]);
    const field = screen.getByRole("textbox");
    fireEvent.change(field, { target: { value: "0s" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith({ start, end: start });
  });
});
