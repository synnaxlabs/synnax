// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";
import { renderWithConsole } from "@/testutil";

describe("Export.ToolbarButton", () => {
  it("invokes onExport when clicked", async () => {
    const onExport = vi.fn();
    await renderWithConsole(<Export.ToolbarButton onExport={onExport} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onExport).toHaveBeenCalledTimes(1);
  });

  it("forwards the disabled prop, suppressing the click", async () => {
    const onExport = vi.fn();
    await renderWithConsole(<Export.ToolbarButton onExport={onExport} disabled />);
    fireEvent.click(screen.getByRole("button"));
    expect(onExport).not.toHaveBeenCalled();
  });
});
