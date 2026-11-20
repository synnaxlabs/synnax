// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { Ranger } from "@/ranger";

const NOW = TimeStamp.now().nanoseconds;
const HOUR = Number(TimeSpan.HOUR.valueOf());
const DAY = Number(TimeSpan.DAY.valueOf());
const UNSET = TimeStamp.MAX.nanoseconds;

/** Opens the `index`th time cell of a rendered timeline. */
const openCell = async (container: HTMLElement, index: number): Promise<Locator> => {
  const triggers = container.querySelectorAll(".lyra-time-editor__trigger");
  await userEvent.click(triggers[index]);
  return page.getByRole("textbox");
};

const action = (text: string): Locator => page.getByRole("menuitem", { name: text });

const EDITOR_EFFECT = ".lyra-time-effect";
const STAGE_EFFECT = ".pluto-stage-button__effect";

/** The text the effect footer shows, or null while it is collapsed. */
const effectText = (selector = EDITOR_EFFECT): string | null => {
  const effect = document.querySelector(selector);
  if (effect == null) throw new Error(`no ${selector}`);
  return effect.classList.contains("lyra-time-effect--visible")
    ? effect.textContent
    : null;
};

describe("Ranger.TimelineEffect", () => {
  describe("time editor", () => {
    it("should name the stage a hovered action would land the range in", async () => {
      const { container } = await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      await openCell(container, 0);
      await userEvent.hover(action("Now"));
      await expect.poll(() => effectText()).toBe("To doIn progress");
    });

    it("should say nothing when a commit leaves the stage and the other end alone", async () => {
      const { container } = await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW - HOUR, end: UNSET }}
          onChange={() => {}}
        />,
      );
      await openCell(container, 0);
      await userEvent.hover(action("Now"));
      expect(effectText()).toBeNull();
    });

    it("should describe the highlighted reading when no action is hovered", async () => {
      const { container } = await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      const field = await openCell(container, 0);
      await userEvent.fill(field, "now + 3h");
      await expect.poll(() => effectText()).toMatch(/^CompletedTo doMoves end to /);
    });

    it("should return to the highlighted reading when the pointer leaves an action", async () => {
      const { container } = await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      await openCell(container, 0);
      await userEvent.hover(action("Now"));
      await expect.poll(() => effectText()).not.toBeNull();
      await userEvent.unhover(action("Now"));
      // The highlighted reading is the current value, which changes nothing.
      await expect.poll(() => effectText()).toBeNull();
    });

    it("should keep its last content while it collapses out", async () => {
      const { container } = await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW + HOUR, end: NOW + 2 * HOUR }}
          onChange={() => {}}
        />,
      );
      await openCell(container, 0);
      await userEvent.hover(action("Now"));
      await expect.poll(() => effectText()).not.toBeNull();
      await userEvent.unhover(action("Now"));
      await expect.poll(() => effectText()).toBeNull();
      expect(document.querySelector(EDITOR_EFFECT)?.textContent).toBe(
        "To doIn progress",
      );
    });

    it("should report a cleared end with the value it loses", async () => {
      const end = NOW + 2 * HOUR;
      const { container } = await renderThemed(
        <Ranger.Timeline value={{ start: NOW + HOUR, end }} onChange={() => {}} />,
      );
      await openCell(container, 0);
      await userEvent.hover(action("Unschedule"));
      await expect.poll(() => effectText()).toMatch(/^Clears endwas /);
    });

    it("should commit on an outside click while it is visible", async () => {
      const onChange = vi.fn();
      const { container, getByTestId } = await renderThemed(
        <>
          <Ranger.Timeline
            value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
            onChange={onChange}
          />
          <div data-testid="outside" style={{ marginTop: 600, height: 20 }} />
        </>,
      );
      const field = await openCell(container, 0);
      await userEvent.fill(field, "end + 1h");
      await expect.poll(() => effectText()).not.toBeNull();
      await userEvent.click(getByTestId("outside"));
      expect(onChange).toHaveBeenCalledOnce();
    });

    it("should drop digits below the row's resolution", async () => {
      // A range over a day long reads to the minute.
      const start = NOW - 3 * DAY;
      const end = NOW - DAY + 1_234_567_000;
      const { container } = await renderThemed(
        <Ranger.Timeline value={{ start, end }} onChange={() => {}} />,
      );
      const field = await openCell(container, 0);
      await userEvent.fill(field, "end + 1h");
      await expect.poll(() => effectText()).toMatch(/Moves end to \D+ \d{2}:\d{2}$/);
    });
  });

  describe("stage menu", () => {
    it("should say what a hovered transition would clear", async () => {
      await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      await page.getByText("Completed").click();
      await userEvent.hover(action("Reopen"));
      await expect
        .poll(() => effectText(STAGE_EFFECT))
        .toMatch(/^CompletedIn progressClears endwas /);
    });

    it("should show no effect until a transition is hovered", async () => {
      await renderThemed(
        <Ranger.Timeline
          value={{ start: NOW - 2 * HOUR, end: NOW - HOUR }}
          onChange={() => {}}
        />,
      );
      await page.getByText("Completed").click();
      await expect.element(action("Reopen")).toBeVisible();
      expect(effectText(STAGE_EFFECT)).toBeNull();
    });
  });
});
