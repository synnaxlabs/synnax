// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it, vi } from "vitest";

import { ChangeVariant } from "@/table/cells/ChangeVariant";

describe("ChangeVariant", () => {
  it("should list every variant when opened", async () => {
    const c = await renderThemed(<ChangeVariant value="text" onChange={vi.fn()} />);
    await c.getByLabelText("Change cell type").click();
    await expect.element(c.getByRole("option", { name: "Text" })).toBeInTheDocument();
    await expect.element(c.getByRole("option", { name: "Value" })).toBeInTheDocument();
  });

  it("should call onChange with the picked variant", async () => {
    const onChange = vi.fn();
    const c = await renderThemed(<ChangeVariant value="text" onChange={onChange} />);
    await c.getByLabelText("Change cell type").click();
    await c.getByText("Value").click();
    expect(onChange).toHaveBeenCalledWith("value", expect.anything());
  });
});
