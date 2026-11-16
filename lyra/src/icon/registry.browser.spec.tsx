// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { renderThemed } from "@/styletest";

const middle = (el: Element): number => {
  const { top, bottom } = el.getBoundingClientRect();
  return (top + bottom) / 2;
};

describe("Icon registry", () => {
  it("should move each star up to the center of its ink", async () => {
    const { container } = await renderThemed(
      <Flex.Box x align="center">
        <Icon.StarFilled />
        <Icon.StarOutlined />
        <Icon.Search />
      </Flex.Box>,
    );
    const [filled, outlined, search] = Array.from(container.querySelectorAll("svg"));
    // The ink of a star sits 3.6% of its box below the middle of the box.
    const offset = 0.036 * filled.getBoundingClientRect().height;
    expect(middle(search) - middle(filled)).toBeCloseTo(offset, 1);
    expect(middle(search) - middle(outlined)).toBeCloseTo(offset, 1);
  });
});
