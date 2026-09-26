// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { firePointerDown, mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";

const NOW = TimeStamp.now().nanoseconds;
const HOUR = Number(TimeSpan.HOUR.valueOf());
const DAY = Number(TimeSpan.DAY.valueOf());
const UNSET = TimeStamp.MAX.nanoseconds;

/** Opens the `index`th time cell of a rendered timeline. */
const openCell = (container: HTMLElement, index: number): HTMLInputElement => {
  const triggers = container.querySelectorAll(".pluto-time-editor__trigger");
  fireEvent.click(triggers[index]);
  return screen.getByRole<HTMLInputElement>("textbox");
};

const hover = (text: string): void => {
  fireEvent.mouseEnter(screen.getByRole("menuitem", { name: text }));
};

const EDITOR_EFFECT = ".pluto-time-effect";
const STAGE_EFFECT = ".pluto-stage-button__effect";

/** The text the effect footer shows, or null while it is collapsed. */
const effectText = (selector = EDITOR_EFFECT): string | null => {
  const effect = document.querySelector(selector);
  if (effect == null) throw new Error(`no ${selector}`);
  return effect.classList.contains("pluto-time-effect--visible")
    ? effect.textContent
    : null;
};

describe("Ranger.TimelineEffect", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  describe("time editor", () => {
    it("should name the stage a hovered action would land the range in", () => {
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      openCell(container, 0);
      hover("Now");
      expect(effectText()).toBe("To doIn progress");
    });

    it("should say nothing when a commit leaves the stage and the other end alone", () => {
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW - HOUR, end: UNSET }}
          onChange={() => {}}
        />,
      );
      openCell(container, 0);
      hover("Now");
      expect(effectText()).toBeNull();
    });

    it("should describe the highlighted reading when no action is hovered", () => {
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      const field = openCell(container, 0);
      fireEvent.change(field, { target: { value: "now + 3h" } });
      expect(effectText()).toMatch(/^CompletedTo doMoves end to /);
    });

    it("should return to the highlighted reading when the pointer leaves an action", () => {
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      openCell(container, 0);
      hover("Now");
      fireEvent.mouseLeave(screen.getByRole("menuitem", { name: "Now" }));
      // The highlighted reading is the current value, which changes nothing.
      expect(effectText()).toBeNull();
    });

    it("should keep its last content while it collapses out", () => {
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      openCell(container, 0);
      hover("Now");
      fireEvent.mouseLeave(screen.getByRole("menuitem", { name: "Now" }));
      expect(document.querySelector(EDITOR_EFFECT)?.textContent).toBe(
        "To doIn progress",
      );
    });

    it("should report a cleared end with the value it loses", () => {
      const end = NOW + 2 * HOUR;
      const { container } = render(
        <Ranger.Timeline value={{ start: NOW + HOUR, end }} onChange={() => {}} />,
      );
      openCell(container, 0);
      hover("Unschedule");
      expect(effectText()).toMatch(/^Clears endwas /);
    });

    it("should commit on an outside click while it is visible", () => {
      // Every element reads as the same box, so the viewport needs one of its own
      // for a click to land outside the dialog.
      document.documentElement.getBoundingClientRect = mockBoundingClientRect(
        0,
        0,
        1000,
        1000,
      );
      const onChange = vi.fn();
      const { container } = render(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={onChange}
        />,
      );
      const field = openCell(container, 0);
      fireEvent.change(field, { target: { value: "end + 1h" } });
      expect(effectText()).not.toBeNull();
      firePointerDown(document.body, { x: 500, y: 500 });
      expect(onChange).toHaveBeenCalledOnce();
    });

    it("should drop digits below the row's resolution", () => {
      // A range over a day long reads to the minute.
      const start = NOW - 3 * DAY;
      const end = NOW - DAY + 1_234_567_000;
      const { container } = render(
        <Ranger.Timeline value={{ start, end }} onChange={() => {}} />,
      );
      const field = openCell(container, 0);
      fireEvent.change(field, { target: { value: "end + 1h" } });
      expect(effectText()).toMatch(/Moves end to \D+ \d{2}:\d{2}$/);
    });
  });

  describe("stage menu", () => {
    it("should say what a hovered transition would clear", () => {
      render(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      fireEvent.click(screen.getByText("Completed"));
      hover("Reopen");
      expect(effectText(STAGE_EFFECT)).toMatch(/^CompletedIn progressClears endwas /);
    });

    it("should show no effect until a transition is hovered", () => {
      render(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      fireEvent.click(screen.getByText("Completed"));
      expect(effectText(STAGE_EFFECT)).toBeNull();
    });
  });
});
