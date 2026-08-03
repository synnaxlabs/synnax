// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimeZone } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Telem } from "@/telem";

describe("SelectTimeZone", () => {
  const Wrapper = ({
    initial = "UTC",
    onChange,
  }: {
    initial?: TimeZone;
    onChange?: (v: TimeZone) => void;
  }) => {
    const [value, setValue] = useState<TimeZone>(initial);
    return (
      <Telem.SelectTimeZone
        value={value}
        onChange={(v: TimeZone) => {
          setValue(v);
          onChange?.(v);
        }}
      />
    );
  };

  it("should render UTC and Local buttons", () => {
    const c = render(<Wrapper />);
    expect(c.getByText("UTC")).toBeTruthy();
    expect(c.getByText("Local")).toBeTruthy();
  });

  it("should mark the selected time zone as selected", () => {
    const c = render(<Wrapper initial="UTC" />);
    expect(c.getByText("UTC").closest("button")?.classList).toContain("pluto--selected");
    expect(c.getByText("Local").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
  });

  it("should call onChange with the new time zone when a button is clicked", () => {
    const onChange = vi.fn();
    const c = render(<Wrapper initial="UTC" onChange={onChange} />);
    fireEvent.click(c.getByText("Local"));
    expect(onChange).toHaveBeenCalledWith("local");
  });

  it("should move the selected state to the newly selected button", () => {
    const c = render(<Wrapper initial="UTC" />);
    fireEvent.click(c.getByText("Local"));
    expect(c.getByText("Local").closest("button")?.classList).toContain(
      "pluto--selected",
    );
    expect(c.getByText("UTC").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
  });
});
