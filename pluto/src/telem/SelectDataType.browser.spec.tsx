// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { DataType } from "@synnaxlabs/x";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { RenderResult } from "vitest-browser-react";

import { Telem } from "@/telem";

describe("SelectDataType", () => {
  const onChange = vi.fn();

  const SelectWrapper = (props: {
    hideVariableDensity?: boolean;
    hideDataTypes?: DataType[];
  }) => {
    const [value, setValue] = useState("");
    const handleChange = (key: string) => {
      setValue(key);
      onChange(key);
    };
    return <Telem.SelectDataType value={value} onChange={handleChange} {...props} />;
  };

  const text = (c: RenderResult, value: string) => c.getByText(value, { exact: true });

  it("should render a selection trigger", async () => {
    const c = await renderThemed(<SelectWrapper />);
    await expect.element(text(c, "Data type")).toBeInTheDocument();
  });

  it("should display data types when opened", async () => {
    const c = await renderThemed(<SelectWrapper />);
    await text(c, "Data type").click();
    await expect.element(text(c, "float64")).toBeInTheDocument();
    await expect.element(text(c, "int32")).toBeInTheDocument();
    await expect.element(text(c, "Timestamp")).toBeInTheDocument();
  });

  it("should not display the UNKNOWN data type", async () => {
    const c = await renderThemed(<SelectWrapper />);
    await text(c, "Data type").click();
    await expect.element(text(c, "float64")).toBeInTheDocument();
    await expect.element(text(c, "Unknown")).not.toBeInTheDocument();
  });

  it("should display the boolean data type as Boolean", async () => {
    const c = await renderThemed(<SelectWrapper />);
    await text(c, "Data type").click();
    await expect.element(text(c, "Boolean")).toBeInTheDocument();
    await expect.element(text(c, "boolean")).not.toBeInTheDocument();
  });

  it("should display UUID and JSON in all caps", async () => {
    const c = await renderThemed(<SelectWrapper />);
    await text(c, "Data type").click();
    await expect.element(text(c, "UUID")).toBeInTheDocument();
    await expect.element(text(c, "JSON")).toBeInTheDocument();
  });

  it("should call onChange when a data type is selected", async () => {
    onChange.mockClear();
    const c = await renderThemed(<SelectWrapper />);
    await text(c, "Data type").click();
    await text(c, "float32").click();
    expect(onChange).toHaveBeenCalledWith("float32");
  });

  it("should hide variable density types when hideVariableDensity is true", async () => {
    const c = await renderThemed(<SelectWrapper hideVariableDensity />);
    await text(c, "Data type").click();
    await expect.element(text(c, "float64")).toBeInTheDocument();
    await expect.element(text(c, "String")).not.toBeInTheDocument();
    await expect.element(text(c, "JSON")).not.toBeInTheDocument();
  });

  it("should hide specific data types when hideDataTypes is provided", async () => {
    const c = await renderThemed(
      <SelectWrapper hideDataTypes={[DataType.FLOAT32, DataType.INT64]} />,
    );
    await text(c, "Data type").click();
    await expect.element(text(c, "float64")).toBeInTheDocument();
    await expect.element(text(c, "float32")).not.toBeInTheDocument();
    await expect.element(text(c, "int64")).not.toBeInTheDocument();
  });
});
