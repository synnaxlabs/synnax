// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

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

const renderGroup = (labels: string[]): ReactElement => (
  <div data-testid="group" style={{ display: "flex", width: "100%" }}>
    <Tag.Tags>
      {labels.map((label) => (
        <Tag.Tag key={label} color="#3774d0">
          {label}
        </Tag.Tag>
      ))}
    </Tag.Tags>
  </div>
);

// Two groups of one list: one with every label, one with the first three.
const renderStepped = async (width: number): Promise<HTMLElement[]> => {
  const { container } = await renderThemed(
    <div data-testid="list" style={{ width: `${width}px` }}>
      <Tag.StepProvider>
        {renderGroup(LABELS)}
        {renderGroup(LABELS.slice(0, 3))}
      </Tag.StepProvider>
    </div>,
  );
  return Array.from(container.querySelectorAll<HTMLElement>("[data-testid=group]"));
};

const count = (frame: HTMLElement): string | null => {
  const chip = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
  return getComputedStyle(chip).display === "none"
    ? null
    : getComputedStyle(chip, "::after").content.replace(/["\s]/g, "");
};

const stacked = (frame: HTMLElement): string =>
  frame
    .querySelector<HTMLElement>(`.${CSS.B("tags")}`)!
    .style.getPropertyValue("--lyra-tags-stacked");

const visible = (frame: HTMLElement): HTMLElement[] =>
  Array.from(frame.querySelectorAll<HTMLElement>(`.${CSS.B("tag")}`)).filter(
    (tag) => getComputedStyle(tag).display !== "none",
  );

const hasName = (tag: HTMLElement): boolean =>
  getComputedStyle(tag.querySelector(`.${CSS.B("text")}`)!).display !== "none";

// The tags that show their names.
const named = (frame: HTMLElement): string[] =>
  visible(frame)
    .filter(hasName)
    .map((tag) => tag.textContent ?? "");

// The tags that show as dots in the count.
const dots = (frame: HTMLElement): string[] =>
  visible(frame)
    .filter((tag) => !hasName(tag))
    .map((tag) => tag.textContent ?? "");

// A box to the pixel. The lift's margin may round its neighbors by a fraction of one.
const box = (el: Element): string => {
  const r = el.getBoundingClientRect();
  return `${Math.round(r.left)},${Math.round(r.width)}`;
};

describe("Stepped tags", () => {
  it("should show every tag at full width when the list has room", async () => {
    const [all, three] = await renderStepped(1000);
    await expect.poll(() => stacked(all)).toBe("false");
    expect(named(all)).toEqual(LABELS);
    expect(named(three)).toEqual(LABELS.slice(0, 3));
    expect(count(all)).toBeNull();
  });

  it("should show one step in every group, and count the rest", async () => {
    const [all, three] = await renderStepped(260);
    await expect.poll(() => count(all)).not.toBeNull();
    const step = named(all).length;
    expect(step).toBeGreaterThan(0);
    expect(count(all)).toBe(`+${LABELS.length - step}`);
    expect(named(three)).toEqual(LABELS.slice(0, Math.min(3, step)));
  });

  it("should show the first hidden tags as dots in the count", async () => {
    const [all] = await renderStepped(260);
    await expect.poll(() => count(all)).not.toBeNull();
    const step = named(all).length;
    expect(dots(all)).toEqual(LABELS.slice(step, step + 3));
  });

  it("should stack only the groups that do not fit at full width", async () => {
    const [all, three] = await renderStepped(330);
    await expect.poll(() => stacked(all)).toBe("true");
    expect(named(three)).toEqual(LABELS.slice(0, 3));
    expect(stacked(three)).toBe("false");
  });

  it("should lift a hovered stacked tag without moving the others", async () => {
    const [all] = await renderStepped(330);
    await expect.poll(() => stacked(all)).toBe("true");
    const [first, ...rest] = visible(all);
    const others = (): string[] =>
      [...rest, all.querySelector(`.${CSS.BE("tags", "more")}`)!].map(box);
    const before = others();
    const cut = first.getBoundingClientRect().width;
    await userEvent.hover(first);
    await expect.poll(() => first.hasAttribute("data-lifted")).toBe(true);
    const full = parseFloat(first.style.getPropertyValue("--lyra-tag-full"));
    expect(full).toBeGreaterThan(cut);
    expect(first.getBoundingClientRect().width).toBeCloseTo(full, 0);
    expect(others()).toEqual(before);
  });

  it("should show the first tags as dots in the count when no tag fits", async () => {
    const [all] = await renderStepped(90);
    await expect.poll(() => count(all)).toBe(`+${LABELS.length}`);
    expect(named(all)).toEqual([]);
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
  });

  it("should keep a pressed group on the shared step", async () => {
    const [all, three] = await renderStepped(90);
    await expect.poll(() => count(all)).toBe(`+${LABELS.length}`);
    // A press renders the group with a new class name, as a dialog trigger does.
    const tags = all.querySelector<HTMLElement>(`.${CSS.B("tags")}`)!;
    await userEvent.click(tags);
    await userEvent.click(tags);
    await new Promise(requestAnimationFrame);
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
    expect(dots(three)).toEqual(LABELS.slice(0, 3));
    expect(count(all)).toBe(`+${LABELS.length}`);
  });
});
