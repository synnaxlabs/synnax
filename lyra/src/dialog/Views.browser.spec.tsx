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

describe("Views", () => {
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
      .closest(`.${CSS.B("views")}`)!;
    const before = box.getBoundingClientRect().width;
    await getByText("open").click();
    await expect.element(getByText("wide")).toBeVisible();
    expect(box.getBoundingClientRect().width).toBe(before);
  });
});
