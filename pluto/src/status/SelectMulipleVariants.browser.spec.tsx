// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Status as Base } from "@synnaxlabs/lyra/status";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { Status } from "@/status";

describe("SelectMultipleVariants", () => {
  const createTestComponent = () => {
    const onChange = vi.fn();
    const TestSelect = () => {
      const [value, setValue] = useState<Base.Variant[]>([]);
      const handleChange = (keys: Base.Variant[]) => {
        setValue(keys);
        onChange(keys);
      };
      return (
        <Triggers.Provider>
          <Status.SelectMultipleVariants value={value} onChange={handleChange} />
        </Triggers.Provider>
      );
    };
    return { TestSelect, onChange };
  };

  it("should render a selection trigger", async () => {
    const { TestSelect } = createTestComponent();
    const c = await renderThemed(<TestSelect />);
    await expect.element(c.getByText("Variants")).toBeInTheDocument();
  });

  it("should open and show all variant options", async () => {
    const { TestSelect } = createTestComponent();
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variants").click();
    await expect.element(c.getByText("Success")).toBeInTheDocument();
    await expect.element(c.getByText("Error")).toBeInTheDocument();
    await expect.element(c.getByText("Warning")).toBeInTheDocument();
    await expect.element(c.getByText("Info")).toBeInTheDocument();
    await expect.element(c.getByText("Loading")).toBeInTheDocument();
    await expect.element(c.getByText("Disabled")).toBeInTheDocument();
  });

  it("should call onChange when a variant is selected", async () => {
    const { TestSelect, onChange } = createTestComponent();
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variants").click();
    await c.getByText("Error").click();
    expect(onChange).toHaveBeenCalledWith(["error"]);
  });

  it("should allow selecting multiple variants", async () => {
    const { TestSelect, onChange } = createTestComponent();
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variants").click();
    await c.getByRole("option", { name: "Error" }).click();
    await c.getByRole("option", { name: "Warning" }).click();
    expect(onChange).toHaveBeenCalledWith(["error", "warning"]);
  });

  it("should filter options when searching", async () => {
    const { TestSelect } = createTestComponent();
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variants").click();
    await userEvent.fill(c.getByPlaceholder("Search variants..."), "Err");
    const hidden = (text: string) =>
      c.getByText(text).element().closest("[hidden]") != null;
    await expect.poll(() => hidden("Success")).toBe(true);
    expect(hidden("Error")).toBe(false);
    expect(hidden("Warning")).toBe(true);
  });
});
