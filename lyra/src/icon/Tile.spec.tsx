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

import { Icon } from "@/icon";

describe("Icon.Tile", () => {
  it("should frame its icon and hide the frame from assistive technology", () => {
    const { container } = render(
      <Icon.Tile>
        <Icon.Rack />
      </Icon.Tile>,
    );
    const tile = container.querySelector(".lyra-icon-tile");
    expect(tile?.getAttribute("aria-hidden")).toBe("true");
    expect(tile?.querySelector("svg")).not.toBeNull();
  });

  it.each(["small", "large"] as const)("should mark the %s size", (size) => {
    const { container } = render(
      <Icon.Tile size={size}>
        <Icon.Rack />
      </Icon.Tile>,
    );
    expect(container.querySelector(".lyra-icon-tile")?.classList).toContain(
      `lyra-icon-tile--${size}`,
    );
  });

  it("should default to the large size", () => {
    const { container } = render(
      <Icon.Tile>
        <Icon.Rack />
      </Icon.Tile>,
    );
    expect(container.querySelector(".lyra-icon-tile")?.classList).toContain(
      "lyra-icon-tile--large",
    );
  });
});
