// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

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

  it("should render a selection trigger", () => {
    const c = render(<SelectWrapper />);
    expect(c.getByText("Select a data type")).toBeTruthy();
  });

  it("should display data types when opened", () => {
    const c = render(<SelectWrapper />);
    fireEvent.click(c.getByText("Select a data type"));
    expect(c.getByText("float64")).toBeTruthy();
    expect(c.getByText("int32")).toBeTruthy();
    expect(c.getByText("Timestamp")).toBeTruthy();
  });

  it("should not display the UNKNOWN data type", () => {
    const c = render(<SelectWrapper />);
    fireEvent.click(c.getByText("Select a data type"));
    expect(c.queryByText("Unknown")).toBeNull();
  });

  it("should display UUID and JSON in all caps", () => {
    const c = render(<SelectWrapper />);
    fireEvent.click(c.getByText("Select a data type"));
    expect(c.getByText("UUID")).toBeTruthy();
    expect(c.getByText("JSON")).toBeTruthy();
  });

  it("should call onChange when a data type is selected", () => {
    onChange.mockClear();
    const c = render(<SelectWrapper />);
    fireEvent.click(c.getByText("Select a data type"));
    fireEvent.click(c.getByText("float32"));
    expect(onChange).toHaveBeenCalledWith("float32");
  });

  it("should hide variable density types when hideVariableDensity is true", () => {
    const c = render(<SelectWrapper hideVariableDensity />);
    fireEvent.click(c.getByText("Select a data type"));
    expect(c.queryByText("String")).toBeNull();
    expect(c.queryByText("JSON")).toBeNull();
    expect(c.getByText("float64")).toBeTruthy();
  });

  it("should hide specific data types when hideDataTypes is provided", () => {
    const c = render(
      <SelectWrapper hideDataTypes={[DataType.FLOAT32, DataType.INT64]} />,
    );
    fireEvent.click(c.getByText("Select a data type"));
    expect(c.queryByText("float32")).toBeNull();
    expect(c.queryByText("int64")).toBeNull();
    expect(c.getByText("float64")).toBeTruthy();
  });
});
