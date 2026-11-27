// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { assert, describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { Color } from "@/color";
import { CSS } from "@/css";
import { renderThemed, treatment } from "@/styletest";
import { Triggers } from "@/triggers";

const RED = "#ff0000";

const LIBRARY: Color.Library = {
  palettes: [
    {
      key: "p",
      name: "Alarms",
      colors: [{ key: "fault", name: "Pressurization fault", light: RED }],
    },
  ],
  named: "fault",
  onNamedChange: () => {},
};

const boxWidth = (name: string): number => {
  const box = page
    .getByRole("textbox", { name })
    .element()
    .closest(`.${CSS.B("color-hex-text")}`);
  assert(box != null);
  return box.getBoundingClientRect().width;
};

describe("Input", () => {
  it("should draw a hex, its #, and a linked name in the text font", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Input
          aria-label="Fixed"
          value={RED}
          fallback={RED}
          onChange={() => {}}
        />
        <Color.Input
          aria-label="Linked"
          value={RED}
          fallback={RED}
          library={LIBRARY}
          onChange={() => {}}
        />
      </Triggers.Provider>,
    );
    for (const name of ["Fixed", "Linked"])
      expect(treatment(page.getByRole("textbox", { name }).element()), name).toBe(
        "prose",
      );
    expect(treatment(page.getByText("#", { exact: true }).element())).toBe("prose");
  });

  it("should take one width while fixed, linked, Auto, or mixed", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Input
          aria-label="Fixed"
          value={RED}
          fallback={RED}
          onChange={() => {}}
        />
        <Color.Input
          aria-label="Linked"
          value={RED}
          fallback={RED}
          library={LIBRARY}
          onChange={() => {}}
        />
        <Color.Input aria-label="Auto" fallback={RED} onChange={() => {}} />
        <Color.Input
          aria-label="Mixed"
          value={RED}
          fallback={RED}
          mixed
          onChange={() => {}}
        />
      </Triggers.Provider>,
    );
    const fixed = boxWidth("Fixed");
    for (const name of ["Linked", "Auto", "Mixed"])
      expect(boxWidth(name), name).toBeCloseTo(fixed, 1);
  });

  it("should fit the widest 8-digit hex", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Input
          aria-label="Translucent"
          value="#dddddddd"
          fallback={RED}
          onChange={() => {}}
        />
      </Triggers.Provider>,
    );
    const hex = page.getByRole("textbox", { name: "Translucent" }).element();
    assert(hex instanceof HTMLInputElement);
    expect(hex.value).toHaveLength(8);
    expect(hex.scrollWidth).toBeLessThanOrEqual(hex.clientWidth);
  });
});
