// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ChangeVariant } from "@/table/cells/ChangeVariant";

describe("ChangeVariant", () => {
  it("should list every variant when opened", () => {
    render(<ChangeVariant value="text" onChange={vi.fn()} />);
    fireEvent.click(screen.getByLabelText("Change cell type"));
    expect(screen.getByRole("option", { name: "Text" })).toBeDefined();
    expect(screen.getByRole("option", { name: "Value" })).toBeDefined();
  });

  it("should call onChange with the picked variant", () => {
    const onChange = vi.fn();
    render(<ChangeVariant value="text" onChange={onChange} />);
    fireEvent.click(screen.getByLabelText("Change cell type"));
    fireEvent.click(screen.getByText("Value"));
    expect(onChange).toHaveBeenCalledWith("value", expect.anything());
  });
});
