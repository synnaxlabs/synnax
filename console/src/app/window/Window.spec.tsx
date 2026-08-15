// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/pluto";
import { createEvent, fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Window } from "@/app/window";
import { Session } from "@/session";
import { getBySelector, installPortalRoot, renderWithConsole } from "@/testutil";

// jsdom has no DragEvent, so the transfer is defined on the event testing-library falls
// back to. effectAllowed marks an OS file drag rather than an internal haul.
const fireFileDragOver = (target: HTMLElement): void => {
  const event = createEvent.dragOver(target);
  Object.defineProperty(event, "dataTransfer", { value: { effectAllowed: "all" } });
  fireEvent(target, event);
};

describe("app/window/Window", () => {
  installPortalRoot();

  // Dropping a file onto the mosaic imports it, but only while the haul carries FILE.
  // Nothing else starts that drag, so losing this makes every file drop a no-op.
  it("hauls a file when an OS file enters the window", async () => {
    const { store, container } = await renderWithConsole(
      <Haul.Provider {...Session.Haul.PROVIDER_PROPS}>
        <Window.Window />
      </Haul.Provider>,
    );
    fireFileDragOver(getBySelector<HTMLElement>(container, ".console-main"));
    await waitFor(() =>
      expect(store.getState()[Session.Haul.SLICE_NAME].state.items).toEqual([
        Haul.FILE,
      ]),
    );
  });
});
