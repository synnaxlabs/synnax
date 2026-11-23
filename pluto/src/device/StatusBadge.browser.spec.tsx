// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status } from "@synnaxlabs/client";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { Device } from "@/device";

describe("StatusBadge", () => {
  it("should show the message and the description in its tooltip", async () => {
    const s = status.create<typeof device.statusDetailsZ>({
      variant: "error",
      message: "Health check failed",
      description: "The server returned 503.",
      details: { rack: 1, connected: true },
    });
    const c = await renderThemed(
      <Triggers.Provider>
        <Tooltip.Config delay={TimeSpan.milliseconds(1)}>
          <Device.StatusBadge status={s} />
        </Tooltip.Config>
      </Triggers.Provider>,
    );
    await userEvent.hover(c.getByText("Connected"));
    await expect
      .poll(() => document.querySelector(".lyra-tooltip")?.textContent)
      .toBe("Health check failedThe server returned 503.");
  });
});
