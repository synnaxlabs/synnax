// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, testProjectKey } from "@/feature/schematic/testutil";
import { Layout } from "@/platform/layout";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import { createConsoleWrapper, waitForPlacedLayout } from "@/testutil";

const SelectableHarness = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const Selectable = Schematic.SELECTABLES[0];
  return (
    <Selectable
      layoutKey={layoutKey}
      rename={Modals.useRename()}
      onPlace={Layout.usePlacer()}
      handleError={Status.useErrorHandler()}
    />
  );
};

describe("Schematic.SELECTABLES", () => {
  it("creates a schematic in the active project when clicked", async () => {
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
    render(<SelectableHarness layoutKey={uuid.create()} />, { wrapper });
    fireEvent.click(await screen.findByText("Schematic"));
    const placedKey = await waitForPlacedLayout(store, "schematic");
    await waitFor(async () => {
      const created = await client.schematics.retrieve({ key: placedKey });
      expect(created.name).toBe("Schematic");
    });
    expect(
      Session.Schematic.selectSliceState(store.getState()).schematics[placedKey],
    ).toBeDefined();
  });
});
