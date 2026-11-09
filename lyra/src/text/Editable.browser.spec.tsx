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

import { renderThemed, selectedByTripleClick, withBodySelectionOff } from "@/styletest";
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
  // Text.edit sets contenteditable before the field takes focus and selects its text.
  await expect.poll(() => document.activeElement).toBe(el);
  await userEvent.keyboard(LONG_NAME);
  expect(el.scrollLeft).toBeGreaterThan(0);
};

describe("Editable", () => {
  describe("under a body with selection off", () => {
    it("should not let a user select a name at rest", async () => {
      const { getByText } = await renderThemed(
        <Text.Editable
          id={ID}
          value="Name"
          allowDoubleClick={false}
          onChange={() => {}}
        />,
      );
      expect(await selectedByTripleClick(getByText("Name").element())).toBe("");
    });

    it("should replace the name with what a user types", async () => {
      const { getByText } = await renderThemed(<Field />);
      const el = getByText("Name").element();
      await withBodySelectionOff(async () => {
        await userEvent.dblClick(el);
        await expect.poll(() => document.activeElement).toBe(el);
        await userEvent.keyboard("Renamed{Enter}");
      });
      await expect.poll(() => el.textContent).toBe("Renamed");
    });
  });

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
