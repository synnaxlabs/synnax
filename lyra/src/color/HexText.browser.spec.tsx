// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Color } from "@/color";
import { renderThemed, treatment } from "@/styletest";

const box = (name: string): Element => page.getByRole("textbox", { name }).element();

const caseOf = (name: string): string => getComputedStyle(box(name)).textTransform;

describe("HexText", () => {
  it("should capitalize a hex but not a name until the user types", async () => {
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
      </>,
    );
    expect(caseOf("Fixed")).toBe("uppercase");
    expect(caseOf("Named")).toBe("none");
    await userEvent.fill(page.getByRole("textbox", { name: "Named" }), "0f0");
    expect(caseOf("Named")).toBe("uppercase");
  });

  it("should draw a hex, its #, and a name in the text font", async () => {
    await renderThemed(
      <>
        <Color.HexText
          aria-label="Fixed"
          text="ff0000"
          startContent="#"
          alpha={1}
          onChange={() => {}}
        />
        <Color.HexText
          aria-label="Named"
          text="Fault"
          named
          alpha={1}
          onChange={() => {}}
        />
      </>,
    );
    expect(treatment(page.getByText("#", { exact: true }).element())).toBe("prose");
    expect(treatment(box("Fixed"))).toBe("prose");
    expect(treatment(box("Named"))).toBe("prose");
  });
});
