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

import { Value } from "@/schematic/node/general/value/Primitive";

const queryBorder = (container: HTMLElement): string | undefined =>
  container.querySelector<HTMLElement>(".pluto-value")?.style.borderColor;

describe("Value", () => {
  describe("color", () => {
    it("should resolve a missing color to a themed border", () => {
      const { container } = render(<Value units="psi" />);
      expect(queryBorder(container)).toMatch(/^rgba?\(/);
    });

    it("should honor an explicit border color", () => {
      const { container } = render(<Value units="psi" color="#ff0000" />);
      expect(queryBorder(container)).toMatch(/255\s*,\s*0\s*,\s*0/);
    });
  });
});
