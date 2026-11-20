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
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { Device } from "@/device";

const DELAY = TimeSpan.milliseconds(10);

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY}>{children}</Tooltip.Config>
  </Triggers.Provider>
);

const getTooltip = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".lyra-tooltip");

describe("Device location tooltip", () => {
  it("should show the whole location when part of it is hidden", async () => {
    const value = "\\Device\\NPF_{6A1B}.Slot 3";
    const c = await renderThemed(
      <Wrapper>
        <Device.Location value={value} format={{ shape: "slot" }} />
      </Wrapper>,
    );
    await userEvent.hover(c.getByText("Slot 3"));
    await expect.poll(() => getTooltip()?.textContent).toBe(value);
  });

  it("should not show when the whole location shows", async () => {
    const c = await renderThemed(
      <Wrapper>
        <Device.Location value="cDAQ1Mod1" />
      </Wrapper>,
    );
    await userEvent.hover(c.getByText("cDAQ1Mod1"));
    await new Promise((resolve) => setTimeout(resolve, DELAY.milliseconds * 10));
    expect(getTooltip()).toBeNull();
  });
});
