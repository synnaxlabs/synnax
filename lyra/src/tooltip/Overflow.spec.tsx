// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Text } from "@/text";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

const DELAY = TimeSpan.milliseconds(100);
const PAST_DELAY = TimeSpan.milliseconds(110);
const PAST_CLOSE = TimeSpan.milliseconds(200);
const LONG_NAME = "Reactor coolant loop outlet temperature";

const LABEL_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 120,
  width: 100,
  height: 20,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const Wrapper = ({ children }: { children: ReactNode }): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY}>
      <Tooltip.Overflow />
      {children}
    </Tooltip.Config>
  </Triggers.Provider>
);

const getTooltips = (): HTMLElement[] => [
  ...document.querySelectorAll<HTMLElement>(".lyra-tooltip"),
];

const mockWidth = (el: HTMLElement, scrollWidth: number): void => {
  Object.defineProperty(el, "scrollWidth", { configurable: true, value: scrollWidth });
  Object.defineProperty(el, "clientWidth", { configurable: true, value: 100 });
  el.getBoundingClientRect = () => LABEL_RECT;
};

const advance = (span: TimeSpan): void => {
  act(() => {
    vi.advanceTimersByTime(span.milliseconds);
  });
};

const renderLabel = (scrollWidth: number): HTMLElement => {
  const c = render(<Text.Text overflow="fade">{LONG_NAME}</Text.Text>, {
    wrapper: Wrapper,
  });
  const label = c.getByText(LONG_NAME);
  mockWidth(label, scrollWidth);
  return label;
};

describe("Tooltip.Overflow", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should show the full text of a cut label after the delay", () => {
    const label = renderLabel(300);
    fireEvent.pointerOver(label);
    expect(getTooltips()).toHaveLength(0);
    advance(PAST_DELAY);
    const tips = getTooltips();
    expect(tips).toHaveLength(1);
    expect(tips[0].textContent).toBe(LONG_NAME);
  });

  it("should not show a tooltip for a label that fits", () => {
    const label = renderLabel(100);
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    expect(getTooltips()).toHaveLength(0);
  });

  it("should not show a tooltip for a label under edit", () => {
    const label = renderLabel(300);
    Object.defineProperty(label, "isContentEditable", { value: true });
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    expect(getTooltips()).toHaveLength(0);
  });

  it("should close when the pointer leaves the label", () => {
    const label = renderLabel(300);
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    fireEvent.pointerOver(document.body);
    advance(PAST_CLOSE);
    expect(getTooltips()).toHaveLength(0);
  });

  it("should close when the pointer leaves the window", () => {
    const label = renderLabel(300);
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    fireEvent.pointerOut(label, { relatedTarget: null });
    advance(PAST_CLOSE);
    expect(getTooltips()).toHaveLength(0);
  });

  it("should close at once on a pointer press", () => {
    const label = renderLabel(300);
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    fireEvent.pointerDown(label);
    expect(getTooltips()).toHaveLength(0);
  });

  it("should yield to the tooltip of an element that holds the label", () => {
    const c = render(
      <Tooltip.Dialog>
        Own tooltip
        <Text.Text overflow="fade">{LONG_NAME}</Text.Text>
      </Tooltip.Dialog>,
      { wrapper: Wrapper },
    );
    const label = c.getByText(LONG_NAME);
    mockWidth(label, 300);
    fireEvent.pointerOver(label);
    advance(PAST_DELAY);
    const tips = getTooltips();
    expect(tips).toHaveLength(1);
    expect(tips[0].textContent).toBe("Own tooltip");
  });
});
