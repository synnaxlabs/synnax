// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { Description } from "@/description";
import { Icon } from "@/icon";
import { renderThemed } from "@/styletest";

const centerY = (el: Element): number => {
  const { top, height } = el.getBoundingClientRect();
  return top + height / 2;
};

describe("Description", () => {
  it("should center a value on a label that starts with an icon", async () => {
    await renderThemed(
      <Description.List>
        <Description.Item>
          <Description.Label>
            <Icon.Time />
            Last sample
          </Description.Label>
          <Description.Value>3s ago</Description.Value>
        </Description.Item>
      </Description.List>,
    );
    const label = page.getByRole("term").element();
    const value = page.getByRole("definition").element();
    expect(Math.abs(centerY(label) - centerY(value))).toBeLessThan(1);
  });
});
