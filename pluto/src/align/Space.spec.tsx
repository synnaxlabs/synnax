// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Align } from "@/align";

describe("Space", () => {
  it("should render items with a space between them", () => {
    const c = render(
      <Align.Space size="small">
        <div>Hello</div>
        <div>World</div>
      </Align.Space>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    const parent = world.parentElement;
    expect(parent?.classList.toString()).toContain("small");
  });
  it("should render items with no gap", () => {
    const c = render(
      <Align.Space empty>
        <div>Hello</div>
        <div>World</div>
      </Align.Space>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    const parent = world.parentElement;
    expect(parent?.style.gap).toBe("0");
  });
  it("should render items with a multiple of the base size", () => {
    const c = render(
      <Align.Space size={2}>
        <div>Hello</div>
        <div>World</div>
      </Align.Space>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
    const world = c.getByText("World");
    expect(world).toBeTruthy();
    expect(world.parentElement?.style.gap).toBe("2rem");
  });
});
