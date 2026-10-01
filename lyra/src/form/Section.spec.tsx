// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createEvent, fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Form } from "@/form";

describe("Form.Sections", () => {
  it("should stack its sections by default", () => {
    const c = render(<Form.Sections data-testid="sections" />);
    const classes = c.getByTestId("sections").classList;
    expect(classes).toContain("pluto-form-sections");
    expect(classes).toContain("pluto--direction-y");
  });

  it("should lay its sections side by side when x is set", () => {
    const c = render(<Form.Sections x data-testid="sections" />);
    const classes = c.getByTestId("sections").classList;
    expect(classes).toContain("pluto--direction-x");
    expect(classes).not.toContain("pluto--direction-y");
  });

  describe("wheel scrolling", () => {
    const stubScroll = (
      el: HTMLElement,
      metrics: Partial<
        Record<"scrollWidth" | "clientWidth" | "scrollHeight" | "clientHeight", number>
      >,
    ): void => {
      let scrollLeft = 0;
      Object.defineProperty(el, "scrollLeft", {
        get: () => scrollLeft,
        set: (v: number) => (scrollLeft = v),
        configurable: true,
      });
      Object.entries(metrics).forEach(([k, value]) =>
        Object.defineProperty(el, k, { value, configurable: true }),
      );
    };

    const wheel = (
      target: HTMLElement,
      deltaY: number,
      init: WheelEventInit = {},
    ): Event => {
      const event = createEvent.wheel(target, { cancelable: true, deltaY, ...init });
      fireEvent(target, event);
      return event;
    };

    it("should scroll an overflowing side-by-side strip horizontally", () => {
      const c = render(<Form.Sections x data-testid="sections" />);
      const strip = c.getByTestId("sections");
      stubScroll(strip, { scrollWidth: 500, clientWidth: 200 });
      const event = wheel(strip, 100);
      expect(strip.scrollLeft).toEqual(100);
      expect(event.defaultPrevented).toBe(true);
    });

    it("should leave a Ctrl or Cmd wheel to the browser zoom", () => {
      const c = render(<Form.Sections x data-testid="sections" />);
      const strip = c.getByTestId("sections");
      stubScroll(strip, { scrollWidth: 500, clientWidth: 200 });
      const ctrl = wheel(strip, 100, { ctrlKey: true });
      const meta = wheel(strip, 100, { metaKey: true });
      expect(strip.scrollLeft).toEqual(0);
      expect(ctrl.defaultPrevented).toBe(false);
      expect(meta.defaultPrevented).toBe(false);
    });

    it("should leave the wheel to a stacked layout", () => {
      const c = render(<Form.Sections data-testid="sections" />);
      const strip = c.getByTestId("sections");
      stubScroll(strip, { scrollWidth: 500, clientWidth: 200 });
      const event = wheel(strip, 100);
      expect(strip.scrollLeft).toEqual(0);
      expect(event.defaultPrevented).toBe(false);
    });

    it("should let a vertically scrollable section scroll first", () => {
      const c = render(
        <Form.Sections x data-testid="sections">
          <div data-testid="section" style={{ overflowY: "auto" }}>
            <input aria-label="Name" />
          </div>
        </Form.Sections>,
      );
      const strip = c.getByTestId("sections");
      const section = c.getByTestId("section");
      stubScroll(strip, { scrollWidth: 500, clientWidth: 200 });
      stubScroll(section, { scrollHeight: 300, clientHeight: 100 });
      const event = wheel(c.getByLabelText("Name"), 100);
      expect(strip.scrollLeft).toEqual(0);
      expect(event.defaultPrevented).toBe(false);
    });

    it("should scroll the strip once the section reaches its edge", () => {
      const c = render(
        <Form.Sections x data-testid="sections">
          <div data-testid="section" style={{ overflowY: "auto" }}>
            <input aria-label="Name" />
          </div>
        </Form.Sections>,
      );
      const strip = c.getByTestId("sections");
      const section = c.getByTestId("section");
      stubScroll(strip, { scrollWidth: 500, clientWidth: 200 });
      stubScroll(section, { scrollHeight: 300, clientHeight: 100 });
      Object.defineProperty(section, "scrollTop", { value: 200, configurable: true });
      wheel(c.getByLabelText("Name"), 100);
      expect(strip.scrollLeft).toEqual(100);
    });
  });
});

describe("Form.Section", () => {
  it("should render its title above its fields", () => {
    const c = render(
      <Form.Section title="Connection">
        <input aria-label="Host" />
      </Form.Section>,
    );
    const header = c.getByText("Connection").closest(".pluto-form-section__header");
    const body = c.getByLabelText("Host").closest(".pluto-form-section__body");
    expect(header).not.toBeNull();
    expect(body).not.toBeNull();
    expect(header?.parentElement).toBe(body?.parentElement);
  });

  it("should render its actions in the header", () => {
    const c = render(
      <Form.Section title="Channels" actions={<button>Add</button>}>
        <input aria-label="Name" />
      </Form.Section>,
    );
    expect(
      c.getByRole("button", { name: "Add" }).closest(".pluto-form-section__header"),
    ).not.toBeNull();
  });
});
