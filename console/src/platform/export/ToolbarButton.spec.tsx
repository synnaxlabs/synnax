// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Export } from "@/platform/export";
import { renderWithConsole } from "@/testutil";

const getID = (): ontology.ID => ({ type: "log", key: "k" });

describe("Export.ToolbarButton", () => {
  it("resolves the id to export when clicked", async () => {
    const resolve = vi.fn(getID);
    await renderWithConsole(<Export.ToolbarButton getID={resolve} />);
    fireEvent.click(screen.getByRole("button"));
    expect(resolve).toHaveBeenCalledTimes(1);
  });

  it("forwards the disabled prop, suppressing the click", async () => {
    const resolve = vi.fn(getID);
    await renderWithConsole(<Export.ToolbarButton getID={resolve} disabled />);
    fireEvent.click(screen.getByRole("button"));
    expect(resolve).not.toHaveBeenCalled();
  });
});
