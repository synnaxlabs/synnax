// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Flex } from "@/flex";

describe("Box", () => {
  it("should render a basic flex box", () => {
    const c = render(<Flex.Box>Hello</Flex.Box>);
    const el = c.getByText("Hello");
    expect(el).toBeTruthy();
    expect(el.className).toContain("pluto-flex");
  });

  describe("direction", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--direction");
    });

    it("should allow the caller change the direction via the direction prop", () => {
      const c = render(<Flex.Box direction="x">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-x");
    });

    it("should allow the caller to set the direction via the x prop", () => {
      const c = render(<Flex.Box x>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-x");
    });

    it("should allow the caller to set the direction via the y prop", () => {
      const c = render(<Flex.Box y>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-y");
    });

    it("should prefer the x prop over the direction prop", () => {
      const c = render(
        <Flex.Box direction="y" x>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-x");
    });

    it("should prefer the y prop over the direction prop", () => {
      const c = render(
        <Flex.Box direction="x" y>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-y");
    });

    it("should prefer the x prop over the y prop", () => {
      const c = render(
        <Flex.Box direction="y" x>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--direction-x");
    });

    it("should reverse the direction if the reverse prop is true", () => {
      const c = render(
        <Flex.Box direction="x" reverse>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--reverse");
    });

    it("should reverse the direction if the direction is right", () => {
      const c = render(<Flex.Box direction="right">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--reverse");
    });

    it("should reverse the direction if the direction is bottom", () => {
      const c = render(<Flex.Box direction="bottom">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--reverse");
    });
  });

  describe("justify", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el).toBeTruthy();
    });

    it("should set center if the justify is center", () => {
      const c = render(<Flex.Box justify="center">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-center");
    });

    it("should set end if the justify is end", () => {
      const c = render(<Flex.Box justify="end">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-end");
    });

    it("should set start if the justify is start", () => {
      const c = render(<Flex.Box justify="start">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-start");
    });

    it("should set between if the justify is between", () => {
      const c = render(<Flex.Box justify="between">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-between");
    });

    it("should set around if the justify is around", () => {
      const c = render(<Flex.Box justify="around">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-around");
    });

    it("should set evenly if the justify is evenly", () => {
      const c = render(<Flex.Box justify="evenly">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--justify-evenly");
    });
  });

  describe("align", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--align");
    });

    it("should set center if the align is center", () => {
      const c = render(<Flex.Box align="center">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--align-center");
    });

    it("should set end if the align is end", () => {
      const c = render(<Flex.Box align="end">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--align-end");
    });

    it("should set start if the align is start", () => {
      const c = render(<Flex.Box align="start">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--align-start");
    });

    it("should set stretch if the align is stretch", () => {
      const c = render(<Flex.Box align="stretch">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--align-stretch");
    });
  });

  describe("grow", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--grow");
    });

    it("should set grow if the grow is true", () => {
      const c = render(<Flex.Box grow>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--grow");
    });

    describe("when the grow is a number", () => {
      it("should set the flex-grow style to the number", () => {
        const c = render(<Flex.Box grow={2}>Hello</Flex.Box>);
        const el = c.getByText("Hello");
        expect(el.style.flexGrow).toBe("2");
      });
    });
  });

  describe("shrink", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--shrink");
    });

    it("should set shrink if the shrink is true", () => {
      const c = render(<Flex.Box shrink>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--shrink");
    });

    describe("when the shrink is a number", () => {
      it("should set the flex-shrink style to the number", () => {
        const c = render(<Flex.Box shrink={2}>Hello</Flex.Box>);
        const el = c.getByText("Hello");
        expect(el.style.flexShrink).toBe("2");
      });
    });
  });

  describe("sharp", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--sharp");
    });

    it("should set sharp if the sharp is true", () => {
      const c = render(<Flex.Box sharp>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--sharp");
    });
  });

  describe("rounded", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--rounded");
    });

    it("should set rounded if the rounded is true", () => {
      const c = render(<Flex.Box rounded>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--rounded");
    });

    it("should set the border radius directly if the rounded is a number", () => {
      const c = render(<Flex.Box rounded={2}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderRadius).toBe("2rem");
      expect(el.className).not.toContain("pluto--rounded");
    });

    it("should not set rounded if rounded is set to false", () => {
      const c = render(<Flex.Box rounded={false}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--rounded");
    });
  });

  describe("empty", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--empty");
    });

    it("should set empty if the empty is true", () => {
      const c = render(<Flex.Box empty>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--empty");
    });
  });

  describe("gap", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--gap");
    });

    it("should set gap if the gap is a number", () => {
      const c = render(<Flex.Box gap={2}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.gap).toBe("2rem");
    });

    it("should set gap if the gap is small", () => {
      const c = render(<Flex.Box gap="small">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--gap-small");
    });

    it("should set gap if the gap is medium", () => {
      const c = render(<Flex.Box gap="medium">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--gap-medium");
    });

    it("should set gap if the gap is large", () => {
      const c = render(<Flex.Box gap="large">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--gap-large");
    });

    it("should set gap if the gap is huge", () => {
      const c = render(<Flex.Box gap="huge">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--gap-huge");
    });

    it("should set gap if the gap is tiny", () => {
      const c = render(<Flex.Box gap="tiny">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--gap-tiny");
    });
  });

  describe("full", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--full");
    });

    it("should set full if the full is true", () => {
      const c = render(<Flex.Box full>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--full");
    });

    it("should set full if the full is x", () => {
      const c = render(<Flex.Box full="x">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--full-x");
    });

    it("should set full if the full is y", () => {
      const c = render(<Flex.Box full="y">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--full-y");
    });
  });

  describe("wrap", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--wrap");
    });

    it("should set wrap if the wrap is true", () => {
      const c = render(<Flex.Box wrap>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--wrap");
    });
  });

  describe("size", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--height");
    });

    it("should set size if the size is small", () => {
      const c = render(<Flex.Box size="small">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--height-small");
    });

    it("should set size if the size is medium", () => {
      const c = render(<Flex.Box size="medium">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--height-medium");
    });

    it("should set size if the size is large", () => {
      const c = render(<Flex.Box size="large">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--height-large");
    });

    it("should set size if the size is huge", () => {
      const c = render(<Flex.Box size="huge">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--height-huge");
    });

    it("should set size if the size is tiny", () => {
      const c = render(<Flex.Box size="tiny">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--height-tiny");
    });
  });

  describe("square", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--square");
    });

    it("should set square if the square is true", () => {
      const c = render(<Flex.Box square>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--square");
    });
  });

  describe("pack", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--pack");
    });

    it("should set pack if the pack is true", () => {
      const c = render(<Flex.Box pack>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--pack");
    });
  });

  describe("color", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--color");
    });

    it("should set a color classname if the color is a shade", () => {
      const c = render(<Flex.Box color={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--color-11");
    });

    it("should set the color directly if the color is a color", () => {
      const c = render(<Flex.Box color="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.color).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("pluto--color");
    });
  });

  describe("background", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--bg");
    });

    it("should set a background classname if the background is a shade", () => {
      const c = render(<Flex.Box background={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--bg-11");
    });

    it("should set the background directly if the background is a color", () => {
      const c = render(<Flex.Box background="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.backgroundColor).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("pluto--background");
    });
  });

  describe("borderColor", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("pluto--border-color");
    });

    it("should set a border color classname if the border color is a shade", () => {
      const c = render(<Flex.Box borderColor={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("pluto--border-color-11");
    });

    it("should set the border color directly if the border color is a color", () => {
      const c = render(<Flex.Box borderColor="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderColor).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("pluto--border-color");
    });
  });
});
