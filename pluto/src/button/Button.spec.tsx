// Copyright 2025 Synnax Labs, Inc.
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

  describe("size", () => {
    it("should render a medium button by default", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-medium");
    });
    it("should render a small button if the size is small", () => {
      const c = render(<Button.Button size="small">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-small");
    });
    it("should render a large button if the size is large", () => {
      const c = render(<Button.Button size="large">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-large");
    });
    it("should render a huge button if the size is huge", () => {
      const c = render(<Button.Button size="huge">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-huge");
    });
    it("should render a tiny button if the size is tiny", () => {
      const c = render(<Button.Button size="tiny">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-tiny");
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

    it("should add the prevent-click class to the button when the preventClick prop is true", () => {
      const c = render(<Button.Button preventClick>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--prevent-click");
    });
  });

  describe("disabled", () => {
    it("should add the disabled class to the button when the disabled prop is true", () => {
      const c = render(<Button.Button disabled>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--disabled");
    });

    it("should add the disabled class to the button when the status is disabled", () => {
      const c = render(<Button.Button status="disabled">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--disabled");
    });

    it("should add the disabled class to the button when the status is loading", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--disabled");
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

    it("should add the btn-delay style to the button when the onClickDelay prop is set", () => {
      const c = render(<Button.Button onClickDelay={1000}>Hello</Button.Button>);
      expect(c.getByText("Hello").style.getPropertyValue("--pluto-btn-delay")).toBe(
        "1s",
      );
    });

    it("should not add the btn-delay style to the button when the onClickDelay prop is 0", () => {
      const c = render(<Button.Button onClickDelay={0}>Hello</Button.Button>);
      expect(c.getByText("Hello").style.getPropertyValue("--pluto-btn-delay")).toBe("");
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
  });

  describe("variant", () => {
    it("should render an outlined button by default", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--outlined");
    });
    it("should render a filled button if the variant is filled", () => {
      const c = render(<Button.Button variant="filled">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--filled");
    });
    it("should render a text button if the variant is text", () => {
      const c = render(<Button.Button variant="text">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--text");
    });
  });

  describe("status", () => {
    it("should not add a status class to the button when the status is not provided", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).not.toContain("pluto--status-success");
      expect(c.getByText("Hello").className).not.toContain("pluto--status-error");
      expect(c.getByText("Hello").className).not.toContain("pluto--status-warning");
    });
    it("should add the status class to the button when the status is success", () => {
      const c = render(<Button.Button status="success">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--status-success");
    });
    it("should add the status class to the button when the status is error", () => {
      const c = render(<Button.Button status="error">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--status-error");
    });
    it("should add the status class to the button when the status is warning", () => {
      const c = render(<Button.Button status="warning">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--status-warning");
    });
    it("should add the status class to the button when the status is loading", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--status-loading");
    });
  });

  describe("loading", () => {
    it("should not display a loading indicator when the status is not loading", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").children).not.toContain("Loading...");
    });
    it("should display a loading indicator when the status is loading", () => {
      const c = render(<Button.Button status="loading">Hello</Button.Button>);
      expect(c.getByLabelText("pluto-icon--loading")).toBeTruthy();
    });

    it("should display the content along with the loading indicator when the button is not square", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").children).toBeTruthy();
    });

    it("should not display the content when the button is square and the status is loading", () => {
      const c = render(
        <Button.Button status="loading">
          <Icon.Access />
        </Button.Button>,
      );
      const el = c.queryByLabelText("pluto-icon--access");
      expect(el?.parentElement).not.toBeTruthy();
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

  describe("contrast", () => {
    it("should not set the contrast class to the button when the contrast is not set or false", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).not.toContain("contrast");
    });
    it("should set the contrast class to the button when the contrast is set", () => {
      const c = render(<Button.Button contrast={0}>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--contrast-0");
    });
    it("should not set the contrast class to the button when the contrast is not set", () => {
      const c = render(<Button.Button contrast={1}>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-btn--contrast-1");
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

    it("should set the trigger text level to a level below the button", () => {
      const c = render(
        <Button.Button triggerIndicator={["Enter"]} level="p">
          Hello
        </Button.Button>,
      );
      expect(c.getByLabelText("trigger-indicator").className).toContain(
        "pluto-text--small",
      );
    });
  });

  describe("customColor", () => {
    it("should not add the custom-color class to the button when the color is not set", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).not.toContain("pluto-btn--custom-color");
    });
    it("should allow the caller to set a custom color to the button", () => {
      const c = render(<Button.Button color="#00FF00">Hello</Button.Button>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto-btn--custom-color");
      expect(el.style.getPropertyValue("--pluto-btn-color")).toBe("0, 255, 0");
    });
  });

  describe("textColor", () => {
    it("should allow the caller to set a custom text color to the button", () => {
      const c = render(<Button.Button textColor={0}>Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--color-0");
    });
  });

  describe("gap", () => {
    it("should not set the gap for the default size", () => {
      const c = render(<Button.Button>Hello</Button.Button>);
      expect(c.getByText("Hello").className).not.toContain("gap");
    });
    it("should set the gap to small when the size is small", () => {
      const c = render(<Button.Button size="small">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--gap-small");
    });

    it("should set the gap to small when the size is tiny", () => {
      const c = render(<Button.Button size="tiny">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--gap-small");
    });
  });

  describe("Variants", () => {
    it("should add a filled class to the button when the variant is filled", () => {
      const c = render(
        <Button.Button size="small" variant="filled">
          Hello
        </Button.Button>,
      );
      expect(c.getByText("Hello").className).toContain("pluto-btn--filled");
    });
    it("should add a text class to the button when the variant is text", () => {
      const c = render(
        <Button.Button size="small" variant="text">
          Hello
        </Button.Button>,
      );
      expect(c.getByText("Hello").className).toContain("pluto-btn--text");
    });
    it("should add a outlined class to the button when the variant is outlined", () => {
      const c = render(
        <Button.Button size="small" variant="outlined">
          Hello
        </Button.Button>,
      );
      expect(c.getByText("Hello").className).toContain("pluto-btn--outlined");
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
  });

  describe("textVariant", () => {
    it("should set the text variant to the provided value", () => {
      const c = render(<Button.Button textVariant="link">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto-text--link");
    });
  });

  describe("Sizes", () => {
    it("should add a small class to the button when the size is small", () => {
      const c = render(<Button.Button size="small">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-small");
    });
    it("should add a medium class to the button when the size is medium", () => {
      const c = render(<Button.Button size="medium">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-medium");
    });
    it("should add a large class to the button when the size is large", () => {
      const c = render(<Button.Button size="large">Hello</Button.Button>);
      expect(c.getByText("Hello").className).toContain("pluto--height-large");
    });
  });
});
