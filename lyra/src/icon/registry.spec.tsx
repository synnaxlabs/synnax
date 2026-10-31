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

describe("registry", () => {
  describe("AutoStart", () => {
    it("should draw its arc and play glyph unmasked", () => {
      const { container } = render(<Icon.AutoStart />);
      const svg = container.querySelector("svg");
      expect(svg?.querySelectorAll("path")).toHaveLength(2);
      expect(svg?.querySelector("mask")).toBeNull();
    });
  });

  describe("AutoStartOff", () => {
    it("should cut its glyph under the slash with a mask of its own", () => {
      const { container } = render(
        <>
          <Icon.AutoStartOff />
          <Icon.AutoStartOff />
        </>,
      );
      const ids = Array.from(container.querySelectorAll("svg"), (svg) => {
        const id = svg.querySelector("mask")?.id;
        expect(svg.querySelector("g")?.getAttribute("mask")).toBe(`url(#${id})`);
        return id;
      });
      expect(new Set(ids).size).toBe(2);
    });
  });
});
