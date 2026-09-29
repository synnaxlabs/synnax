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

import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

const DELAY = TimeSpan.milliseconds(100);
const SKIP_DELAY = TimeSpan.milliseconds(300);

const BUTTON_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 140,
  width: 100,
  height: 40,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const ZERO_RECT = {
  left: 0,
  top: 0,
  right: 0,
  bottom: 0,
  width: 0,
  height: 0,
  x: 0,
  y: 0,
  toJSON: () => ({}),
} as DOMRect;

const getTooltip = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".pluto-tooltip");

const mockRect = (el: HTMLElement, rect: DOMRect = BUTTON_RECT): void => {
  el.getBoundingClientRect = () => (el.isConnected ? rect : ZERO_RECT);
};

const Wrapper = ({ children }: { children: ReactNode }): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY} skipDelay={SKIP_DELAY}>
      {children}
    </Tooltip.Config>
  </Triggers.Provider>
);

const hover = (el: HTMLElement): void => {
  fireEvent.pointerOver(el, { pointerType: "mouse" });
};

const unhover = (el: HTMLElement): void => {
  fireEvent.pointerOut(el, { pointerType: "mouse" });
};

const advance = (span: TimeSpan): void => {
  act(() => {
    vi.advanceTimersByTime(span.milliseconds);
  });
};

const PAST_DELAY = TimeSpan.milliseconds(DELAY.milliseconds + 10);
const PAST_CLOSE = TimeSpan.milliseconds(200);

describe("Tooltip.Dialog", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  const mountButton = (key: string = "a") => {
    const res = render(
      <Tooltip.Dialog>
        Tip content
        <button key={key}>Target</button>
      </Tooltip.Dialog>,
      { wrapper: Wrapper },
    );
    const btn = res.getByText("Target");
    mockRect(btn);
    return { ...res, btn };
  };

  describe("opening", () => {
    it("should show the tooltip above the trigger after the delay", () => {
      const { btn } = mountButton();
      hover(btn);
      expect(getTooltip()).toBeNull();
      advance(PAST_DELAY);
      const tooltip = getTooltip();
      expect(tooltip).not.toBeNull();
      expect(tooltip?.style.left).toBe("150px");
      expect(tooltip?.style.top).toBe("94px");
      expect(tooltip?.textContent).toBe("Tip content");
    });

    it("should not show the tooltip when the cursor leaves before the delay", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(TimeSpan.milliseconds(50));
      unhover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).toBeNull();
    });

    it("should never open for touch pointers", () => {
      const { btn } = mountButton();
      fireEvent.pointerOver(btn, { pointerType: "touch" });
      advance(PAST_DELAY);
      expect(getTooltip()).toBeNull();
    });

    it("should not open when hide is set", () => {
      const res = render(
        <Tooltip.Dialog hide>
          Tip content
          <button>Target</button>
        </Tooltip.Dialog>,
        { wrapper: Wrapper },
      );
      const btn = res.getByText("Target");
      mockRect(btn);
      hover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).toBeNull();
    });
  });

  describe("closing", () => {
    it("should close after the cursor leaves the trigger", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).not.toBeNull();
      unhover(btn);
      advance(PAST_CLOSE);
      expect(getTooltip()).toBeNull();
    });

    it("should close immediately on pointer down", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      fireEvent.pointerDown(btn, { pointerType: "mouse" });
      expect(getTooltip()).toBeNull();
    });

    it("should close on Escape", () => {
      const { btn, container } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).not.toBeNull();
      fireEvent.keyDown(container, { code: "Escape" });
      expect(getTooltip()).toBeNull();
    });

    it("should close when an ancestor of the trigger scrolls", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).not.toBeNull();
      fireEvent.scroll(document);
      expect(getTooltip()).toBeNull();
    });

    it("should close when hide becomes true while open", () => {
      const { btn, rerender } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      expect(getTooltip()).not.toBeNull();
      rerender(
        <Tooltip.Dialog hide>
          Tip content
          <button key="a">Target</button>
        </Tooltip.Dialog>,
      );
      expect(getTooltip()).toBeNull();
    });
  });

  describe("warm window", () => {
    it("should reopen instantly within the skip delay after a close", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      unhover(btn);
      advance(PAST_CLOSE);
      expect(getTooltip()).toBeNull();
      hover(btn);
      expect(getTooltip()).not.toBeNull();
    });

    it("should wait for the full delay once the warm window expires", () => {
      const { btn } = mountButton();
      hover(btn);
      advance(PAST_DELAY);
      unhover(btn);
      advance(PAST_CLOSE);
      advance(SKIP_DELAY);
      hover(btn);
      expect(getTooltip()).toBeNull();
      advance(PAST_DELAY);
      expect(getTooltip()).not.toBeNull();
    });
  });

  describe("anchor replacement", () => {
    it("should never render at the screen corner when the trigger is replaced", () => {
      const { btn, rerender } = mountButton("a");
      hover(btn);
      rerender(
        <Tooltip.Dialog>
          Tip content
          <button key="b">Target</button>
        </Tooltip.Dialog>,
      );
      expect(btn.isConnected).toBe(false);
      advance(PAST_DELAY);
      expect(getTooltip()).toBeNull();
    });
  });

  describe("singleton", () => {
    it("should close an open tooltip when another opens", () => {
      const res = render(
        <>
          <Tooltip.Dialog>
            Tip A<button>A</button>
          </Tooltip.Dialog>
          <Tooltip.Dialog>
            Tip B<button>B</button>
          </Tooltip.Dialog>
        </>,
        { wrapper: Wrapper },
      );
      const a = res.getByText("A");
      const b = res.getByText("B");
      mockRect(a);
      mockRect(b);
      hover(a);
      advance(PAST_DELAY);
      expect(getTooltip()?.textContent).toBe("Tip A");
      hover(b);
      advance(PAST_DELAY);
      const tooltips = document.querySelectorAll(".pluto-tooltip");
      expect(tooltips).toHaveLength(1);
      expect(tooltips[0].textContent).toBe("Tip B");
    });
  });

  describe("accessibility", () => {
    it("should describe the trigger with the tooltip while open", () => {
      const { btn } = mountButton();
      expect(btn.getAttribute("aria-describedby")).toBeNull();
      hover(btn);
      advance(PAST_DELAY);
      const tooltip = getTooltip();
      expect(tooltip?.getAttribute("role")).toBe("tooltip");
      expect(btn.getAttribute("aria-describedby")).toBe(tooltip?.id);
      unhover(btn);
      advance(PAST_CLOSE);
      expect(btn.getAttribute("aria-describedby")).toBeNull();
    });

    it("should open instantly on keyboard focus and close on blur", () => {
      const { btn } = mountButton();
      // jsdom parses :focus-visible but never matches it; treat focus as
      // focus-visible like a keyboard-driven browser would
      const matches = btn.matches.bind(btn);
      btn.matches = ((sel: string) =>
        sel === ":focus-visible"
          ? document.activeElement === btn
          : matches(sel)) as typeof btn.matches;
      act(() => btn.focus());
      expect(getTooltip()).not.toBeNull();
      act(() => btn.blur());
      expect(getTooltip()).toBeNull();
    });
  });
});
