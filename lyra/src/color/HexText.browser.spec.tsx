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

const fontOf = (
  name: string,
): Pick<CSSStyleDeclaration, "fontFamily" | "textTransform"> => {
  const { fontFamily, textTransform } = getComputedStyle(
    page.getByRole("textbox", { name }).element(),
  );
  return { fontFamily, textTransform };
};

describe("HexText", () => {
  it("should show a name in the text font until the user types", async () => {
    await renderThemed(
      <>
        <Color.HexText aria-label="Fixed" text="ff0000" alpha={1} onChange={() => {}} />
        <Color.HexText
          aria-label="Named"
          text="Fault"
          named
          alpha={1}
          onChange={() => {}}
        />
        <Input.Text aria-label="Plain" value="" onChange={() => {}} />
      </>,
    );
    expect(fontOf("Named")).toEqual(fontOf("Plain"));
    expect(fontOf("Fixed")).not.toEqual(fontOf("Plain"));
    await userEvent.fill(page.getByRole("textbox", { name: "Named" }), "0f0");
    expect(fontOf("Named")).toEqual(fontOf("Fixed"));
  });

  it("should draw a # start in the code font of the digits", async () => {
    await renderThemed(
      <>
        <Color.HexText
          aria-label="Fixed"
          text="ff0000"
          startContent="#"
          alpha={1}
          onChange={() => {}}
        />
        <Input.Text aria-label="Plain" value="" onChange={() => {}} />
      </>,
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
});
