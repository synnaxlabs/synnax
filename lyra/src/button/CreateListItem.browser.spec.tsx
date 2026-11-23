// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { Button } from "@/button";
import { renderThemed } from "@/styletest";

describe("CreateListItem", () => {
  it("should take the height and font size of a small button when small", async () => {
    await renderThemed(
      <>
        <Button.CreateListItem size="small">Create</Button.CreateListItem>
        <Button.Button size="small">Save</Button.Button>
      </>,
    );
    const create = page.getByRole("button", { name: "Create" }).element();
    const save = page.getByRole("button", { name: "Save" }).element();
    expect(create.getBoundingClientRect().height).toBe(
      save.getBoundingClientRect().height,
    );
    expect(getComputedStyle(create).fontSize).toBe(getComputedStyle(save).fontSize);
  });
});
