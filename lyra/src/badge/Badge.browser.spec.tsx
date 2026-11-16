// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Badge } from "@/badge";
import { Input } from "@/input";
import { computed, renderThemed, resolve } from "@/styletest";

const expectColors = (el: Element, text: string, fill: string): void => {
  expect(computed(el, "color")).toBe(resolve(`var(${text})`));
  expect(computed(el, "background-color")).toBe(resolve(`var(${fill})`));
};

describe("Badge", () => {
  it("should be gray without a status", async () => {
    const { getByText } = await renderThemed(<Badge.Badge>Dev3</Badge.Badge>);
    expectColors(getByText("Dev3").element(), "--pluto-gray-l10", "--pluto-gray-l3");
  });

  it.each([
    ["success", "--pluto-primary-text", "--pluto-primary-z-15"],
    ["warning", "--pluto-warning-text", "--pluto-warning-z-15"],
    ["error", "--pluto-error-text", "--pluto-error-z-15"],
  ] as const)("should color a %s badge with its tokens", async (status, text, fill) => {
    const { getByText } = await renderThemed(
      <Badge.Badge status={status}>Word</Badge.Badge>,
    );
    expectColors(getByText("Word").element(), text, fill);
  });

  it("should color the auto badge violet", async () => {
    const { getByText } = await renderThemed(<Input.AutoBadge />);
    expectColors(getByText("Auto").element(), "--pluto-auto-text", "--pluto-auto-z-15");
  });
});
