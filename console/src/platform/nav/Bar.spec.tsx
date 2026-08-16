// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Nav } from "@/platform/nav";
import { renderWithConsole } from "@/testutil";

describe("nav Bar", () => {
  it("should render its children and merge the passed className", async () => {
    const { container } = await renderWithConsole(
      <Nav.Bar className="extra-class">
        <span>nav content</span>
      </Nav.Bar>,
    );
    expect(screen.getByText("nav content")).toBeTruthy();
    const bar = container.querySelector(".console-nav__bar");
    expect(bar).not.toBeNull();
    expect(bar?.className).toContain("extra-class");
  });
});
