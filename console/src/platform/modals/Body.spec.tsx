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

import { Modals } from "@/platform/modals";
import { Wrapper } from "@/platform/modals/testutil";

describe("Body", () => {
  it("should render children inside the modal body element and forward className", () => {
    const { baseElement } = render(
      <Modals.Body className="extra">body content</Modals.Body>,
      { wrapper: Wrapper },
    );
    const el = baseElement.querySelector(".console-modal__body");
    expect(el).not.toBeNull();
    expect(el?.className).toContain("extra");
    expect(screen.getByText("body content")).toBeTruthy();
  });
});
