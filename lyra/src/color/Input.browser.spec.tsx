// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { assert, describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Input } from "@/input";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const RED = "#ff0000";

const LIBRARY: NonNullable<Color.InputProps["library"]> = {
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

const fontOf = (
  name: string,
): Pick<CSSStyleDeclaration, "fontFamily" | "textTransform"> => {
  const { fontFamily, textTransform } = getComputedStyle(
    page.getByRole("textbox", { name }).element(),
  );
  return { fontFamily, textTransform };
};

describe("Input", () => {
  it("should show a linked name in the text font until the user types", async () => {
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
        <Input.Text aria-label="Plain" value="" onChange={() => {}} />
      </Triggers.Provider>,
    );
    expect(fontOf("Linked")).toEqual(fontOf("Plain"));
    await userEvent.fill(page.getByRole("textbox", { name: "Linked" }), "0f0");
    expect(fontOf("Linked")).toEqual(fontOf("Fixed"));
  });

  it("should draw the # in the code font of the digits", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Input
          aria-label="Fixed"
          value={RED}
          fallback={RED}
          onChange={() => {}}
        />
        <Input.Text aria-label="Plain" value="" onChange={() => {}} />
      </Triggers.Provider>,
    );
    const hash = page
      .getByRole("textbox", { name: "Fixed" })
      .element()
      .closest(`.${CSS.B("color-hex-text")}`)
      ?.querySelector(`.${CSS.BE("input", "start-content")}`);
    assert(hash != null);
    expect(hash.textContent).toBe("#");
    expect(getComputedStyle(hash).fontFamily).toBe(fontOf("Fixed").fontFamily);
    expect(fontOf("Fixed").fontFamily).not.toBe(fontOf("Plain").fontFamily);
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
});
