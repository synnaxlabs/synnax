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

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const layer = (): HTMLElement => {
  const el = document.querySelector(`.${CSS.BE("haul", "layer")}`);
  assert(el instanceof HTMLElement);
  return el;
};

describe("Haul", () => {
  it("should keep the preview layer above an open modal", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Haul.Provider>
          <Dialog.Frame variant="modal">
            <Dialog.Trigger>Open</Dialog.Trigger>
            <Dialog.Dialog>Modal</Dialog.Dialog>
          </Dialog.Frame>
        </Haul.Provider>
      </Triggers.Provider>,
    );
    expect(layer().closest("dialog")).toBeNull();
    await userEvent.click(page.getByRole("button", { name: "Open" }));
    expect(layer().closest("dialog")).toBe(page.getByRole("dialog").element());
  });
});
