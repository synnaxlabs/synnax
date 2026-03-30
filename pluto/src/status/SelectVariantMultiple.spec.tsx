// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Status } from "@/status";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

describe("SelectVariantMultiple", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  const createTestComponent = () => {
    const onChange = vi.fn();
    const TestSelect = () => {
      const [value, setValue] = useState<status.Variant[]>([]);
      const handleChange = (keys: status.Variant[]) => {
        setValue(keys);
        onChange(keys);
      };
      return (
        <Triggers.Provider>
          <Status.SelectVariantMultiple value={value} onChange={handleChange} />
        </Triggers.Provider>
      );
    };
    return { TestSelect, onChange };
  };

  it("should render a selection trigger", () => {
    const { TestSelect } = createTestComponent();
    const c = render(<TestSelect />);
    expect(c.getByText("Select variants")).toBeTruthy();
  });

  it("should open and show all variant options", () => {
    const { TestSelect } = createTestComponent();
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select variants"));
    expect(c.getByText("Success")).toBeTruthy();
    expect(c.getByText("Error")).toBeTruthy();
    expect(c.getByText("Warning")).toBeTruthy();
    expect(c.getByText("Info")).toBeTruthy();
    expect(c.getByText("Loading")).toBeTruthy();
    expect(c.getByText("Disabled")).toBeTruthy();
  });

  it("should call onChange when a variant is selected", () => {
    const { TestSelect, onChange } = createTestComponent();
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select variants"));
    fireEvent.click(c.getByText("Error"));
    expect(onChange).toHaveBeenCalledWith(["error"]);
  });

  it("should allow selecting multiple variants", () => {
    const { TestSelect, onChange } = createTestComponent();
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select variants"));
    fireEvent.click(c.getByText("Error"));
    fireEvent.click(c.getByText("Warning"));
    expect(onChange).toHaveBeenCalledWith(["error", "warning"]);
  });

  it("should filter options when searching", () => {
    const { TestSelect } = createTestComponent();
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select variants"));
    fireEvent.change(c.getByPlaceholderText("Search variants..."), {
      target: { value: "Err" },
    });
    expect(c.getByText("Error")).toBeTruthy();
    expect(c.queryByText("Success")).toBeNull();
    expect(c.queryByText("Warning")).toBeNull();
  });
});
