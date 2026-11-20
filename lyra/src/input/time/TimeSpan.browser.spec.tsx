// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Input } from "@/input";
import { renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const HINT = "Try 2h 30min, 1:30:00, 90, or an hour and a half";

describe("Input.TimeSpan", () => {
  // Only a real key press lets React hide the editor before the modal sees the Escape.
  it("should close only its suggestions in a modal on one Escape", async () => {
    await renderThemed(
      <Triggers.Provider>
        <Dialog.Frame variant="modal">
          <Dialog.Trigger>Open</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Modal content</p>
            <Input.TimeSpan
              inline
              value={Number(TimeSpan.HOUR.valueOf())}
              onChange={vi.fn()}
            />
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    await userEvent.click(page.getByText("Open"));
    await userEvent.click(page.getByRole("textbox"));
    await expect.element(page.getByText(HINT)).toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText(HINT)).not.toBeInTheDocument();
    await expect.element(page.getByText("Modal content")).toBeVisible();
    await userEvent.keyboard("{Escape}");
    await expect.element(page.getByText("Modal content")).not.toBeInTheDocument();
  });
});
