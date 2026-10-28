// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { computed, renderThemed, resolve } from "@/styletest";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

const openTooltip = async (): Promise<{ target: Element; tooltip: Element }> => {
  const { getByRole } = await renderThemed(
    <Triggers.Provider>
      <Tooltip.Config delay={TimeSpan.milliseconds(1)}>
        <Dialog.Frame visible>
          <Dialog.Trigger>Open</Dialog.Trigger>
          <Dialog.Dialog>
            <Tooltip.Dialog>
              Tip
              <button>Target</button>
            </Tooltip.Dialog>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Tooltip.Config>
    </Triggers.Provider>,
  );
  const target = page.getByRole("button", { name: "Target" }).element();
  await userEvent.hover(target);
  await expect.element(getByRole("tooltip")).toBeVisible();
  return { target, tooltip: page.getByRole("tooltip").element() };
};

describe("Tooltip", () => {
  it("should paint on the overlay surface", async () => {
    const { tooltip } = await openTooltip();
    expect(computed(tooltip, "background-color")).toBe(resolve("var(--pluto-gray-l2)"));
    expect(computed(tooltip, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    expect(computed(tooltip, "box-shadow")).toBe(
      resolve("var(--pluto-shadow-overlay)", "box-shadow"),
    );
  });

  it("should never take the pointer from its target", async () => {
    const { tooltip } = await openTooltip();
    expect(computed(tooltip, "pointer-events")).toBe("none");
  });

  it("should stack over an open dropdown without covering its target", async () => {
    const { target, tooltip } = await openTooltip();
    const dialog = page.getByRole("dialog").element();
    expect(Number(computed(tooltip, "z-index"))).toBeGreaterThan(
      Number(computed(dialog, "z-index")),
    );
    const t = target.getBoundingClientRect();
    const tip = tooltip.getBoundingClientRect();
    const overlaps =
      tip.left < t.right &&
      tip.right > t.left &&
      tip.top < t.bottom &&
      tip.bottom > t.top;
    expect(overlaps).toBe(false);
  });
});
