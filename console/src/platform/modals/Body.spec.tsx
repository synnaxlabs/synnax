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

import { Body } from "@/platform/modals/Body";
import { Wrapper } from "@/platform/modals/testutil";

describe("Body", () => {
  it("should render its children", () => {
    render(<Body>body content</Body>, { wrapper: Wrapper });
    expect(screen.getByText("body content")).toBeTruthy();
  });

  it("should apply the modal body element class", () => {
    const { baseElement } = render(<Body>body content</Body>, { wrapper: Wrapper });
    expect(baseElement.querySelector(".console-modal__body")).not.toBeNull();
  });

  it("should forward a custom className", () => {
    const { baseElement } = render(<Body className="extra">body content</Body>, {
      wrapper: Wrapper,
    });
    expect(baseElement.querySelector(".console-modal__body")?.className).toContain(
      "extra",
    );
  });
});
