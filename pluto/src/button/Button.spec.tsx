// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vitest } from "vitest";

import { Button } from "@/button";

describe("Button", () => {
  describe("Basic Rendering", () => {
    it("should render a button with the provided text", () => {
      const c = render(<Button.Button size="small">Hello</Button.Button>);
      expect(c.getByText("Hello")).toBeTruthy();
    });
  });

  describe("Variants", () => {
    it("shoulud add a filled class to the button when the variant is filled", () => {
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

  // describe("Icon", () => {
  //   it("should render a button with the provided icon", () => {
  //     const c = render(
  //       <Button.Button size="small">
  //         <svg aria-label="icon" />
  //       </Button.Button>,
  //     );
  //     expect(c.getByLabelText("icon")).toBeTruthy();
  //   });
  // });
  // describe("Link", () => {
  //   it("should render a link with the provided text", () => {
  //     const c = render(<Button.Button size="small">Hello</Button.Button>);
  //     expect(c.getByText("Hello")).toBeTruthy();
  //   });
  // });
  // describe("Toggle", () => {
  //   it("should a button that can be toggled", async () => {
  //     const onChange = vitest.fn();
  //     const ToggleTest = (): ReactElement => {
  //       const [value, setValue] = useState(false);
  //       return (
  //         <Button.Toggle
  //           size="small"
  //           value={value}
  //           onChange={() => {
  //             onChange();
  //             setValue(!value);
  //           }}
  //         >
  //           Hello
  //         </Button.Toggle>
  //       );
  //     };
  //     const c = render(<ToggleTest />);
  //     const label = c.getByText("Hello");
  //     expect(label).toBeTruthy();
  //     const button = label.parentElement as HTMLElement;
  //     expect(button).toBeTruthy();
  //     expect(button.className).not.toContain("filled");
  //     await act(async () => {
  //       await userEvent.click(label);
  //     });
  //     expect(onChange).toHaveBeenCalled();
  //     expect(button.className).toContain("filled");
  //   });
  // });
});
