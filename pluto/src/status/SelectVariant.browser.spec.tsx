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
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Status } from "@/status";

describe("SelectVariant", () => {
  const onChange = vi.fn();
  const TestSelect = () => {
    const [value, setValue] = useState<Base.Variant | undefined>(undefined);
    const handleChange = (key: Base.Variant) => {
      setValue(key);
      onChange(key);
    };
    return <Status.SelectVariant value={value} onChange={handleChange} />;
  };

  it("should render a selection trigger", async () => {
    const c = await renderThemed(<TestSelect />);
    await expect.element(c.getByText("Variant")).toBeInTheDocument();
  });

  it("should open and show all variant options", async () => {
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variant").click();
    await expect.element(c.getByText("Success")).toBeInTheDocument();
    await expect.element(c.getByText("Error")).toBeInTheDocument();
    await expect.element(c.getByText("Warning")).toBeInTheDocument();
    await expect.element(c.getByText("Info")).toBeInTheDocument();
    await expect.element(c.getByText("Loading")).toBeInTheDocument();
    await expect.element(c.getByText("Disabled")).toBeInTheDocument();
  });

  it("should call onChange when a variant is selected", async () => {
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Variant").click();
    await c.getByText("Error").click();
    expect(onChange).toHaveBeenCalledWith("error");
  });
});
