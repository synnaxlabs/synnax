// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Theme } from "@/feature/theme";
import { renderWithConsole } from "@/testutil";

describe("Theme.Settings", () => {
  it("should apply the color theme that the user selects", async () => {
    const { store } = await renderWithConsole(<Theme.Settings />);
    fireEvent.click(screen.getByText("Dark"));
    await waitFor(() => expect(store.getState().theme.mode).toBe("dark"));
  });
});
