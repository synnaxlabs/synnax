// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { renderThemed } from "@/styletest";

const WideOpener = (): ReactElement => {
  const { open } = Dialog.useViews();
  return (
    <button onClick={() => open("Edit item", <div style={{ width: 600 }}>wide</div>)}>
      open
    </button>
  );
};

const ClickOpener = (): ReactElement => {
  const { open } = Dialog.useViews();
  return (
    <Button.Button
      onClick={(e) => open("Edit item", <span>view</span>, e.currentTarget)}
    >
      open
    </Button.Button>
  );
};

describe("Views", () => {
  it("should focus a clicked opener after Back", async () => {
    await renderThemed(
      <Dialog.Views>
        <ClickOpener />
      </Dialog.Views>,
    );
    const opener = page.getByRole("button", { name: "open" });
    await userEvent.click(opener);
    await userEvent.click(page.getByRole("button", { name: "Back" }));
    expect(document.activeElement).toBe(opener.element());
  });

  it("should keep the width of the children under a wider view", async () => {
    const { getByText } = await renderThemed(
      <div style={{ display: "inline-flex" }}>
        <Dialog.Views>
          <div style={{ width: 200 }} />
          <WideOpener />
        </Dialog.Views>
      </div>,
    );
    const box = getByText("open")
      .element()
      .closest(`.${CSS.B("dialog-views")}`)!;
    const before = box.getBoundingClientRect().width;
    await getByText("open").click();
    await expect.element(getByText("wide")).toBeVisible();
    await expect.element(getByText("open")).not.toBeVisible();
    expect(box.getBoundingClientRect().width).toBe(before);
  });

  it("should keep the unscaled width when a view opens under a transform", async () => {
    const { getByText } = await renderThemed(
      <div style={{ display: "inline-flex", transform: "scale(0.5)" }}>
        <Dialog.Views>
          <div style={{ width: 200 }} />
          <WideOpener />
        </Dialog.Views>
      </div>,
    );
    const box = getByText("open")
      .element()
      .closest<HTMLElement>(`.${CSS.B("dialog-views")}`)!;
    await getByText("open").click();
    await expect.element(getByText("wide")).toBeVisible();
    expect(box.offsetWidth).toBe(200);
  });
});
