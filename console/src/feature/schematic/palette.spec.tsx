// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flux, type Pluto, Status, Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, testProjectKey } from "@/feature/schematic/testutil";
import { Layout } from "@/platform/layout";
import { Modals } from "@/platform/modals";
import { type Palette } from "@/platform/palette";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderHookWithConsole,
  waitForPlacedLayout,
} from "@/testutil";

interface CommandHarnessProps {
  Command: Palette.Command;
}

const CommandHarness = ({ Command }: CommandHarnessProps): ReactElement => (
  <Command
    key={Command.key}
    itemKey={Command.key}
    index={0}
    placeLayout={Layout.usePlacer()}
    confirm={Modals.useConfirm()}
    rename={Modals.useRename()}
    handleError={Status.useErrorHandler()}
    addStatus={Status.useAdder()}
    store={Session.useStore()}
    fluxStore={Flux.useStore<Pluto.FluxStore>()}
    client={Synnax.use()}
  />
);

describe("Schematic.COMMANDS", () => {
  it("is visible when the user may create schematics", async () => {
    const { result } = await renderHookWithConsole(
      () => Schematic.COMMANDS[0].useVisible?.(),
      { client },
    );
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("creates and places a schematic when selected", async () => {
    const project = await testProjectKey();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: project,
        },
      },
    });
    render(<CommandHarness Command={Schematic.COMMANDS[0]} />, { wrapper });
    fireEvent.click(await screen.findByText("Create a schematic"));
    const placedKey = await waitForPlacedLayout(store, "schematic");
    await waitFor(async () => {
      const created = await client.schematics.retrieve({ key: placedKey });
      expect(created.name).toBe("Schematic");
    });
  });
});
