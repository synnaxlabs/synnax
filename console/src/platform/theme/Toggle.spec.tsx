// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";
import { fireEvent, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Theme } from "@/platform/theme";
import { Session } from "@/session";
import { getIconButton, renderWithConsole } from "@/testutil";

// Wires the app's real theming provider so a press flips the rendered icon.
const ThemedToggle = (): ReactElement => (
  <Theming.Provider {...Session.Theme.useProviderProps()}>
    <Theme.Toggle />
  </Theming.Provider>
);
ThemedToggle.displayName = "ThemedToggle";

describe("Theme.Toggle", () => {
  it("should flip the theme and its icon on each press", async () => {
    const { container, store } = await renderWithConsole(<ThemedToggle />);

    fireEvent.click(getIconButton(container, "dark-mode"));
    await waitFor(() => {
      expect(store.getState()[Session.Theme.SLICE_NAME].mode).toEqual("dark");
      getIconButton(container, "light-mode");
    });

    fireEvent.click(getIconButton(container, "light-mode"));
    await waitFor(() => {
      expect(store.getState()[Session.Theme.SLICE_NAME].mode).toEqual("light");
      getIconButton(container, "dark-mode");
    });
  });
});
