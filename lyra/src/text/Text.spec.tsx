// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { text } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vitest } from "vitest";

import { Icon } from "@/icon";
import { Text } from "@/text";

describe("Text", () => {
  describe("levels", () => {
    text.LEVELS.filter((level) => level !== "tiny").forEach((level) => {
      it(`should render text with the correct HTML tag for ${level}`, () => {
        const c = render(<Text.Text level={level}>Hello</Text.Text>);
        expect(c.getByText("Hello").tagName.toLowerCase()).toBe(level);
      });
    });

    it("should render tiny text in a small element", () => {
      const c = render(<Text.Text level="tiny">Hello</Text.Text>);
      expect(c.getByText("Hello").tagName.toLowerCase()).toBe("small");
    });
  });

  describe("gap", () => {
    it("should render the element with a small gap by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--gap-small");
    });

    it("should render the element with a different gap", () => {
      const c = render(<Text.Text gap="medium">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--gap-medium");
    });
  });

  describe("direction", () => {
    it("should render the element with a direction of x by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--direction-x");
    });

    it("should render the element with a different direction", () => {
      const c = render(<Text.Text direction="y">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--direction-y");
    });

    it("should render the element with a different direction", () => {
      const c = render(<Text.Text y>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--direction-y");
    });
  });

  describe("variants", () => {
    it("should not add a variant classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("lyra-text--link");
      expect(c.getByText("Hello").className).not.toContain("lyra-text--code");
      expect(c.getByText("Hello").className).not.toContain("lyra-text--keyboard");
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
        expect(c.getByText("Hello").className).toContain("lyra-text--code");
      });
    });

    describe("keyboard", () => {
      it("should render a keyboard key if the variant is set to keyboard", () => {
        const c = render(<Text.Text variant="keyboard">Hello</Text.Text>);
        expect(c.getByText("Hello").className).toContain("lyra-text--keyboard");
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
      expect(c.getByText("Hello").className).not.toContain("lyra-text--overflow");
    });

    it("should add a overflow classname if the overflow is set to fade", () => {
      const c = render(<Text.Text overflow="fade">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra-text--overflow-fade");
    });

    it("should add a overflow classname if the overflow is set to clip", () => {
      const c = render(<Text.Text overflow="clip">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra-text--overflow-clip");
    });

    it("should add a overflow classname if the overflow is set to nowrap", () => {
      const c = render(<Text.Text overflow="nowrap">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra-text--overflow-nowrap");
    });

    it("should add a overflow classname if the overflow is set to clip", () => {
      const c = render(<Text.Text overflow="clip">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra-text--overflow-clip");
    });
  });

  describe("square", () => {
    it("should not add a square classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("lyra-text--square");
    });

    it("should add a square classname if the square is true", () => {
      const c = render(<Text.Text square>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--square");
    });

    it("should add a square classname if the child is a single string letter", () => {
      const c = render(<Text.Text>A</Text.Text>);
      expect(c.getByText("A").className).toContain("lyra--square");
    });

    it("should add a square classname if the el an icon", () => {
      const c = render(
        <Text.Text>
          <Icon.Acquire aria-label="Acquire" />
        </Text.Text>,
      );
      const el = c.getByLabelText("Acquire").parentElement;
      expect(el).toBeTruthy();
      expect(el?.className).toContain("lyra--square");
    });

    it("should treat an icon with a trailing space as square", () => {
      const c = render(
        <Text.Text>
          <Icon.Acquire aria-label="Acquire" /> {false}
        </Text.Text>,
      );
      const el = c.getByLabelText("Acquire").parentElement;
      expect(el).toBeTruthy();
      expect(el?.className).toContain("lyra--square");
    });
  });

  describe("status", () => {
    it("should not add a status classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("lyra--status-error");
      expect(c.getByText("Hello").className).not.toContain("lyra--status-warning");
      expect(c.getByText("Hello").className).not.toContain("lyra--status-success");
    });

    it("should add a status classname if the status is set to error", () => {
      const c = render(<Text.Text status="error">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--status-error");
    });

    it("should add a status classname if the status is set to warning", () => {
      const c = render(<Text.Text status="warning">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--status-warning");
    });

    it("should add a status classname if the status is set to success", () => {
      const c = render(<Text.Text status="success">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--status-success");
    });

    it("should add a status classname if the status is set to loading", () => {
      const c = render(<Text.Text status="loading">Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra--status-loading");
    });
  });

  describe("lineClamp", () => {
    it("should not add a line-clamp classname by default", () => {
      const c = render(<Text.Text>Hello</Text.Text>);
      expect(c.getByText("Hello").className).not.toContain("lyra-text--line-clamp");
    });

    it("should add a line-clamp classname when lineClamp is set", () => {
      const c = render(<Text.Text lineClamp={2}>Hello</Text.Text>);
      expect(c.getByText("Hello").className).toContain("lyra-text--line-clamp");
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

  describe("highlight", () => {
    const marks = (el: HTMLElement): string[] =>
      Array.from(el.querySelectorAll("mark")).map((m) => m.textContent);

    it("should tint the start of each word a token begins", () => {
      const c = render(
        <Text.Text highlight="pre ox">pressure_sensor oxygen.press</Text.Text>,
      );
      const el = c.getByText((_, node) => node?.tagName === "P");
      expect(marks(el)).toEqual(["pre", "ox", "pre"]);
      expect(el.textContent).toBe("pressure_sensor oxygen.press");
    });

    it("should tint a separator between two whole tinted words", () => {
      const c = render(
        <Text.Text highlight="press_pt_0">press_pt_02 pt_x_pt</Text.Text>,
      );
      const el = c.getByText((_, node) => node?.tagName === "P");
      expect(marks(el)).toEqual(["press_pt_0", "pt", "pt"]);
      expect(el.textContent).toBe("press_pt_02 pt_x_pt");
    });

    it("should not tint a separator after a partly tinted word", () => {
      const c = render(<Text.Text highlight="pre sen">pressure_sensor</Text.Text>);
      const el = c.getByText((_, node) => node?.tagName === "P");
      expect(marks(el)).toEqual(["pre", "sen"]);
    });

    it("should match without regard to case", () => {
      const c = render(<Text.Text highlight="PRESS">Pressure</Text.Text>);
      expect(marks(c.getByText((_, node) => node?.tagName === "P"))).toEqual(["Press"]);
    });

    it("should tint the longest token that begins a word", () => {
      const c = render(<Text.Text highlight="p pres">pressure</Text.Text>);
      expect(marks(c.getByText((_, node) => node?.tagName === "P"))).toEqual(["pres"]);
    });

    it("should not tint a token inside a word", () => {
      const c = render(<Text.Text highlight="sure">pressure</Text.Text>);
      const el = c.getByText("pressure");
      expect(el.querySelector("mark")).toBeNull();
    });

    it("should render the children unchanged when nothing matches", () => {
      const c = render(<Text.Text highlight="valve">pressure</Text.Text>);
      expect(c.getByText("pressure").childNodes).toHaveLength(1);
    });

    it("should leave children that are not a string alone", () => {
      const c = render(
        <Text.Text highlight="press">
          <span>pressure</span>
        </Text.Text>,
      );
      expect(c.getByText("pressure").querySelector("mark")).toBeNull();
    });

    describe("Editable", () => {
      const editable = (c: ReturnType<typeof render>): HTMLElement =>
        c.getByText((_, node) => node?.tagName === "P");

      it("should tint the value while not editing", () => {
        const c = render(
          <Text.Editable value="pressure" highlight="pre" onChange={vitest.fn()} />,
        );
        expect(marks(editable(c))).toEqual(["pre"]);
      });

      it("should edit plain text and commit it", () => {
        const onChange = vitest.fn();
        const c = render(
          <Text.Editable value="pressure" highlight="pre" onChange={onChange} />,
        );
        const el = editable(c);
        fireEvent.dblClick(el);
        expect(el.querySelector("mark")).toBeNull();
        expect(window.getSelection()?.toString()).toBe("pressure");
        el.innerText = "preheat";
        fireEvent.keyDown(el, { key: "Enter" });
        expect(onChange).toHaveBeenCalledWith("preheat");
        expect(marks(el)).toEqual(["pre"]);
        expect(el.textContent).toBe("preheat");
      });

      it("should tint again after an edit is cancelled", () => {
        const c = render(
          <Text.Editable value="pressure" highlight="pre" onChange={vitest.fn()} />,
        );
        const el = editable(c);
        fireEvent.dblClick(el);
        el.innerText = "pressed";
        fireEvent.keyDown(el, { key: "Escape" });
        expect(marks(el)).toEqual(["pre"]);
        expect(el.textContent).toBe("pressure");
      });

      it("should follow a change to the term", () => {
        const c = render(
          <Text.Editable value="pressure" highlight="pre" onChange={vitest.fn()} />,
        );
        c.rerender(
          <Text.Editable value="pressure" highlight="press" onChange={vitest.fn()} />,
        );
        expect(marks(editable(c))).toEqual(["press"]);
      });
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
