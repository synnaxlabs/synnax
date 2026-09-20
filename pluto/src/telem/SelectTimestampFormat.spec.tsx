// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimestampFormat } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Telem } from "@/telem";
import { mockBoundingClientRect } from "@/testutil/dom";

describe("SelectTimestampFormat", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  const onChange = vi.fn();
  const TestSelect = () => {
    const [value, setValue] = useState<TimestampFormat | undefined>(undefined);
    const handleChange = (key: TimestampFormat) => {
      setValue(key);
      onChange(key);
    };
    return <Telem.SelectTimestampFormat value={value} onChange={handleChange} />;
  };

  it("should render a selection trigger", () => {
    const c = render(<TestSelect />);
    expect(c.getByText("Select timestamp format")).toBeTruthy();
  });

  it("should open and show all format options", () => {
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select timestamp format"));
    expect(c.getByText("ISO 8601")).toBeTruthy();
    expect(c.getByText("ISO date")).toBeTruthy();
    expect(c.getByText("Time")).toBeTruthy();
    expect(c.getByText("Precise time")).toBeTruthy();
    expect(c.getByText("Date")).toBeTruthy();
    expect(c.getByText("Date + Time")).toBeTruthy();
    expect(c.getByText("Precise date")).toBeTruthy();
  });

  it("should call onChange when a format is selected", () => {
    const c = render(<TestSelect />);
    fireEvent.click(c.getByText("Select timestamp format"));
    fireEvent.click(c.getByText("Precise date"));
    expect(onChange).toHaveBeenCalledWith("preciseDate");
  });
});
