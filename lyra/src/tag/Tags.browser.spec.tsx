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
    .filter((tag) => getComputedStyle(tag).display !== "none")
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
