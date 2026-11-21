// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { Fragment, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Icon } from "@/icon";
import { renderThemed, resolve } from "@/styletest";
import { Tag } from "@/tag";

const LABELS = ["Hotfire", "Stand 2", "Pass", "Calibration", "Cold flow", "Anomaly"];
const NOUN: Tag.TagsProps["noun"] = { one: "label", other: "labels" };

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

const count = (frame: HTMLElement): string | null => {
  const chip = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
  return getComputedStyle(chip).display === "none"
    ? null
    : getComputedStyle(chip, "::after")
        .content.replace(/"\s*"/g, "")
        .replaceAll('"', "");
};

const stacked = (frame: HTMLElement): boolean =>
  frame.querySelector(`.${CSS.B("tags")}`)!.hasAttribute("data-stacked");

// Whether the step has fitted the group.
const fitted = (frame: HTMLElement): boolean =>
  frame
    .querySelector<HTMLElement>(`.${CSS.B("tags")}`)!
    .style.getPropertyValue("--lyra-tags-gap") !== "";

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

// Whether the count's face, which reaches back under its dots, starts inside its group.
const faceInside = (frame: HTMLElement): boolean => {
  const tags = frame.querySelector<HTMLElement>(`.${CSS.B("tags")}`)!;
  const more = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
  const face =
    more.getBoundingClientRect().left +
    parseFloat(getComputedStyle(more, "::before").left);
  const inner =
    tags.getBoundingClientRect().left +
    parseFloat(getComputedStyle(tags).borderLeftWidth);
  return face >= inner - 0.5;
};

const overflows = (frame: HTMLElement): boolean => {
  const tags = frame.querySelector(`.${CSS.B("tags")}`)!;
  return tags.getBoundingClientRect().right > frame.getBoundingClientRect().right + 0.5;
};

describe("Tags", () => {
  it("should show every tag when they fit", async () => {
    const frame = await renderTags();
    await expect.poll(() => fitted(frame)).toBe(true);
    expect(named(frame)).toEqual(LABELS);
    expect(count(frame)).toBeNull();
  });

  it("should fold the last tags into a count when the container narrows", async () => {
    const frame = await renderTags();
    frame.style.width = "260px";
    await expect.poll(() => count(frame)).not.toBeNull();
    const step = named(frame);
    expect(step.length).toBeLessThan(LABELS.length);
    expect(step).toEqual(LABELS.slice(0, step.length));
    expect(count(frame)).toBe(`+${LABELS.length - step.length}`);
    expect(dots(frame)).toEqual(LABELS.slice(step.length, step.length + 3));
    expect(overflows(frame)).toBe(false);
  });

  it("should show every tag again when the container widens", async () => {
    const frame = await renderTags();
    frame.style.width = "200px";
    await expect.poll(() => count(frame)).not.toBeNull();
    frame.style.width = "1000px";
    await expect.poll(() => named(frame)).toEqual(LABELS);
    await expect.poll(() => count(frame)).toBeNull();
  });

  it("should fold the same at a width however it got there", async () => {
    const frame = await renderTags();
    frame.style.width = "300px";
    await expect.poll(() => count(frame)).not.toBeNull();
    const direct = named(frame);
    frame.style.width = "120px";
    await expect.poll(() => named(frame).length).toBeLessThan(direct.length);
    frame.style.width = "300px";
    await expect.poll(() => named(frame)).toEqual(direct);
  });

  it.each([110, 160])(
    "should keep the count's face inside the group with no tag shown at %ipx",
    async (width) => {
      const frame = await renderTags();
      frame.style.width = `${width}px`;
      await expect.poll(() => named(frame)).toEqual([]);
      await expect.poll(() => count(frame)).not.toBeNull();
      expect(faceInside(frame)).toBe(true);
    },
  );

  it("should keep the outline of an outlined group as its tags stack", async () => {
    const group = (width: number): ReactElement => (
      <div data-testid="frame" style={{ display: "flex", width: `${width}px` }}>
        <Tag.Tags variant="outlined">
          {LABELS.map((label) => (
            <Tag.Tag key={label} color="#3774d0">
              {label}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>
    );
    const { container } = await renderThemed(
      <>
        {group(1000)}
        {group(330)}
      </>,
    );
    const [wide, narrow] = Array.from(
      container.querySelectorAll<HTMLElement>("[data-testid=frame]"),
    );
    await expect.poll(() => stacked(narrow)).toBe(true);
    const outline = (frame: HTMLElement): string =>
      getComputedStyle(frame.querySelector(`.${CSS.B("tags")}`)!).borderTopColor;
    expect(outline(wide)).not.toBe(resolve("transparent"));
    expect(outline(narrow)).toBe(outline(wide));
  });

  it("should fold tags with no color dot into a count with no dots", async () => {
    const { container } = await renderThemed(
      <div data-testid="frame" style={{ display: "flex", width: "260px" }}>
        <Tag.Tags>
          {LABELS.map((label) => (
            <Tag.Tag key={label} icon={<Icon.Python />}>
              {label}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>,
    );
    const frame = container.querySelector<HTMLElement>("[data-testid=frame]")!;
    await expect.poll(() => count(frame)).not.toBeNull();
    expect(dots(frame)).toEqual([]);
    const more = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
    const face =
      more.getBoundingClientRect().left +
      parseFloat(getComputedStyle(more, "::before").left);
    const last = visible(frame).at(-1)!;
    expect(face).toBeGreaterThanOrEqual(last.getBoundingClientRect().right);
    const { paddingLeft, paddingRight } = getComputedStyle(more);
    expect(paddingLeft).toBe(paddingRight);
    await userEvent.hover(more);
    const list = await expect
      .poll(() => document.querySelector(`.${CSS.BE("tags", "hidden")}`))
      .not.toBeNull()
      .then(() => document.querySelector(`.${CSS.BE("tags", "hidden")}`)!);
    expect(list.textContent).toContain(LABELS.at(-1));
    expect(list.querySelector("svg")).toBeNull();
  });

  it.each([240, 110])(
    "should keep the count's face off the shown tags when a hidden tag has no dot at %ipx",
    async (width) => {
      const { container } = await renderThemed(
        <div data-testid="frame" style={{ display: "flex", width: `${width}px` }}>
          <Tag.Tags>
            {LABELS.map((label, i) => (
              <Tag.Tag key={label} color={i >= 4 ? "#3774d0" : undefined}>
                {label}
              </Tag.Tag>
            ))}
          </Tag.Tags>
        </div>,
      );
      const frame = container.querySelector<HTMLElement>("[data-testid=frame]")!;
      // The group fits again once its font loads, which moves the count.
      await document.fonts.ready;
      await scheduler.nextFrame();
      await scheduler.nextFrame();
      await expect.poll(() => count(frame)).not.toBeNull();
      expect(dots(frame)).toEqual(LABELS.slice(4));
      expect(faceInside(frame)).toBe(true);
      const shown = visible(frame).filter(hasName);
      if (shown.length === 0) return;
      const more = frame.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
      const face =
        more.getBoundingClientRect().left +
        parseFloat(getComputedStyle(more, "::before").left);
      expect(face).toBeGreaterThan(
        shown[shown.length - 1].getBoundingClientRect().right,
      );
    },
  );

  it("should center a tag's close over its dot", async () => {
    const { container } = await renderThemed(
      <Tag.Tags>
        <Tag.Tag color="#3774d0" onClose={() => {}}>
          Hotfire
        </Tag.Tag>
      </Tag.Tags>,
    );
    const tag = container.querySelector<HTMLElement>(`.${CSS.B("tag")}`)!;
    const center = (el: Element): number => {
      const { left, width } = el.getBoundingClientRect();
      return left + width / 2;
    };
    const dot = tag.querySelector(":scope > svg")!;
    const close = tag.querySelector(`.${CSS.BE("tag", "close")} svg`)!;
    await expect.poll(() => Math.abs(center(close) - center(dot))).toBeLessThan(2);
  });

  it("should hand a click on its count to onCountClick alone", async () => {
    const onClick = vi.fn();
    const onCountClick = vi.fn();
    const { container } = await renderThemed(
      <div data-testid="frame" style={{ display: "flex", width: "260px" }}>
        <Tag.Tags onClick={onClick} onCountClick={onCountClick}>
          {LABELS.map((label) => (
            <Tag.Tag key={label} color="#3774d0">
              {label}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>,
    );
    const frame = container.querySelector<HTMLElement>("[data-testid=frame]")!;
    await expect.poll(() => count(frame)).not.toBeNull();
    await userEvent.click(frame.querySelector(`.${CSS.BE("tags", "more")}`)!);
    expect(onCountClick).toHaveBeenCalledOnce();
    expect(onClick).not.toHaveBeenCalled();
  });

  it("should give a stacked tag the hover of a full tag", async () => {
    const group = (width: number): ReactElement => (
      <div data-testid="frame" style={{ display: "flex", width: `${width}px` }}>
        <Tag.Tags onCountClick={() => {}}>
          {LABELS.map((label) => (
            <Tag.Tag key={label} color="#3774d0">
              {label}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>
    );
    const { container } = await renderThemed(
      <>
        {group(1000)}
        {group(330)}
      </>,
    );
    const [wide, narrow] = Array.from(
      container.querySelectorAll<HTMLElement>("[data-testid=frame]"),
    );
    await expect.poll(() => stacked(narrow)).toBe(true);
    // A stacked tag paints its face and outline on a pseudo-element.
    const look = (tag: HTMLElement, face: boolean): string[] => {
      const box = getComputedStyle(tag, face ? "::before" : undefined);
      return [box.borderTopColor, box.backgroundColor, getComputedStyle(tag).color];
    };
    const hovered = async (tag: HTMLElement, face: boolean): Promise<string[]> => {
      await userEvent.hover(tag);
      return look(tag, face);
    };
    const full = await hovered(visible(wide)[0], false);
    const shown = visible(narrow).filter(hasName);
    expect(await hovered(shown[0], true)).toEqual(full);
    expect(await hovered(shown[shown.length - 1], true)).toEqual(full);
  });

  it("should keep a capped name inside a stacked tag with no tag over it", async () => {
    const { container } = await renderThemed(
      <div data-testid="frame" style={{ display: "flex", width: "600px" }}>
        <Tag.Tags>
          {LABELS.map((label) => (
            <Tag.Tag key={label} color="#3774d0">
              {`${label} sensor metrics cpu percentage`}
            </Tag.Tag>
          ))}
        </Tag.Tags>
      </div>,
    );
    const frame = container.querySelector<HTMLElement>("[data-testid=frame]")!;
    await expect.poll(() => stacked(frame)).toBe(true);
    const inside = (tag: HTMLElement): boolean =>
      tag.querySelector(`.${CSS.B("text")}`)!.getBoundingClientRect().right <=
      tag.getBoundingClientRect().right;
    const shown = visible(frame).filter(hasName);
    const last = shown[shown.length - 1];
    expect(last.hasAttribute("data-last")).toBe(true);
    expect(inside(last)).toBe(true);
    await userEvent.hover(shown[0]);
    await expect.poll(() => shown[0].hasAttribute("data-lifted")).toBe(true);
    expect(inside(shown[0])).toBe(true);
  });

  it("should keep its width as it hides tags", async () => {
    const { container } = await renderThemed(
      <div style={{ display: "flex", width: "1000px" }}>
        <div data-testid="frame" style={{ display: "flex", maxWidth: "260px" }}>
          <Tag.Tags>
            {LABELS.map((label) => (
              <Tag.Tag key={label} color="#3774d0">
                {label}
              </Tag.Tag>
            ))}
          </Tag.Tags>
        </div>
      </div>,
    );
    const frame = container.querySelector<HTMLElement>("[data-testid=frame]")!;
    await expect.poll(() => count(frame)).not.toBeNull();
    frame.style.maxWidth = "1000px";
    await expect.poll(() => named(frame)).toEqual(LABELS);
  });
});

const renderGroup = (labels: string[], noun?: Tag.TagsProps["noun"]): ReactElement => (
  <div data-testid="group" style={{ display: "flex", width: "100%" }}>
    <Tag.Tags noun={noun}>
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
        {renderGroup(LABELS, NOUN)}
        {renderGroup(LABELS.slice(0, 3), NOUN)}
      </Tag.StepProvider>
    </div>,
  );
  return Array.from(container.querySelectorAll<HTMLElement>("[data-testid=group]"));
};

// A box to the pixel. The lift's margin may round its neighbors by a fraction of one.
const box = (el: Element): string => {
  const r = el.getBoundingClientRect();
  return `${Math.round(r.left)},${Math.round(r.width)}`;
};

describe("Stepped tags", () => {
  it("should show every tag at full width when the list has room", async () => {
    const [all, three] = await renderStepped(1000);
    await expect.poll(() => fitted(all)).toBe(true);
    expect(stacked(all)).toBe(false);
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
    await expect.poll(() => stacked(all)).toBe(true);
    expect(named(three)).toEqual(LABELS.slice(0, 3));
    expect(stacked(three)).toBe(false);
  });

  it("should lift a hovered stacked tag without moving the others", async () => {
    const [all] = await renderStepped(330);
    await expect.poll(() => stacked(all)).toBe(true);
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
    // It draws over the next tag where they overlap.
    const r = first.getBoundingClientRect();
    expect(
      first.contains(document.elementFromPoint(r.right - 2, r.top + r.height / 2)),
    ).toBe(true);
    expect(others()).toEqual(before);
  });

  it("should show the first tags as dots and the total when no tag fits", async () => {
    const [all] = await renderStepped(150);
    await expect.poll(() => count(all)).toBe(`${LABELS.length} labels`);
    expect(getComputedStyle(all.querySelector(`.${CSS.B("tags")}`)!).visibility).toBe(
      "visible",
    );
    expect(named(all)).toEqual([]);
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
  });

  it("should count with +N when it is not told what its tags are", async () => {
    const { container } = await renderThemed(
      <div style={{ width: "150px" }}>
        <Tag.StepProvider>{renderGroup(LABELS)}</Tag.StepProvider>
      </div>,
    );
    const all = container.querySelector<HTMLElement>("[data-testid=group]")!;
    await expect.poll(() => count(all)).toBe(`+${LABELS.length}`);
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
  });

  it("should fit the other groups again when a group goes", async () => {
    const list = (groups: string[][]): ReactElement => (
      <div style={{ width: "150px" }}>
        <Tag.StepProvider>
          {groups.map((labels) => (
            <Fragment key={labels.length}>{renderGroup(labels)}</Fragment>
          ))}
        </Tag.StepProvider>
      </div>
    );
    const { container, rerender } = await renderThemed(list([LABELS, ["Pass"]]));
    const last = (): HTMLElement =>
      Array.from(container.querySelectorAll<HTMLElement>("[data-testid=group]")).at(
        -1,
      )!;
    await expect.poll(() => count(last())).not.toBeNull();
    await rerender(list([["Pass"]]));
    await expect.poll(() => named(last())).toEqual(["Pass"]);
    expect(count(last())).toBeNull();
  });

  it("should keep a hovered tag lifted as text beside its group changes", async () => {
    const { container } = await renderThemed(
      <div style={{ width: "330px" }}>
        <Tag.StepProvider>
          {renderGroup(LABELS)}
          <span data-testid="clock">0</span>
        </Tag.StepProvider>
      </div>,
    );
    const all = container.querySelector<HTMLElement>("[data-testid=group]")!;
    await expect.poll(() => stacked(all)).toBe(true);
    const [first] = visible(all);
    await userEvent.hover(first);
    await expect.poll(() => first.hasAttribute("data-lifted")).toBe(true);
    container.querySelector("[data-testid=clock]")!.firstChild!.nodeValue = "1";
    await scheduler.nextFrame();
    await scheduler.nextFrame();
    expect(first.hasAttribute("data-lifted")).toBe(true);
  });

  it("should show the first tags as dots and a count when the total does not fit", async () => {
    const [all] = await renderStepped(110);
    await expect.poll(() => count(all)).toBe(`+${LABELS.length}`);
    expect(named(all)).toEqual([]);
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
    expect(overflows(all)).toBe(false);
  });

  it("should hide the tags of a group too narrow for its dots and count", async () => {
    const [all] = await renderStepped(60);
    const tags = all.querySelector<HTMLElement>(`.${CSS.B("tags")}`)!;
    const more = all.querySelector<HTMLElement>(`.${CSS.BE("tags", "more")}`)!;
    await expect.poll(() => getComputedStyle(more).visibility).toBe("hidden");
    expect(visible(all).every((t) => getComputedStyle(t).visibility === "hidden")).toBe(
      true,
    );
    expect(getComputedStyle(tags).visibility).toBe("visible");
  });

  it("should keep a pressed group on the shared step", async () => {
    const [all, three] = await renderStepped(150);
    await expect.poll(() => count(all)).toBe(`${LABELS.length} labels`);
    // A press renders the group with a new class name, as a dialog trigger does.
    const tags = all.querySelector<HTMLElement>(`.${CSS.B("tags")}`)!;
    await userEvent.click(tags);
    await userEvent.click(tags);
    await scheduler.nextFrame();
    expect(dots(all)).toEqual(LABELS.slice(0, 3));
    expect(dots(three)).toEqual(LABELS.slice(0, 3));
    expect(count(all)).toBe(`${LABELS.length} labels`);
  });
});
