// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { renderThemed } from "@/styletest";
import { Text } from "@/text";

const ID = "editable";
const LONG_NAME = "A name far wider than the box that holds it";

const Field = (): ReactElement => {
  const [value, setValue] = useState("Name");
  return (
    <div style={{ width: "8rem" }}>
      <Text.Editable id={ID} value={value} overflow="fade" onChange={setValue} />
      <button>Elsewhere</button>
    </div>
  );
};

const typeLongName = async (el: Element): Promise<void> => {
  await expect.poll(() => el.getAttribute("contenteditable")).toBe("true");
  await userEvent.keyboard(LONG_NAME);
  expect(el.scrollLeft).toBeGreaterThan(0);
};

describe("Editable", () => {
  describe("a cut name", () => {
    it("should scroll back to its start when Enter commits", async () => {
      const { getByText } = await renderThemed(<Field />);
      const el = getByText("Name").element();
      await userEvent.dblClick(el);
      await typeLongName(el);
      await userEvent.keyboard("{Enter}");
      await expect.poll(() => el.textContent).toBe(LONG_NAME);
      expect(el.scrollLeft).toBe(0);
    });

    it("should scroll back to its start when a blur commits", async () => {
      const { getByText, getByRole } = await renderThemed(<Field />);
      const el = getByText("Name").element();
      await userEvent.dblClick(el);
      await typeLongName(el);
      await userEvent.click(getByRole("button", { name: "Elsewhere" }));
      await expect.poll(() => el.textContent).toBe(LONG_NAME);
      expect(el.scrollLeft).toBe(0);
    });

    it("should scroll back to its start when an edit from elsewhere ends", async () => {
      const { getByText } = await renderThemed(<Field />);
      const el = getByText("Name").element();
      Text.edit(ID);
      await typeLongName(el);
      await userEvent.keyboard("{Enter}");
      await expect.poll(() => el.textContent).toBe(LONG_NAME);
      expect(el.scrollLeft).toBe(0);
    });
  });
});
