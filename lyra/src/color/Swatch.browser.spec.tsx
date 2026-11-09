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
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

describe("Swatch", () => {
  it("should give the focus back to the swatch when its picker closes", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Color.Swatch value="#ff0000" onChange={() => {}} aria-label="Fill" />
      </Triggers.Provider>,
    );
    const swatch = page.getByRole("button", { name: "Fill" }).element();
    assert(swatch instanceof HTMLElement);
    swatch.focus();
    await userEvent.keyboard("{Enter}");
    const preset = page.getByRole("dialog").getByRole("button").last().element();
    assert(preset instanceof HTMLElement);
    preset.focus();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(swatch);
  });
});
