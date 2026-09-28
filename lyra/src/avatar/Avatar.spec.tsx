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

import { Avatar } from "@/avatar/Avatar";

const hueOf = (name: string): string | undefined =>
  render(<Avatar name={name} />)
    .container.querySelector<HTMLElement>(".pluto-avatar")
    ?.style.getPropertyValue("--pluto-avatar-hue");

describe("Avatar", () => {
  describe("initials", () => {
    it.each([
      ["Hot Fire", "HF"],
      ["Primary", "PR"],
      ["Test Stand 2", "T2"],
      ["stand_1", "S1"],
      ["  ", "?"],
    ])("should show %j as %s", (name, initials) => {
      const c = render(<Avatar name={name} />);
      expect(c.container.textContent).toEqual(initials);
    });
  });

  it("should show the image in place of the initials", () => {
    const c = render(<Avatar name="Hot Fire" image="https://a.test/p.png" />);
    expect(c.container.textContent).toEqual("");
    expect(c.container.querySelector("img")?.getAttribute("src")).toEqual(
      "https://a.test/p.png",
    );
  });

  it("should give the same name the same hue", () => {
    expect(hueOf("Hot Fire")).toMatch(/^\d+$/);
    expect(hueOf("Hot Fire")).toEqual(hueOf("Hot Fire"));
  });

  it("should give same-letter names different hues", () => {
    expect(hueOf("Hot Fire")).not.toEqual(hueOf("Hot Fill"));
  });

  it("should apply the size modifier", () => {
    const c = render(<Avatar name="Hot Fire" size="small" />);
    expect(c.container.firstElementChild?.classList).toContain("pluto--height-small");
  });
});
