// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import type { TimestampFormat } from "@synnaxlabs/x";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Telem } from "@/telem";

describe("SelectTimestampFormat", () => {
  const onChange = vi.fn();
  const TestSelect = () => {
    const [value, setValue] = useState<TimestampFormat | undefined>(undefined);
    const handleChange = (key: TimestampFormat) => {
      setValue(key);
      onChange(key);
    };
    return <Telem.SelectTimestampFormat value={value} onChange={handleChange} />;
  };

  it("should render a selection trigger", async () => {
    const c = await renderThemed(<TestSelect />);
    await expect.element(c.getByText("Timestamp format")).toBeInTheDocument();
  });

  it("should open and show all format options", async () => {
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Timestamp format").click();
    await expect.element(c.getByText("ISO 8601")).toBeInTheDocument();
    await expect.element(c.getByText("ISO date")).toBeInTheDocument();
    await expect.element(c.getByText("Time", { exact: true })).toBeInTheDocument();
    await expect.element(c.getByText("Precise time")).toBeInTheDocument();
    await expect.element(c.getByText("Date", { exact: true })).toBeInTheDocument();
    await expect.element(c.getByText("Date and time")).toBeInTheDocument();
    await expect.element(c.getByText("Precise date")).toBeInTheDocument();
  });

  it("should call onChange when a format is selected", async () => {
    const c = await renderThemed(<TestSelect />);
    await c.getByText("Timestamp format").click();
    await c.getByText("Precise date").click();
    expect(onChange).toHaveBeenCalledWith("preciseDate");
  });
});
