// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { renderThemed } from "@/styletest";
import { Tag } from "@/tag";

const LABELS = ["Hotfire", "Stand 2", "Pass", "Calibration", "Cold flow", "Anomaly"];

const renderTags = async (): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <div data-testid="frame" style={{ display: "flex", width: "1000px" }}>
      <Tag.Tags>
        {LABELS.map((label) => (
          <Tag.Tag key={label} color="#3774d0">
            {label}
          </Tag.Tag>
        ))}
      </Tag.Tags>
    </div>,
  );
  return container.querySelector<HTMLElement>("[data-testid=frame]")!;
};

const shown = (frame: HTMLElement): string[] =>
  Array.from(frame.querySelectorAll<HTMLElement>(`.${CSS.B("tag")}`))
    .filter((tag) => {
      const style = getComputedStyle(tag);
      return style.display !== "none" && style.visibility !== "hidden";
    })
    .map((tag) => tag.textContent ?? "");

const more = (frame: HTMLElement): string | null =>
  frame.querySelector(`.${CSS.BE("tags", "more")}`)?.textContent ?? null;

const overflows = (frame: HTMLElement): boolean => {
  const tags = frame.querySelector(`.${CSS.B("tags")}`)!;
  return tags.getBoundingClientRect().right > frame.getBoundingClientRect().right + 0.5;
};

describe("Tags", () => {
  it("should show every tag when they fit", async () => {
    const frame = await renderTags();
    await expect.poll(() => shown(frame)).toEqual(LABELS);
    expect(more(frame)).toBeNull();
  });

  it("should fold the last tags into a count when the container narrows", async () => {
    const frame = await renderTags();
    frame.style.width = "260px";
    await expect.poll(() => more(frame)).not.toBeNull();
    const visible = shown(frame);
    expect(visible.length).toBeLessThan(LABELS.length);
    expect(visible).toEqual(LABELS.slice(0, visible.length));
    expect(more(frame)).toContain(`+${LABELS.length - visible.length}`);
    expect(overflows(frame)).toBe(false);
  });

  it("should show every tag again when the container widens", async () => {
    const frame = await renderTags();
    frame.style.width = "200px";
    await expect.poll(() => more(frame)).not.toBeNull();
    frame.style.width = "1000px";
    await expect.poll(() => shown(frame)).toEqual(LABELS);
    await expect.poll(() => more(frame)).toBeNull();
  });

  it("should fold the same at a width however it got there", async () => {
    const frame = await renderTags();
    frame.style.width = "300px";
    await expect.poll(() => more(frame)).not.toBeNull();
    const direct = shown(frame);
    frame.style.width = "120px";
    await expect.poll(() => shown(frame).length).toBeLessThan(direct.length);
    frame.style.width = "300px";
    await expect.poll(() => shown(frame)).toEqual(direct);
  });
});

const renderStepped = async (step: number): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <Tag.StepProvider>
      <div
        data-testid="frame"
        style={{ display: "flex", width: "600px", "--lyra-tags-shown": step } as object}
      >
        <Tag.Tags>
          {LABELS.map((label) => (
            <Tag.Tag key={label} color="#3774d0">
              {label}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>
    </Tag.StepProvider>,
  );
  return container.querySelector<HTMLElement>("[data-testid=frame]")!;
};

// The count draws its text from a counter of the hidden tags.
const count = (frame: HTMLElement): string | null => {
  const chip = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`);
  if (chip == null || getComputedStyle(chip).display === "none") return null;
  return getComputedStyle(chip, "::after").content;
};

const COUNTED = '"+" counter(lyra-tags-hidden)';

describe("Stepped tags", () => {
  it("should show the number of tags its container sets, and count the rest", async () => {
    const frame = await renderStepped(2);
    await expect.poll(() => shown(frame)).toEqual(LABELS.slice(0, 2));
    expect(count(frame)).toBe(`${COUNTED} " labels"`);
  });

  it("should show the first tags as dots in the count when it sets no tags", async () => {
    const frame = await renderStepped(0);
    await expect.poll(() => shown(frame)).toEqual(LABELS.slice(0, 3));
    const dots = Array.from(
      frame.querySelectorAll(`.${CSS.B("tag")} .${CSS.B("text")}`),
    )
      .slice(0, 3)
      .map((t) => getComputedStyle(t).display);
    expect(dots).toEqual(["none", "none", "none"]);
    expect(count(frame)).toBe(COUNTED);
  });

  it("should name one hidden label in the singular", async () => {
    const frame = await renderStepped(5);
    await expect.poll(() => shown(frame)).toEqual(LABELS.slice(0, 5));
    expect(count(frame)).toBe(`"+1 label"`);
  });

  it("should show no count when every tag fits the step", async () => {
    const frame = await renderStepped(5);
    frame.style.setProperty("--lyra-tags-shown", "6");
    await expect.poll(() => shown(frame)).toEqual(LABELS);
    expect(count(frame)).toBeNull();
  });
});
