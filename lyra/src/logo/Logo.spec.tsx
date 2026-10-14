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

import { Logo } from "@/logo";

describe("Logo", () => {
  it("should render the icon without the name by default", () => {
    const c = render(<Logo.Logo />);
    expect(c.container.querySelector("svg")).not.toBeNull();
    expect(c.container.textContent).toEqual("");
  });

  it("should render the name in the title variant", () => {
    const c = render(<Logo.Logo variant="title" />);
    expect(c.container.querySelector("svg text")?.textContent).toEqual("Synnax");
  });

  it("should pass props through to the SVG", () => {
    const c = render(<Logo.Logo aria-label="Synnax" />);
    expect(c.getByLabelText("Synnax").tagName).toEqual("svg");
  });

  describe("Watermark", () => {
    it("should render the logo beside its children", () => {
      const c = render(
        <Logo.Watermark variant="title">
          <span>Empty</span>
        </Logo.Watermark>,
      );
      expect(c.container.querySelector("svg text")?.textContent).toEqual("Synnax");
      expect(c.getByText("Empty")).toBeTruthy();
    });
  });
});
