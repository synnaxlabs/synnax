// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { Steps } from "@/steps";
import { renderThemed, resolve } from "@/styletest";

describe("Steps", () => {
  it("should draw the step number in the text font", async () => {
    await renderThemed(
      <Steps.List>
        <Steps.Item>Install the Driver.</Steps.Item>
      </Steps.List>,
    );
    const prose = resolve("var(--pluto-font-family)", "font-family");
    expect(prose).not.toBe(resolve("var(--pluto-code-font-family)", "font-family"));
    const marker = getComputedStyle(page.getByRole("listitem").element(), "::before");
    expect(marker.content).toBe("counter(list-item)");
    expect(marker.fontFamily).toBe(prose);
    expect(marker.fontVariantNumeric).toBe("normal");
  });
});
