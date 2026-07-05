// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Footer } from "@/layered/service/modals/Footer";
import { Wrapper } from "@/layered/service/modals/testutil";

describe("Footer", () => {
  it("should render its children", () => {
    render(<Footer>footer content</Footer>, { wrapper: Wrapper });
    expect(screen.getByText("footer content")).toBeTruthy();
  });

  it("should apply the modal footer element class", () => {
    const { baseElement } = render(<Footer>footer content</Footer>, {
      wrapper: Wrapper,
    });
    expect(baseElement.querySelector(".console-modal__footer")).not.toBeNull();
  });

  it("should forward a custom className", () => {
    const { baseElement } = render(<Footer className="extra">footer content</Footer>, {
      wrapper: Wrapper,
    });
    expect(baseElement.querySelector(".console-modal__footer")?.className).toContain(
      "extra",
    );
  });
});
