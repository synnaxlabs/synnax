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

import { Circle } from "@/schematic/node/general/circle/Primitive";

const queryStroke = (container: HTMLElement): string | null | undefined =>
  container.querySelector("circle")?.getAttribute("stroke");

describe("Circle", () => {
  describe("color", () => {
    it("should resolve a missing color to a themed stroke", () => {
      const { container } = render(<Circle radius={20} />);
      expect(queryStroke(container)).toMatch(/^rgba?\(/);
    });

    it("should resolve the ZERO follow-theme sentinel like a missing color", () => {
      const zero = render(<Circle radius={20} color={color.ZERO} />).container;
      const unset = render(<Circle radius={20} />).container;
      expect(queryStroke(zero)).toBe(queryStroke(unset));
    });

    it("should honor an explicit color", () => {
      const { container } = render(<Circle radius={20} color="#ff0000" />);
      expect(queryStroke(container)).toMatch(/255\s*,\s*0\s*,\s*0/);
    });
  });
});
