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

import { mockBoundingClientRect } from "@/testutil/dom";

import { SelectVariant } from "./SelectVariant";
import { VARIANT_DATA } from "./variantData";

describe("SelectVariant", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should export all six variant entries in VARIANT_DATA", () => {
    expect(VARIANT_DATA).toHaveLength(6);
    const keys = VARIANT_DATA.map((d) => d.key);
    expect(keys).toContain("success");
    expect(keys).toContain("error");
    expect(keys).toContain("warning");
    expect(keys).toContain("info");
    expect(keys).toContain("loading");
    expect(keys).toContain("disabled");
  });

  const onChange = vi.fn();
  const TestSelect = () => {
    const [value, setValue] = useState<status.Variant | undefined>(undefined);
    const handleChange = (key: status.Variant) => {
      setValue(key);
      onChange(key);
    };
    return <SelectVariant value={value as status.Variant} onChange={handleChange} />;
  };

  it("should render a selection trigger", () => {
    const c = render(<TestSelect />);
    expect(c.getByText("Select a variant")).toBeTruthy();
  });

  it("should open and show all variant options", () => {
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select a variant"));
    expect(c.getByText("Success")).toBeTruthy();
    expect(c.getByText("Error")).toBeTruthy();
    expect(c.getByText("Warning")).toBeTruthy();
    expect(c.getByText("Info")).toBeTruthy();
    expect(c.getByText("Loading")).toBeTruthy();
    expect(c.getByText("Disabled")).toBeTruthy();
  });

  it("should call onChange when a variant is selected", () => {
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select a variant"));
    fireEvent.click(c.getByText("Error"));
    expect(onChange).toHaveBeenCalledWith("error");
  });
});
