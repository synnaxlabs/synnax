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

  it("should pass logo props through to the logo", () => {
    const c = render(<Logo.Watermark aria-label="Synnax" />);
    expect(c.getByLabelText("Synnax").tagName).toEqual("svg");
  });
});
