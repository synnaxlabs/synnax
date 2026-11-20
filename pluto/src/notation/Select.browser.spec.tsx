// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import type { PropsWithChildren, ReactElement } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";
import type { RenderResult } from "vitest-browser-react";

import { Notation } from "@/notation";

const DELAY = TimeSpan.milliseconds(10);

// The labels give way to icons at this width and below.
const ICONS_WIDTH = 1100;

const { innerWidth, innerHeight } = window;

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY}>{children}</Tooltip.Config>
  </Triggers.Provider>
);

const renderSelect = async () =>
  await renderThemed(
    <Wrapper>
      <Notation.Select value="standard" onChange={() => {}} />
    </Wrapper>,
  );

// CSS hides the label while the button shows its icon, so the label names no button.
const engineering = (c: RenderResult): Locator =>
  c.getByRole("button").filter({ hasText: "Engineering" });

const tooltip = (): string | null | undefined =>
  document.querySelector(".lyra-tooltip")?.textContent;

describe("Notation.Select", () => {
  afterEach(async () => {
    await page.viewport(innerWidth, innerHeight);
  });

  it("should show no tooltip while its buttons show labels", async () => {
    await page.viewport(ICONS_WIDTH + 100, innerHeight);
    const c = await renderSelect();
    await userEvent.hover(engineering(c));
    await new Promise((resolve) => setTimeout(resolve, DELAY.milliseconds * 10));
    expect(tooltip()).toBeUndefined();
  });

  it("should add the tooltips when a resize swaps the labels for icons", async () => {
    await page.viewport(ICONS_WIDTH + 100, innerHeight);
    const c = await renderSelect();
    await page.viewport(ICONS_WIDTH, innerHeight);
    await userEvent.hover(engineering(c));
    await expect.poll(tooltip).toBe("Engineering");
  });

  it("should name each button in a tooltip while it shows icons", async () => {
    await page.viewport(ICONS_WIDTH, innerHeight);
    const c = await renderSelect();
    await userEvent.hover(engineering(c));
    await expect.poll(tooltip).toBe("Engineering");
  });
});
