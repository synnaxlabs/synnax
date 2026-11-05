// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Icon } from "@/icon";
import { computed, renderThemed, resolve } from "@/styletest";

const COLOR = "rgb(255, 0, 0)";

const LEVELS = [
  ["Low", Icon.Meter.Low, 1],
  ["Medium", Icon.Meter.Medium, 2],
  ["High", Icon.Meter.High, 3],
] as const;

describe("Icon.Meter", () => {
  it.each(LEVELS)(
    "should fill the bars of %s up to its level in the icon's color",
    async (_, Meter, filled) => {
      const { container } = await renderThemed(
        <span style={{ color: COLOR }}>
          <Meter />
        </span>,
      );
      const fills = Array.from(container.querySelectorAll("rect"), (bar) =>
        computed(bar, "fill"),
      );
      const empty = resolve("var(--pluto-gray-l5)", "fill");
      expect(fills).toEqual([
        ...Array<string>(filled).fill(COLOR),
        ...Array<string>(3 - filled).fill(empty),
      ]);
    },
  );
});
