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
    expect(el.className).toContain("lyra-flex");
  });

  it("should merge a caller-provided className with the generated classnames", () => {
    const c = render(<Flex.Box className="custom-class">Hello</Flex.Box>);
    const el = c.getByText("Hello");
    expect(el.className).toContain("custom-class");
    expect(el.className).toContain("lyra-flex");
  });

  it("should merge a caller-provided style with the computed style", () => {
    const c = render(<Flex.Box style={{ margin: "1px" }}>Hello</Flex.Box>);
    const el = c.getByText("Hello");
    expect(el.style.margin).toBe("1px");
  });

  describe("direction", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--direction");
    });

    it("should allow the caller change the direction via the direction prop", () => {
      const c = render(<Flex.Box direction="x">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should allow the caller to set the direction via the x prop", () => {
      const c = render(<Flex.Box x>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should allow the caller to set the direction via the y prop", () => {
      const c = render(<Flex.Box y>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-y");
    });

    it("should prefer the x prop over the direction prop", () => {
      const c = render(
        <Flex.Box direction="y" x>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should prefer the y prop over the direction prop", () => {
      const c = render(
        <Flex.Box direction="x" y>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-y");
    });

    it("should prefer the x prop over the y prop", () => {
      const c = render(
        <Flex.Box direction="y" x>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should reverse the direction if the reverse prop is true", () => {
      const c = render(
        <Flex.Box direction="x" reverse>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--reverse");
    });

    it("should reverse the direction if the direction is right", () => {
      const c = render(<Flex.Box direction="right">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--reverse");
    });

    it("should reverse the direction if the direction is bottom", () => {
      const c = render(<Flex.Box direction="bottom">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--reverse");
    });

    it("should not reverse if reverse is explicitly false, even if the direction is right", () => {
      const c = render(
        <Flex.Box direction="right" reverse={false}>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--reverse");
    });

    it("should set direction-x if the direction is left", () => {
      const c = render(<Flex.Box direction="left">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should set direction-y if the direction is top", () => {
      const c = render(<Flex.Box direction="top">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-y");
    });

    it("should set direction-x if the direction is right", () => {
      const c = render(<Flex.Box direction="right">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
    });

    it("should set direction-y if the direction is bottom", () => {
      const c = render(<Flex.Box direction="bottom">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-y");
    });

    it("should default the direction to x when pack is true and no direction is given", () => {
      const c = render(<Flex.Box pack>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--direction-x");
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
      expect(el.className).toContain("lyra--justify-center");
    });

    it("should set end if the justify is end", () => {
      const c = render(<Flex.Box justify="end">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--justify-end");
    });

    it("should set start if the justify is start", () => {
      const c = render(<Flex.Box justify="start">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--justify-start");
    });

    it("should set between if the justify is between", () => {
      const c = render(<Flex.Box justify="between">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--justify-between");
    });

    it("should set around if the justify is around", () => {
      const c = render(<Flex.Box justify="around">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--justify-around");
    });

    it("should set evenly if the justify is evenly", () => {
      const c = render(<Flex.Box justify="evenly">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--justify-evenly");
    });
  });

  describe("align", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--align");
    });

    it("should set center if the align is center", () => {
      const c = render(<Flex.Box align="center">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-center");
    });

    it("should set end if the align is end", () => {
      const c = render(<Flex.Box align="end">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-end");
    });

    it("should set start if the align is start", () => {
      const c = render(<Flex.Box align="start">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-start");
    });

    it("should set stretch if the align is stretch", () => {
      const c = render(<Flex.Box align="stretch">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-stretch");
    });
  });

  describe("alignSelf", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--align-self");
    });

    it("should set center if the alignSelf is center", () => {
      const c = render(<Flex.Box alignSelf="center">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-self-center");
    });

    it("should set end if the alignSelf is end", () => {
      const c = render(<Flex.Box alignSelf="end">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-self-end");
    });

    it("should set start if the alignSelf is start", () => {
      const c = render(<Flex.Box alignSelf="start">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-self-start");
    });

    it("should set stretch if the alignSelf is stretch", () => {
      const c = render(<Flex.Box alignSelf="stretch">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--align-self-stretch");
    });
  });

  describe("grow", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--grow");
    });

    it("should set grow if the grow is true", () => {
      const c = render(<Flex.Box grow>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--grow");
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
      expect(el.className).not.toContain("lyra--shrink");
    });

    it("should set shrink if the shrink is true", () => {
      const c = render(<Flex.Box shrink>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--shrink");
    });

    describe("when the shrink is a number", () => {
      it("should set the flex-shrink style to the number", () => {
        const c = render(<Flex.Box shrink={2}>Hello</Flex.Box>);
        const el = c.getByText("Hello");
        expect(el.style.flexShrink).toBe("2");
      });
    });
  });

  describe("bordered", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--bordered");
    });

    it("should set bordered if the bordered is true", () => {
      const c = render(<Flex.Box bordered>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--bordered");
    });
  });

  describe("borderWidth", () => {
    it("should not set a border width style by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderWidth).toBe("");
    });

    it("should set the border width style if provided", () => {
      const c = render(<Flex.Box borderWidth={3}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderWidth).toBe("3px");
    });
  });

  describe("center", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--center");
    });

    it("should set center if the center is true", () => {
      const c = render(<Flex.Box center>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--center");
    });
  });

  describe("sharp", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--sharp");
    });

    it("should set sharp if the sharp is true", () => {
      const c = render(<Flex.Box sharp>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--sharp");
    });
  });

  describe("rounded", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--rounded");
    });

    it("should set rounded if the rounded is true", () => {
      const c = render(<Flex.Box rounded>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--rounded");
    });

    it("should set the border radius directly if the rounded is a number", () => {
      const c = render(<Flex.Box rounded={2}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderRadius).toBe("2rem");
      expect(el.className).not.toContain("lyra--rounded");
    });

    it("should not set rounded if rounded is set to false", () => {
      const c = render(<Flex.Box rounded={false}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--rounded");
    });

    it("should add a size-step class if rounded is a Component.Size", () => {
      const c = render(<Flex.Box rounded="large">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--rounded-large");
      expect(el.style.borderRadius).toBe("");
    });

    it("should round by default when packed", () => {
      const c = render(<Flex.Box pack>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.classList).toContain("lyra--rounded");
    });

    it("should let an explicit rounded override the packed default", () => {
      const c = render(
        <Flex.Box pack rounded="large">
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.classList).toContain("lyra--rounded-large");
      expect(el.classList).not.toContain("lyra--rounded");
    });

    it("should not round a packed box when rounded is false", () => {
      const c = render(
        <Flex.Box pack rounded={false}>
          Hello
        </Flex.Box>,
      );
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--rounded");
    });
  });

  describe("empty", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--empty");
    });

    it("should set empty if the empty is true", () => {
      const c = render(<Flex.Box empty>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--empty");
    });
  });

  describe("gap", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--gap");
    });

    it("should set gap if the gap is a number", () => {
      const c = render(<Flex.Box gap={2}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.gap).toBe("2rem");
    });

    it("should set gap if the gap is small", () => {
      const c = render(<Flex.Box gap="small">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--gap-small");
    });

    it("should set gap if the gap is medium", () => {
      const c = render(<Flex.Box gap="medium">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--gap-medium");
    });

    it("should set gap if the gap is large", () => {
      const c = render(<Flex.Box gap="large">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--gap-large");
    });

    it("should set gap if the gap is huge", () => {
      const c = render(<Flex.Box gap="huge">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--gap-huge");
    });

    it("should set gap if the gap is tiny", () => {
      const c = render(<Flex.Box gap="tiny">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--gap-tiny");
    });
  });

  describe("full", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--full");
    });

    it("should set full if the full is true", () => {
      const c = render(<Flex.Box full>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--full");
    });

    it("should set full if the full is x", () => {
      const c = render(<Flex.Box full="x">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--full-x");
    });

    it("should set full if the full is y", () => {
      const c = render(<Flex.Box full="y">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--full-y");
    });
  });

  describe("wrap", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--wrap");
    });

    it("should set wrap if the wrap is true", () => {
      const c = render(<Flex.Box wrap>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--wrap");
    });
  });

  describe("size", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--height");
    });

    it("should set size if the size is small", () => {
      const c = render(<Flex.Box size="small">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--height-small");
    });

    it("should set size if the size is medium", () => {
      const c = render(<Flex.Box size="medium">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--height-medium");
    });

    it("should set size if the size is large", () => {
      const c = render(<Flex.Box size="large">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--height-large");
    });

    it("should set size if the size is huge", () => {
      const c = render(<Flex.Box size="huge">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--height-huge");
    });

    it("should set size if the size is tiny", () => {
      const c = render(<Flex.Box size="tiny">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--height-tiny");
    });
  });

  describe("square", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--square");
    });

    it("should set square if the square is true", () => {
      const c = render(<Flex.Box square>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--square");
    });
  });

  describe("pack", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--pack");
    });

    it("should set pack if the pack is true", () => {
      const c = render(<Flex.Box pack>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--pack");
    });
  });

  describe("color", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--color");
    });

    it("should set a color classname if the color is a shade", () => {
      const c = render(<Flex.Box color={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--color-11");
    });

    it("should set the color directly if the color is a color", () => {
      const c = render(<Flex.Box color="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.color).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("lyra--color");
    });

    it("should not add a classname or style if the color is false", () => {
      const c = render(<Flex.Box color={false}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--color");
      expect(el.style.color).toBe("");
    });
  });

  describe("background", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--bg");
    });

    it("should set a background classname if the background is a shade", () => {
      const c = render(<Flex.Box background={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--bg-11");
    });

    it("should set the background directly if the background is a color", () => {
      const c = render(<Flex.Box background="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.backgroundColor).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("lyra--background");
    });

    it("should not add a classname or style if the background is false", () => {
      const c = render(<Flex.Box background={false}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--bg");
      expect(el.style.backgroundColor).toBe("");
    });
  });

  describe("borderColor", () => {
    it("should not add a classname by default", () => {
      const c = render(<Flex.Box>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--border-color");
    });

    it("should set a border color classname if the border color is a shade", () => {
      const c = render(<Flex.Box borderColor={11}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).toContain("lyra--border-color-11");
    });

    it("should set the border color directly if the border color is a color", () => {
      const c = render(<Flex.Box borderColor="#000000">Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.style.borderColor).toBe("rgb(0, 0, 0)");
      expect(el.className).not.toContain("lyra--border-color");
    });

    it("should not add a classname or style if the border color is false", () => {
      const c = render(<Flex.Box borderColor={false}>Hello</Flex.Box>);
      const el = c.getByText("Hello");
      expect(el.className).not.toContain("lyra--border-color");
      expect(el.style.borderColor).toBe("");
    });
  });
});
