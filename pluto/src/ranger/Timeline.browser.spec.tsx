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
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Ranger } from "@/ranger";

const HOUR = Number(TimeSpan.HOUR.valueOf());

/** @returns true if the two elements share a line, by vertical overlap. */
const shareLine = (a: Element, b: Element): boolean => {
  const ra = a.getBoundingClientRect();
  const rb = b.getBoundingClientRect();
  return ra.top < rb.bottom && rb.top < ra.bottom;
};

describe("Ranger.Timeline", () => {
  it("should keep each word on the line of the time it names", async () => {
    const now = TimeStamp.now().nanoseconds;
    const { container } = await renderThemed(
      <div>
        <Ranger.Timeline
          value={{ start: now - 2 * HOUR, end: now - HOUR }}
          onChange={() => {}}
        />
      </div>,
    );
    const frame = container.firstElementChild as HTMLElement;
    const words = [
      ...container.querySelectorAll(
        `.${CSS.BE("range-timeline", "word")}, .${CSS.BE("range-timeline", "dot")}`,
      ),
    ];
    expect(words.length).toBe(3);
    for (let width = 160; width <= 640; width += 8) {
      frame.style.width = `${width}px`;
      for (const word of words) {
        const time = word.nextElementSibling;
        if (time == null) throw new Error(`no time after ${word.textContent}`);
        expect(shareLine(word, time), `${word.textContent} at ${width}px`).toBe(true);
      }
    }
  });
});
