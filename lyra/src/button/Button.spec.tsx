// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";
import { Icon } from "@/icon";
import { Triggers } from "@/triggers";

describe("Button", () => {
  describe("Basic Rendering", () => {
    it("should render a button with the provided text", () => {
      const c = render(<Button.Button size="small">Hello</Button.Button>);
      const el = c.getByText("Hello");
      expect(el).toBeTruthy();
      expect(el.tagName).toBe("BUTTON");
    });
  });

  describe("element override", () => {
    it("should render a button with the provided element", () => {
      const c = render(<Button.Button el="div">Hello</Button.Button>);
      expect(c.getByText("Hello").tagName).toBe("DIV");
    });
  });

  describe("chassis keyboard activation", () => {
    it("should activate a focusable div chassis on Enter", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button el="div" tabIndex={0} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.keyDown(c.getByText("Hello"), { key: "Enter" });
      expect(onClick).toHaveBeenCalledTimes(1);
    });
    it("should activate a focusable div chassis on Space", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button el="div" tabIndex={0} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.keyDown(c.getByText("Hello"), { key: " " });
      expect(onClick).toHaveBeenCalledTimes(1);
    });
    it("should activate a div chassis with roving tabIndex -1", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button el="div" tabIndex={-1} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.keyDown(c.getByText("Hello"), { key: "Enter" });
      expect(onClick).toHaveBeenCalledTimes(1);
    });
    it("should not activate a non-focusable div chassis", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button el="div" onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.keyDown(c.getByText("Hello"), { key: "Enter" });
      expect(onClick).not.toHaveBeenCalled();
    });
    it("should not activate when the keystroke lands on a nested element", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button el="div" tabIndex={0} onClick={onClick}>
          <span>Nested</span>
        </Button.Button>,
      );
      fireEvent.keyDown(c.getByText("Nested"), { key: "Enter" });
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("preview", () => {
    it("should block clicks and leave the tab order", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button preview onClick={onClick}>
          Hello
        </Button.Button>,
      );
      const el = c.getByText("Hello");
      expect(el.tabIndex).toBe(-1);
      fireEvent.click(el);
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("Toggle", () => {
    it("should read as pressed when checked", () => {
      const c = render(
        <Button.Toggle value onChange={vi.fn()}>
          Hello
        </Button.Toggle>,
      );
      expect(c.getByRole("button", { pressed: true })).toBeTruthy();
    });
    it("should read as not pressed when unchecked", () => {
      const c = render(
        <Button.Toggle value={false} onChange={vi.fn()}>
          Hello
        </Button.Toggle>,
      );
      expect(c.getByRole("button", { pressed: false })).toBeTruthy();
    });
    it("should toggle on click", () => {
      const onChange = vi.fn();
      const c = render(
        <Button.Toggle value={false} onChange={onChange}>
          Hello
        </Button.Toggle>,
      );
      fireEvent.click(c.getByText("Hello"));
      expect(onChange).toHaveBeenCalledWith(true);
    });
  });

  describe("draggable", () => {
    it("should not prevent default on mousedown for a draggable button, so a native dragstart can begin", () => {
      const c = render(
        <Button.Button el="div" tabIndex={-1} draggable>
          Drag
        </Button.Button>,
      );
      const notPrevented = fireEvent.mouseDown(c.getByText("Drag"));
      expect(notPrevented).toBe(true);
    });

    it("should prevent default on mousedown for a non-draggable tabIndex=-1 button", () => {
      const c = render(
        <Button.Button el="div" tabIndex={-1}>
          NoDrag
        </Button.Button>,
      );
      const notPrevented = fireEvent.mouseDown(c.getByText("NoDrag"));
      expect(notPrevented).toBe(false);
    });
  });

  describe("onClick", () => {
    it("should not propagate the click event to the parent", () => {
      const onClick = vi.fn();
      const onParentClick = vi.fn();
      const c = render(
        <div onClick={onParentClick}>
          <Button.Button onClick={onClick}>Hello</Button.Button>
        </div>,
      );
      fireEvent.click(c.getByText("Hello"));
      expect(onClick).toHaveBeenCalled();
      expect(onParentClick).not.toHaveBeenCalled();
    });
    it("should propagate the click event to the parent when the propagateClick prop is true", () => {
      const onClick = vi.fn();
      const onParentClick = vi.fn();
      const c = render(
        <div onClick={onParentClick}>
          <Button.Button onClick={onClick} propagateClick>
            Hello
          </Button.Button>
        </div>,
      );
      fireEvent.click(c.getByText("Hello"));
      expect(onClick).toHaveBeenCalled();
      expect(onParentClick).toHaveBeenCalled();
    });
  });

  describe("preventClick", () => {
    it("should not call the onClick handler when the preventClick prop is true", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button preventClick onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.click(c.getByText("Hello"));
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should leave the tab order when the preventClick prop is true", () => {
      const c = render(<Button.Button preventClick>Hello</Button.Button>);
      expect(c.getByText("Hello").tabIndex).toBe(-1);
    });

    it("should cancel the press default on the chassis itself", () => {
      const c = render(
        <Button.Button preventClick el="div">
          Hello
        </Button.Button>,
      );
      expect(fireEvent.mouseDown(c.getByText("Hello"))).toBe(false);
    });

    it("should leave the press default alone for a focusable descendant", () => {
      const c = render(
        <Button.Button preventClick el="div">
          <input aria-label="alias" />
        </Button.Button>,
      );
      expect(fireEvent.mouseDown(c.getByLabelText("alias"))).toBe(true);
    });
  });

  describe("disabled", () => {
    it("should read as disabled when the disabled prop is true", () => {
      const c = render(<Button.Button disabled>Hello</Button.Button>);
      expect(c.getByText("Hello").getAttribute("aria-disabled")).toBe("true");
    });

    it("should read as disabled when the status is disabled", () => {
      const c = render(<Button.Button status="disabled">Hello</Button.Button>);
      expect(c.getByText("Hello").getAttribute("aria-disabled")).toBe("true");
    });

    it("should read as disabled when the status is loading", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByText("Hello").getAttribute("aria-disabled")).toBe("true");
    });

    it("should not read as disabled by default", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").getAttribute("aria-disabled")).toBeNull();
    });

    it("should not call the onClick handler when the disabled prop is true", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button disabled onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.click(c.getByText("Hello"));
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("onClickDelay", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should require the caller to press and hold the button for the onClickDelay to be triggered", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button onClickDelay={1000} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.mouseDown(c.getByText("Hello"));
      expect(onClick).not.toHaveBeenCalled();
      vi.advanceTimersByTime(10000);
      expect(onClick).toHaveBeenCalled();
    });

    it("should not call onClick if the button is not held down for the onClickDelay", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button onClickDelay={1000} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.mouseDown(c.getByText("Hello"));
      vi.advanceTimersByTime(10);
      fireEvent.mouseUp(c.getByText("Hello"));
      vi.advanceTimersByTime(1000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should not fire after the button unmounts mid-hold", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button onClickDelay={1000} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.mouseDown(c.getByText("Hello"));
      c.unmount();
      vi.advanceTimersByTime(2000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should cancel the hold when the button becomes disabled", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button onClickDelay={1000} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.mouseDown(c.getByText("Hello"));
      c.rerender(
        <Button.Button onClickDelay={1000} onClick={onClick} disabled>
          Hello
        </Button.Button>,
      );
      vi.advanceTimersByTime(2000);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("should ignore a secondary-button hold", () => {
      const onClick = vi.fn();
      const c = render(
        <Button.Button onClickDelay={1000} onClick={onClick}>
          Hello
        </Button.Button>,
      );
      fireEvent.mouseDown(c.getByText("Hello"), { button: 2 });
      vi.advanceTimersByTime(10000);
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("loading", () => {
    it("should not display a loading indicator when the status is not loading", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").querySelector("svg")).toBeNull();
    });

    it("should display a loading indicator when the status is loading", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByText("Hello").querySelector("svg")).not.toBeNull();
    });

    it("should display the content along with the loading indicator when the button is not square", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByText("Hello")).toBeTruthy();
    });

    it("should not display the content when the button is square and the status is loading", () => {
      const c = render(
        <Button.Button status="loading">
          <Icon.Access aria-label="access" />
        </Button.Button>,
      );
      expect(c.queryByLabelText("access")).toBeNull();
      expect(c.container.querySelectorAll("svg")).toHaveLength(1);
    });
  });

  describe("link", () => {
    it("should display the button as an anchor when an href is set", () => {
      const c = render(
        <Button.Button href="https://www.google.com">Hello</Button.Button>,
      );
      expect(c.getByText("Hello").tagName).toBe("A");
    });

    it("should not display the button as an anchor when an href is not set", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").tagName).not.toBe("A");
    });
  });

  describe("triggerIndicator", () => {
    it("should not display a trigger indicator when the triggerIndicator is not set", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.queryByLabelText("trigger-indicator")).not.toBeTruthy();
    });

    it("should display a trigger indicator when the triggerIndicator is set", () => {
      const c = render(
        <Button.Button triggerIndicator={["Enter"]}>Hello</Button.Button>,
      );
      expect(c.getByLabelText("trigger-indicator")).toBeTruthy();
    });

    it("should not display the trigger indicator when triggerIndicator is true and no trigger has been set", () => {
      const c = render(<Button.Button triggerIndicator>Hello</Button.Button>);
      expect(c.queryByLabelText("trigger-indicator")).not.toBeTruthy();
    });

    it("should display the trigger indicator when triggerIndicator is true and a trigger has been set", () => {
      const c = render(
        <Button.Button triggerIndicator trigger={["Enter"]}>
          Hello
        </Button.Button>,
      );
      expect(c.getByLabelText("trigger-indicator")).toBeTruthy();
    });

    it("should keep the indicator out of the button's accessible name", () => {
      const c = render(
        <Button.Button trigger={["Control", "Enter"]} triggerIndicator>
          Save
        </Button.Button>,
      );
      // The keycaps are decoration. Without aria-hidden they join the accessible name,
      // so every name-based query for a hinted button breaks.
      expect(c.getByRole("button", { name: "Save" })).toBeTruthy();
    });
  });

  describe("Triggers", () => {
    it("Should call onClick when the trigger is triggered", () => {
      const onClick = vi.fn();
      const c = render(
        <Triggers.Provider>
          <Button.Button trigger={["T"]} onClick={onClick}>
            Hello
          </Button.Button>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Hello"));
      fireEvent.keyDown(c.container, { code: "T" });
      fireEvent.keyUp(c.container, { code: "T" });
      expect(onClick).toHaveBeenCalledTimes(2);
    });

    it("Should not call onClick when the button is inside an inactive scope", () => {
      const onClick = vi.fn();
      const c = render(
        <Triggers.Provider>
          <Triggers.Scope active={false}>
            <Button.Button trigger={["T"]} onClick={onClick}>
              Hello
            </Button.Button>
          </Triggers.Scope>
        </Triggers.Provider>,
      );
      fireEvent.keyDown(c.container, { code: "T" });
      fireEvent.keyUp(c.container, { code: "T" });
      expect(onClick).not.toHaveBeenCalled();
    });
  });

  describe("textVariant", () => {
    it("should render the link variant as an anchor", () => {
      const c = render(<Button.Button textVariant="link">Hello</Button.Button>);
      expect(c.getByText("Hello").tagName).toBe("A");
    });
  });
});
