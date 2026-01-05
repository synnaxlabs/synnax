// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vitest } from "vitest";

import { Icon } from "@/icon";
import { Text } from "@/text";
import { text } from "@/text/core";

describe("Text", () => {
  describe("levels", () => {
    text.LEVELS.forEach((level) => {
      it(`should render text with the correct HTML tag for ${level}`, () => {
        const c = render(<Text.Text level={level}>Hello</Text.Text>);
        expect(c.getByText("Hello").tagName.toLowerCase()).toBe(level);
      });
    });
  });

  describe("gap", () => {
    it("should render the element with a small gap by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--gap-small");
    });

    it("should render the element with a different gap", () => {
      const c = render(<Text.Text gap="medium">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--gap-medium");
    });
  });

  describe("direction", () => {
    it("should render the element with a direction of x by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--direction-x");
    });

    it("should render the element with a different direction", () => {
      const c = render(<Text.Text direction="y">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--direction-y");
    });

    it("should render the element with a different direction", () => {
      const c = render(<Text.Text y>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--direction-y");
    });
  });

  describe("variants", () => {
    it("should not add a variant classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("pluto-text--link");
      expect(c.getByText("Hello").className).not.toContain("pluto-text--code");
      expect(c.getByText("Hello").className).not.toContain("pluto-text--keyboard");
    });

    describe("link", () => {
      it("should render a link if the href is provided", () => {
        const c = render(<Text.Text href="https://www.google.com">Hello</Text.Text>);
        expect(c.getByText("Hello").tagName.toLowerCase()).toBe("a");
      });

      it("should render a link if variant is set to link", () => {
        const c = render(<Text.Text variant="link">Hello</Text.Text>);
        expect(c.getByText("Hello").tagName.toLowerCase()).toBe("a");
      });

      it("should automatically format the href if autoFormatHref is true", () => {
        const c = render(
          <Text.Text href="google.com" autoFormatHref>
            Hello
          </Text.Text>,
        );
        expect((c.getByText("Hello") as HTMLAnchorElement).href).toBe(
          "https://google.com/",
        );
      });
    });

    describe("code", () => {
      it("should render a code block if the variant is set to code", () => {
        const c = render(<Text.Text variant="code">Hello</Text.Text>);
        expect(c.getByText("Hello").className).toContain("pluto-text--code");
      });
    });

    describe("keyboard", () => {
      it("should render a keyboard key if the variant is set to keyboard", () => {
        const c = render(<Text.Text variant="keyboard">Hello</Text.Text>);
        expect(c.getByText("Hello").className).toContain("pluto-text--keyboard");
      });
    });
  });

  describe("element overrides", () => {
    it("should render a custom element if the el prop is provided", () => {
      const c = render(<Text.Text el="span">Hello</Text.Text>);
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("span");
    });

    it("should use a different el if the defaultEl prop is provided", () => {
      const c = render(<Text.Text defaultEl="span">Hello</Text.Text>);
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("span");
    });

    it("should prefer el over defaultEl", () => {
      const c = render(
        <Text.Text el="span" defaultEl="p">
          Hello
        </Text.Text>,
      );
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("span");
    });

    it("should prefer link over defaultEl", () => {
      const c = render(
        <Text.Text defaultEl="p" href="https://www.google.com">
          Hello
        </Text.Text>,
      );
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("a");
    });

    it("should prefer defaultEl over level", () => {
      const c = render(
        <Text.Text defaultEl="span" level="p">
          Hello
        </Text.Text>,
      );
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("span");
    });
  });

  describe("overflow", () => {
    it("should not add a overflow classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("pluto-text--overflow");
    });

    it("should add a overflow classname if the overflow is set to ellipsis", () => {
      const c = render(<Text.Text overflow="ellipsis">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto-text--overflow-ellipsis");
    });

    it("should add a overflow classname if the overflow is set to clip", () => {
      const c = render(<Text.Text overflow="clip">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto-text--overflow-clip");
    });

    it("should add a overflow classname if the overflow is set to nowrap", () => {
      const c = render(<Text.Text overflow="nowrap">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto-text--overflow-nowrap");
    });

    it("should add a overflow classname if the overflow is set to clip", () => {
      const c = render(<Text.Text overflow="clip">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto-text--overflow-clip");
    });
  });

  describe("square", () => {
    it("should not add a square classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("pluto-text--square");
    });

    it("should add a square classname if the square is true", () => {
      const c = render(<Text.Text square>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--square");
    });

    it("should add a square classname if the child is a single string letter", () => {
      const c = render(<Text.Text>A</Text.Text>);
      expect(c.getByText("A").className).toContain("pluto--square");
    });

    it("should add a square classname if the el an icon", () => {
      const c = render(
        <Text.Text>
          <Icon.Acquire aria-label="Acquire" />
        </Text.Text>,
      );
      const el = c.getByLabelText("Acquire").parentElement;
      expect(el).toBeTruthy();
      expect(el?.className).toContain("pluto--square");
    });
  });

  describe("status", () => {
    it("should not add a status classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("pluto--status-error");
      expect(c.getByText("Hello").className).not.toContain("pluto--status-warning");
      expect(c.getByText("Hello").className).not.toContain("pluto--status-success");
    });

    it("should add a status classname if the status is set to error", () => {
      const c = render(<Text.Text status="error">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--status-error");
    });

    it("should add a status classname if the status is set to warning", () => {
      const c = render(<Text.Text status="warning">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--status-warning");
    });

    it("should add a status classname if the status is set to success", () => {
      const c = render(<Text.Text status="success">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--status-success");
    });

    it("should add a status classname if the status is set to loading", () => {
      const c = render(<Text.Text status="loading">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto--status-loading");
    });
  });

  describe("lineClamp", () => {
    it("should not add a line-clamp classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("pluto-text--line-clamp");
    });

    it("should add a line-clamp classname when lineClamp is set", () => {
      const c = render(<Text.Text lineClamp={2}>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("pluto-text--line-clamp");
    });

    it("should set the WebkitLineClamp style to the provided value", () => {
      const c = render(<Text.Text lineClamp={3}>Hello</Text.Text>);
      expect(c.getByText("Hello").style.webkitLineClamp).toBe("3");
    });

    it("should work with different lineClamp values", () => {
      const c = render(<Text.Text lineClamp={5}>Hello</Text.Text>);
      expect(c.getByText("Hello").style.webkitLineClamp).toBe("5");
    });
  });

  describe("Editable", () => {
    it("should focus and select the text when double clicked", () => {
      const c = render(
        <Text.Editable level="h1" value="Hello" onChange={vitest.fn()} />,
      );
      const text = c.getByText("Hello");
      fireEvent.dblClick(text);
      expect(document.activeElement).toBe(text);
      expect(window.getSelection()?.toString()).toBe("Hello");
    });
  });
});
