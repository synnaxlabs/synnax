// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Value } from "@/schematic/node/general/value/Primitive";

const getUnits = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-value__units .lyra-text");
  if (el == null) throw new Error("expected a units text");
  return el;
};

describe("value symbol", () => {
  describe("units", () => {
    it("should paint the units in the quiet gray shade", () => {
      const { container } = render(
        <Value
          strokeColor={color.construct("#00ff00")}
          units="psi"
          orientation="left"
          inlineSize={80}
        />,
      );
      const units = getUnits(container);
      expect(units.classList.contains("lyra--color-9")).toBe(true);
      expect(units.style.color).toBe("");
    });

    it("should hide the units cell when the units are empty", () => {
      const { container } = render(
        <Value
          strokeColor={color.construct("#00ff00")}
          units=""
          orientation="left"
          inlineSize={80}
        />,
      );
      expect(container.querySelector(".pluto-value__units")).toBeNull();
    });
  });
});
